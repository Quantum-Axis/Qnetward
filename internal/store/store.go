package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	DriverPostgres    = "postgres"
	schemaSetupLockID = int64(64442024091357)
	// queryTimeout bounds history reads that don't take a context.
	queryTimeout = 10 * time.Second
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Config struct {
	DSN              string
	BatchSize        int
	FlushInterval    time.Duration
	Retention        time.Duration
	CleanupInterval  time.Duration
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  time.Duration
	RequireTimescale bool
	TableName        string
}

type Store struct {
	db        *sql.DB
	tableName string
	insertCh  chan ResultRecord
	done      chan struct{}
	cfg       Config
	wg        sync.WaitGroup
	closeOnce sync.Once
	dropped   atomic.Uint64
	errors    atomic.Uint64
}

type ResultRecord struct {
	// WorkspaceID owns the sample; UserID records who ran the probe.
	WorkspaceID int64
	UserID      int64
	Ts          time.Time
	Protocol    string
	Target      string
	Addr        string
	Port        int
	Seq         int
	RttMs       *float64
	Error       string
}

type HistoryPoint struct {
	Ts    int64    `json:"ts"`
	Seq   int      `json:"seq,omitempty"`
	RttMs *float64 `json:"rtt_ms,omitempty"`
	Error string   `json:"error,omitempty"`
}

type RecentHistoryTarget struct {
	Protocol     string `json:"protocol"`
	Target       string `json:"target"`
	Port         int    `json:"port,omitempty"`
	LastSampleAt int64  `json:"last_sample_at"`
}

func Open(path string) (*Store, error) {
	return OpenWithConfig(path, Config{})
}

func OpenWithConfig(dsn string, cfg Config) (*Store, error) {
	if cfg.DSN == "" {
		cfg.DSN = dsn
	}
	return OpenConfigured(cfg)
}

func OpenConfigured(cfg Config) (*Store, error) {
	cfg = applyDefaults(cfg)
	tableName, err := normalizeIdentifier(cfg.TableName, "results")
	if err != nil {
		return nil, fmt.Errorf("invalid table name: %w", err)
	}

	db, err := openDB(cfg)
	if err != nil {
		return nil, err
	}

	if err := setupSchema(db, cfg, tableName); err != nil {
		_ = db.Close()
		return nil, err
	}

	chSize := cfg.BatchSize * 4
	if chSize < 512 {
		chSize = 512
	}

	s := &Store{
		db:        db,
		tableName: tableName,
		insertCh:  make(chan ResultRecord, chSize),
		done:      make(chan struct{}),
		cfg:       cfg,
	}
	s.wg.Add(1)
	go s.worker()

	return s, nil
}

func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	var err error
	s.closeOnce.Do(func() {
		close(s.done)
		s.wg.Wait()
		err = s.db.Close()
	})
	return err
}

func (s *Store) Driver() string {
	if s == nil {
		return ""
	}
	return DriverPostgres
}

func (s *Store) Ping(ctx context.Context) error {
	if s == nil {
		return nil
	}
	return s.db.PingContext(ctx)
}

func (s *Store) InsertAsync(rec ResultRecord) {
	if s == nil {
		return
	}
	select {
	case s.insertCh <- rec:
	case <-s.done:
	default:
		dropped := s.dropped.Add(1)
		if dropped == 1 || dropped%100 == 0 {
			log.Printf("store queue full, dropped %d result(s)", dropped)
		}
	}
}

func (s *Store) worker() {
	defer s.wg.Done()

	flushTicker := time.NewTicker(s.cfg.FlushInterval)
	defer flushTicker.Stop()

	var cleanupTicker *time.Ticker
	var cleanupCh <-chan time.Time
	if s.cfg.Retention > 0 {
		cleanupTicker = time.NewTicker(s.cfg.CleanupInterval)
		cleanupCh = cleanupTicker.C
		defer cleanupTicker.Stop()
	}

	batch := make([]ResultRecord, 0, s.cfg.BatchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := s.insertBatch(batch); err != nil {
			errors := s.errors.Add(1)
			log.Printf("store batch insert failed (%d): %v", errors, err)
		}
		batch = batch[:0]
	}

	cleanup := func() {
		if s.cfg.Retention <= 0 {
			return
		}
		cutoff := time.Now().Add(-s.cfg.Retention)
		if err := s.deleteOlderThan(cutoff); err != nil {
			errors := s.errors.Add(1)
			log.Printf("store cleanup failed (%d): %v", errors, err)
		}
	}

	for {
		select {
		case rec := <-s.insertCh:
			batch = append(batch, rec)
			if len(batch) >= s.cfg.BatchSize {
				flush()
			}
		case <-flushTicker.C:
			flush()
		case <-s.done:
			for {
				select {
				case rec := <-s.insertCh:
					batch = append(batch, rec)
				default:
					flush()
					return
				}
			}
		case <-cleanupCh:
			cleanup()
		}
	}
}

