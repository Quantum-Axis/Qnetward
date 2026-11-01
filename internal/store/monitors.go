package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrMonitorNotFound  = errors.New("monitor not found")
	ErrMonitorLimit     = fmt.Errorf("a workspace can have at most %d monitors", MaxMonitorsPerWorkspace)
	ErrMonitorDuplicate = errors.New("this target is already monitored")
)

// MaxMonitorsPerWorkspace caps scheduled checks while every plan is free.
const MaxMonitorsPerWorkspace = 20

// MonitorIntervals are the check intervals a monitor can use, in seconds.
var MonitorIntervals = []int{60, 300, 600, 1800, 3600}

// monitorRecentChecks is how many checks the list returns per monitor.
const monitorRecentChecks = 30

type Monitor struct {
	ID                 int64          `json:"id"`
	WorkspaceID        int64          `json:"-"`
	UserID             int64          `json:"-"`
	Name               string         `json:"name"`
	Protocol           string         `json:"protocol"`
	Target             string         `json:"target"`
	Port               *int           `json:"port,omitempty"`
	IntervalSeconds    int            `json:"interval_seconds"`
	NotifyEmail        string         `json:"notify_email"`
	Enabled            bool           `json:"enabled"`
	Status             string         `json:"status"`
	LastCheckedAt      *time.Time     `json:"last_checked_at,omitempty"`
	LastStatusChangeAt *time.Time     `json:"last_status_change_at,omitempty"`
	LastRttMs          *float64       `json:"last_rtt_ms,omitempty"`
	LastError          string         `json:"last_error,omitempty"`
	NextRunAt          time.Time      `json:"next_run_at"`
	CreatedAt          time.Time      `json:"created_at"`
	Uptime24h          *float64       `json:"uptime_24h,omitempty"`
	RecentChecks       []MonitorCheck `json:"recent_checks"`
}

type MonitorCheck struct {
	CheckedAt   time.Time `json:"checked_at"`
	OK          bool      `json:"ok"`
	RttMs       *float64  `json:"rtt_ms,omitempty"`
	LossPercent float64   `json:"loss_percent"`
	Error       string    `json:"error,omitempty"`
}

const monitorColumns = `id, workspace_id, COALESCE(user_id, 0), name, protocol, target, port,
	interval_seconds, notify_email, enabled, status, last_checked_at,
	last_status_change_at, last_rtt_ms, last_error, next_run_at, created_at`