func (s *Store) insertBatch(batch []ResultRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(s.insertStatement())
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	for _, rec := range batch {
		args := []any{
			rec.WorkspaceID,
			rec.UserID,
			s.normalizeTimestamp(rec.Ts),
			rec.Protocol,
			rec.Target,
			rec.Addr,
			rec.Port,
			rec.Seq,
			rec.RttMs,
			rec.Error,
		}
		if _, err := stmt.Exec(args...); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			return err
		}
	}

	_ = stmt.Close()
	return tx.Commit()
}

// QueryHistory returns the workspace's most recent samples for one target,
// oldest first.
func (s *Store) QueryHistory(workspaceID int64, protocol, target string, port int, limit int) ([]HistoryPoint, error) {
	if s == nil {
		return nil, nil
	}
	if workspaceID <= 0 {
		return nil, fmt.Errorf("workspace id is required")
	}
	if limit <= 0 {
		limit = 120
	}
	if limit > 1000 {
		limit = 1000
	}

	args := []any{workspaceID, protocol, target}
	query := s.queryHistorySQL(port > 0)
	if port > 0 {
		args = append(args, port)
	}
	args = append(args, limit)

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	tx, err := s.beginWorkspaceTx(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]HistoryPoint, 0, limit)
	for rows.Next() {
		var ts int64
		var seq sql.NullInt64
		var rtt sql.NullFloat64
		var errStr sql.NullString
		if err := rows.Scan(&ts, &seq, &rtt, &errStr); err != nil {
			return nil, err
		}

		p := HistoryPoint{Ts: ts}
		if seq.Valid {
			p.Seq = int(seq.Int64)
		}
		if rtt.Valid {
			v := rtt.Float64
			p.RttMs = &v
		}
		if errStr.Valid {
			p.Error = errStr.String
		}
		points = append(points, p)
	}

	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}

	return points, rows.Err()
}

// QueryRecentHistoryTargets lists the targets the workspace probed most recently.
func (s *Store) QueryRecentHistoryTargets(workspaceID int64, limit int) ([]RecentHistoryTarget, error) {
	if s == nil {
		return nil, nil
	}
	if workspaceID <= 0 {
		return nil, fmt.Errorf("workspace id is required")
	}
	if limit <= 0 {
		limit = 12
	}
	if limit > 50 {
		limit = 50
	}

	query := fmt.Sprintf(`SELECT protocol, target, COALESCE(port, 0), CAST(EXTRACT(EPOCH FROM MAX(ts)) * 1000 AS BIGINT)
		FROM %s
		WHERE workspace_id = $1
		GROUP BY protocol, target, port
		ORDER BY MAX(ts) DESC
		LIMIT $2`, s.tableRef())

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	tx, err := s.beginWorkspaceTx(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	targets := make([]RecentHistoryTarget, 0, limit)
	for rows.Next() {
		var item RecentHistoryTarget
		if err := rows.Scan(&item.Protocol, &item.Target, &item.Port, &item.LastSampleAt); err != nil {
			return nil, err
		}
		targets = append(targets, item)
	}

	return targets, rows.Err()
}

func (s *Store) deleteOlderThan(cutoff time.Time) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE ts < $1", s.tableRef())
	args := []any{cutoff.UTC()}
	_, err := s.db.Exec(query, args...)
	return err
}

func (s *Store) insertStatement() string {
	return fmt.Sprintf(`INSERT INTO %s (workspace_id, user_id, ts, protocol, target, addr, port, seq, rtt_ms, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);`, s.tableRef())
}

func (s *Store) queryHistorySQL(withPort bool) string {
	query := fmt.Sprintf(`SELECT CAST(EXTRACT(EPOCH FROM ts) * 1000 AS BIGINT), seq, rtt_ms, error FROM %s
		WHERE workspace_id = $1 AND protocol = $2 AND target = $3`, s.tableRef())
	if withPort {
		query += " AND port = $4"
		query += " ORDER BY ts DESC LIMIT $5"
	} else {
		query += " ORDER BY ts DESC LIMIT $4"
	}
	return query
}

func (s *Store) normalizeTimestamp(ts time.Time) any {
	return ts.UTC()
}

func openDB(cfg Config) (*sql.DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("postgres dsn is required")
	}
	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, err
	}
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	return db, nil
}

// setupSchema brings the database up to date: versioned migrations for the
// shared schema, then the configured results table.
func setupSchema(db *sql.DB, cfg Config, tableName string) error {
	ctx := context.Background()
	return withSchemaLock(ctx, db, func(conn *sql.Conn) error {
		ran, err := runMigrations(ctx, conn)
		for _, m := range ran {
			log.Printf("store: applied migration %04d_%s", m.Version, m.Name)
		}
		if err != nil {
			return err
		}
		if cfg.RequireTimescale {
			if _, err := conn.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS timescaledb;`); err != nil {
				return fmt.Errorf("enable timescaledb extension: %w", err)
			}
		}
		return ensureResultsTable(ctx, conn, tableName, cfg.RequireTimescale)
	})
}

// ensureResultsTable creates the probe results table under its configured name.
// It stays outside the versioned migrations because the name varies per
// deployment and per test run.
func ensureResultsTable(ctx context.Context, conn *sql.Conn, tableName string, timescale bool) error {
	table := quoteIdentifier(tableName)
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id BIGINT GENERATED BY DEFAULT AS IDENTITY,
			workspace_id BIGINT,
			user_id BIGINT NOT NULL,
			ts TIMESTAMPTZ NOT NULL,
			protocol TEXT NOT NULL,
			target TEXT NOT NULL,
			addr TEXT,
			port INTEGER,
			seq INTEGER,
			rtt_ms DOUBLE PRECISION,
			error TEXT,
			PRIMARY KEY (ts, id)
		);
	`, table)); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS user_id BIGINT;`, table)); err != nil {
		return err
	}
	if err := scopeResultsToWorkspaces(ctx, conn, tableName); err != nil {
		return err
	}
	if err := enableWorkspaceRLS(ctx, conn, table, "workspace_isolation"); err != nil {
		return fmt.Errorf("enable results row-level security: %w", err)
	}
	if timescale {
		if _, err := conn.ExecContext(ctx, fmt.Sprintf(`
			SELECT create_hypertable('%s', 'ts', if_not_exists => TRUE, migrate_data => TRUE);
		`, tableName)); err != nil {
			return fmt.Errorf("create timescaledb hypertable: %w", err)
		}
	}
	return nil
}

// scopeResultsToWorkspaces adds the workspace column to older results tables,
// assigns rows recorded before it existed to the prober's personal workspace,
// and replaces the per-user lookup index with a per-workspace one.
func scopeResultsToWorkspaces(ctx context.Context, conn *sql.Conn, tableName string) error {
	table := quoteIdentifier(tableName)
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s ADD COLUMN IF NOT EXISTS workspace_id BIGINT;`, table)); err != nil {
		return err
	}
	// The partial index keeps the backfill check cheap once every row is scoped.
	unscoped := quoteIdentifier(fmt.Sprintf("idx_%s_unscoped", tableName))
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s ON %s(user_id) WHERE workspace_id IS NULL;
	`, unscoped, table)); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s r SET workspace_id = w.id
		FROM workspaces w
		WHERE r.workspace_id IS NULL AND w.owner_user_id = r.user_id;
	`, table)); err != nil {
		return fmt.Errorf("assign results to workspaces: %w", err)
	}
	lookup := quoteIdentifier(fmt.Sprintf("idx_%s_workspace_lookup", tableName))
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS %s
		ON %s(workspace_id, protocol, target, port, ts DESC);
	`, lookup, table)); err != nil {
		return err
	}
	legacy := quoteIdentifier(fmt.Sprintf("idx_%s_user_lookup", tableName))
	_, err := conn.ExecContext(ctx, fmt.Sprintf(`DROP INDEX IF EXISTS %s;`, legacy))
	return err
}

func applyDefaults(cfg Config) Config {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 1 * time.Second
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 1 * time.Hour
	}
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = 10
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 5
	}
	if cfg.ConnMaxLifetime <= 0 {
		cfg.ConnMaxLifetime = 30 * time.Minute
	}
	if cfg.TableName == "" {
		cfg.TableName = "results"
	}
	return cfg
}

func normalizeIdentifier(value string, fallback string) (string, error) {
	if value == "" {
		value = fallback
	}
	if !identifierPattern.MatchString(value) {
		return "", fmt.Errorf("must match %s", identifierPattern.String())
	}
	return value, nil
}

func quoteIdentifier(value string) string {
	return `"` + value + `"`
}

func (s *Store) tableRef() string {
	return quoteIdentifier(s.tableName)
}