// SaveMonitor creates a monitor, or updates one in the same workspace. A new
// or re-pointed monitor is checked right away.
func (s *Store) SaveMonitor(ctx context.Context, m Monitor) (Monitor, error) {
	if s == nil {
		return Monitor{}, errors.New("store is not available")
	}
	if err := validateMonitor(&m); err != nil {
		return Monitor{}, err
	}

	tx, err := s.beginWorkspaceTx(ctx, m.WorkspaceID)
	if err != nil {
		return Monitor{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var saved Monitor
	if m.ID == 0 {
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM monitors WHERE workspace_id = $1`, m.WorkspaceID).Scan(&count); err != nil {
			return Monitor{}, err
		}
		if count >= MaxMonitorsPerWorkspace {
			return Monitor{}, ErrMonitorLimit
		}
		err = scanMonitor(tx.QueryRowContext(ctx, `
			INSERT INTO monitors (workspace_id, user_id, name, protocol, target, port, interval_seconds, notify_email, enabled)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING `+monitorColumns,
			m.WorkspaceID, nullableID(m.UserID), m.Name, m.Protocol, m.Target, m.Port, m.IntervalSeconds, m.NotifyEmail, m.Enabled,
		), &saved)
	} else {
		// Pointing a monitor somewhere else discards the old target's status.
		err = scanMonitor(tx.QueryRowContext(ctx, `
			UPDATE monitors SET
				status = CASE WHEN protocol = $3 AND target = $4 AND port IS NOT DISTINCT FROM $5 THEN status ELSE 'pending' END,
				last_checked_at = CASE WHEN protocol = $3 AND target = $4 AND port IS NOT DISTINCT FROM $5 THEN last_checked_at END,
				last_status_change_at = CASE WHEN protocol = $3 AND target = $4 AND port IS NOT DISTINCT FROM $5 THEN last_status_change_at END,
				last_rtt_ms = CASE WHEN protocol = $3 AND target = $4 AND port IS NOT DISTINCT FROM $5 THEN last_rtt_ms END,
				last_error = CASE WHEN protocol = $3 AND target = $4 AND port IS NOT DISTINCT FROM $5 THEN last_error ELSE '' END,
				next_run_at = CASE
					WHEN protocol = $3 AND target = $4 AND port IS NOT DISTINCT FROM $5 AND (enabled OR NOT $9) THEN next_run_at
					ELSE NOW()
				END,
				name = $6,
				protocol = $3,
				target = $4,
				port = $5,
				interval_seconds = $7,
				notify_email = $8,
				enabled = $9
			WHERE id = $1 AND workspace_id = $2
			RETURNING `+monitorColumns,
			m.ID, m.WorkspaceID, m.Protocol, m.Target, m.Port, m.Name, m.IntervalSeconds, m.NotifyEmail, m.Enabled,
		), &saved)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return Monitor{}, ErrMonitorNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Monitor{}, ErrMonitorDuplicate
	}
	if err != nil {
		return Monitor{}, err
	}
	saved.RecentChecks = []MonitorCheck{}
	return saved, tx.Commit()
}

// ListMonitors returns the workspace's monitors with their uptime over the
// last 24 hours and their most recent checks, oldest first.
func (s *Store) ListMonitors(ctx context.Context, workspaceID int64) ([]Monitor, error) {
	if s == nil {
		return nil, errors.New("store is not available")
	}
	tx, err := s.beginWorkspaceTx(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, `SELECT `+monitorColumns+` FROM monitors WHERE workspace_id = $1 ORDER BY created_at, id`, workspaceID)
	if err != nil {
		return nil, err
	}
	monitors := make([]Monitor, 0, 8)
	index := map[int64]int{}
	for rows.Next() {
		var m Monitor
		if err := scanMonitor(rows, &m); err != nil {
			rows.Close()
			return nil, err
		}
		m.RecentChecks = []MonitorCheck{}
		index[m.ID] = len(monitors)
		monitors = append(monitors, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if len(monitors) == 0 {
		return monitors, nil
	}

	uptime, err := tx.QueryContext(ctx, `
		SELECT monitor_id, COUNT(*) FILTER (WHERE ok)::float8 / COUNT(*) * 100
		FROM monitor_checks
		WHERE workspace_id = $1 AND checked_at > NOW() - INTERVAL '24 hours'
		GROUP BY monitor_id
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	for uptime.Next() {
		var id int64
		var pct float64
		if err := uptime.Scan(&id, &pct); err != nil {
			uptime.Close()
			return nil, err
		}
		if i, ok := index[id]; ok {
			monitors[i].Uptime24h = &pct
		}
	}
	if err := uptime.Err(); err != nil {
		return nil, err
	}
	uptime.Close()

	checks, err := tx.QueryContext(ctx, `
		SELECT monitor_id, checked_at, ok, rtt_ms, loss_percent, error
		FROM (
			SELECT *, ROW_NUMBER() OVER (PARTITION BY monitor_id ORDER BY checked_at DESC) AS n
			FROM monitor_checks
			WHERE workspace_id = $1
		) recent
		WHERE n <= $2
		ORDER BY monitor_id, checked_at
	`, workspaceID, monitorRecentChecks)
	if err != nil {
		return nil, err
	}
	defer checks.Close()
	for checks.Next() {
		var id int64
		var c MonitorCheck
		var rtt sql.NullFloat64
		if err := checks.Scan(&id, &c.CheckedAt, &c.OK, &rtt, &c.LossPercent, &c.Error); err != nil {
			return nil, err
		}
		if rtt.Valid {
			v := rtt.Float64
			c.RttMs = &v
		}
		if i, ok := index[id]; ok {
			monitors[i].RecentChecks = append(monitors[i].RecentChecks, c)
		}
	}
	return monitors, checks.Err()
}

func (s *Store) DeleteMonitor(ctx context.Context, workspaceID, monitorID int64) error {
	if s == nil {
		return errors.New("store is not available")
	}
	tx, err := s.beginWorkspaceTx(ctx, workspaceID)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `DELETE FROM monitors WHERE id = $1 AND workspace_id = $2`, monitorID, workspaceID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrMonitorNotFound
	}
	return tx.Commit()
}

// ClaimDueMonitors picks up to limit enabled monitors whose check is due and
// moves their next run one interval ahead, so another instance won't take
// them. It works across workspaces and runs as the login role.
func (s *Store) ClaimDueMonitors(ctx context.Context, limit int) ([]Monitor, error) {
	if s == nil {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
		UPDATE monitors m
		SET next_run_at = NOW() + make_interval(secs => m.interval_seconds)
		WHERE m.id IN (
			SELECT id FROM monitors
			WHERE enabled AND next_run_at <= NOW()
			ORDER BY next_run_at
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING `+monitorColumns, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var due []Monitor
	for rows.Next() {
		var m Monitor
		if err := scanMonitor(rows, &m); err != nil {
			return nil, err
		}
		due = append(due, m)
	}
	return due, rows.Err()
}

// RecordMonitorCheck stores one check and moves the monitor to up or down.
// It returns the status before the check, so the caller can tell a change.
func (s *Store) RecordMonitorCheck(ctx context.Context, m Monitor, check MonitorCheck) (string, error) {
	if s == nil {
		return "", errors.New("store is not available")
	}
	tx, err := s.beginWorkspaceTx(ctx, m.WorkspaceID)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	var previous string
	err = tx.QueryRowContext(ctx, `SELECT status FROM monitors WHERE id = $1 FOR UPDATE`, m.ID).Scan(&previous)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrMonitorNotFound
	}
	if err != nil {
		return "", err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO monitor_checks (monitor_id, workspace_id, checked_at, ok, rtt_ms, loss_percent, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, m.ID, m.WorkspaceID, check.CheckedAt.UTC(), check.OK, check.RttMs, check.LossPercent, check.Error); err != nil {
		return "", err
	}

	next := "down"
	if check.OK {
		next = "up"
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE monitors SET
			status = $2,
			last_checked_at = $3,
			last_status_change_at = CASE WHEN status = $2 THEN last_status_change_at ELSE $3 END,
			last_rtt_ms = $4,
			last_error = $5
		WHERE id = $1
	`, m.ID, next, check.CheckedAt.UTC(), check.RttMs, check.Error); err != nil {
		return "", err
	}
	return previous, tx.Commit()
}

// PruneMonitorChecks deletes checks older than the cutoff, across workspaces.
func (s *Store) PruneMonitorChecks(ctx context.Context, before time.Time) error {
	if s == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM monitor_checks WHERE checked_at < $1`, before.UTC())
	return err
}

func validateMonitor(m *Monitor) error {
	if m.WorkspaceID <= 0 {
		return errors.New("workspace is required")
	}
	m.Protocol = strings.ToLower(strings.TrimSpace(m.Protocol))
	switch m.Protocol {
	case "ping":
		m.Port = nil
	case "tcpping":
		if m.Port == nil || *m.Port < 1 || *m.Port > 65535 {
			return errors.New("port must be between 1 and 65535")
		}
	default:
		return errors.New("protocol must be ping or tcpping")
	}
	m.Target = strings.TrimSpace(m.Target)
	if m.Target == "" {
		return errors.New("target is required")
	}
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		m.Name = m.Target
		if m.Port != nil {
			m.Name = fmt.Sprintf("%s:%d", m.Target, *m.Port)
		}
	}
	if len(m.Name) > 80 {
		return errors.New("name must be 80 characters or fewer")
	}
	validInterval := false
	for _, v := range MonitorIntervals {
		if m.IntervalSeconds == v {
			validInterval = true
			break
		}
	}
	if !validInterval {
		return fmt.Errorf("interval must be one of %v seconds", MonitorIntervals)
	}
	m.NotifyEmail = strings.TrimSpace(m.NotifyEmail)
	if m.NotifyEmail != "" {
		email, err := NormalizeEmail(m.NotifyEmail)
		if err != nil {
			return err
		}
		m.NotifyEmail = email
	}
	return nil
}

type monitorScanner interface {
	Scan(dest ...any) error
}

func scanMonitor(scanner monitorScanner, m *Monitor) error {
	var port sql.NullInt64
	var lastChecked, lastChange sql.NullTime
	var lastRtt sql.NullFloat64
	if err := scanner.Scan(
		&m.ID, &m.WorkspaceID, &m.UserID, &m.Name, &m.Protocol, &m.Target, &port,
		&m.IntervalSeconds, &m.NotifyEmail, &m.Enabled, &m.Status, &lastChecked,
		&lastChange, &lastRtt, &m.LastError, &m.NextRunAt, &m.CreatedAt,
	); err != nil {
		return err
	}
	m.Port = nil
	if port.Valid {
		v := int(port.Int64)
		m.Port = &v
	}
	m.LastCheckedAt = nil
	if lastChecked.Valid {
		t := lastChecked.Time
		m.LastCheckedAt = &t
	}
	m.LastStatusChangeAt = nil
	if lastChange.Valid {
		t := lastChange.Time
		m.LastStatusChangeAt = &t
	}
	m.LastRttMs = nil
	if lastRtt.Valid {
		v := lastRtt.Float64
		m.LastRttMs = &v
	}
	return nil
}
