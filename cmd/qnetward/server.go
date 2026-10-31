package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"qnetward/internal/platform/cache"
	"qnetward/internal/platform/mailer"
	"qnetward/internal/platform/netguard"
	"qnetward/internal/platform/objectstore"
	"qnetward/internal/store"
)

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	listen := fs.String("listen", envString("QNETWARD_LISTEN", "127.0.0.1:8080"), "listen address")
	dbDSN := fs.String("db-dsn", envString("QNETWARD_DB_DSN", ""), "postgres/timescaledb DSN")
	dbBatch := fs.Int("db-batch", envInt("QNETWARD_DB_BATCH", 100), "database batch insert size")
	dbFlush := fs.Duration("db-flush", envDuration("QNETWARD_DB_FLUSH", 1*time.Second), "database flush interval")
	dbRetention := fs.Duration("db-retention", envDuration("QNETWARD_DB_RETENTION", 168*time.Hour), "retention duration (0 to disable)")
	dbCleanup := fs.Duration("db-cleanup", envDuration("QNETWARD_DB_CLEANUP", 1*time.Hour), "retention cleanup interval")
	dbMaxOpen := fs.Int("db-max-open", envInt("QNETWARD_DB_MAX_OPEN", 20), "database max open connections")
	dbMaxIdle := fs.Int("db-max-idle", envInt("QNETWARD_DB_MAX_IDLE", 10), "database max idle connections")
	dbConnMaxLifetime := fs.Duration("db-conn-max-lifetime", envDuration("QNETWARD_DB_CONN_MAX_LIFETIME", 30*time.Minute), "database connection max lifetime")
	timescale := fs.Bool("timescale", envBool("QNETWARD_TIMESCALE", false), "require TimescaleDB extension when using postgres")
	redisAddr := fs.String("redis-addr", envString("QNETWARD_REDIS_ADDR", ""), "Redis address for history cache")
	redisPassword := fs.String("redis-password", envString("QNETWARD_REDIS_PASSWORD", ""), "Redis password")
	redisDB := fs.Int("redis-db", envInt("QNETWARD_REDIS_DB", 0), "Redis logical DB")
	redisPrefix := fs.String("redis-prefix", envString("QNETWARD_REDIS_PREFIX", "qnetward"), "Redis key prefix")
	redisTTL := fs.Duration("redis-cache-ttl", envDuration("QNETWARD_REDIS_CACHE_TTL", 3*time.Second), "Redis history cache TTL")
	s3Endpoint := fs.String("s3-endpoint", envString("QNETWARD_S3_ENDPOINT", ""), "S3-compatible endpoint")
	s3AccessKey := fs.String("s3-access-key", envString("QNETWARD_S3_ACCESS_KEY", ""), "S3 access key")
	s3SecretKey := fs.String("s3-secret-key", envString("QNETWARD_S3_SECRET_KEY", ""), "S3 secret key")
	s3Bucket := fs.String("s3-bucket", envString("QNETWARD_S3_BUCKET", ""), "S3 bucket for exports and artifacts")
	s3Region := fs.String("s3-region", envString("QNETWARD_S3_REGION", "us-east-1"), "S3 region")
	s3Prefix := fs.String("s3-prefix", envString("QNETWARD_S3_PREFIX", "qnetward"), "S3 object prefix")
	s3UseSSL := fs.Bool("s3-ssl", envBool("QNETWARD_S3_SSL", false), "use TLS for S3-compatible endpoint")
	resendAPIKey := fs.String("resend-api-key", envString("QNETWARD_RESEND_API_KEY", ""), "Resend API key for transactional email")
	emailFrom := fs.String("email-from", envString("QNETWARD_EMAIL_FROM", ""), "from address for transactional emails")
	emailReplyTo := fs.String("email-reply-to", envString("QNETWARD_EMAIL_REPLY_TO", ""), "reply-to address for transactional emails")
	publicBaseURL := fs.String("public-base-url", envString("QNETWARD_PUBLIC_BASE_URL", ""), "public base URL used in emails, for example https://qnetward.example.com")
	allowedOriginsRaw := fs.String("allowed-origins", envString("QNETWARD_ALLOWED_ORIGINS", ""), "comma-separated additional allowed browser origins, for example http://127.0.0.1:3000")
	demoEnabled := fs.Bool("demo-enabled", envBool("QNETWARD_DEMO_ENABLED", false), "enable shared demo login")
	demoName := fs.String("demo-name", envString("QNETWARD_DEMO_NAME", "Contributor Demo"), "display name for demo access")
	demoEmail := fs.String("demo-email", envString("QNETWARD_DEMO_EMAIL", "demo@qnetward.local"), "display email for demo access")
	allowPrivateTargets := fs.Bool("allow-private-targets", envBool("QNETWARD_ALLOW_PRIVATE_TARGETS", false), "allow probes to loopback, private, and link-local addresses (self-hosted LAN monitoring); cloud metadata endpoints stay blocked")
	adminEmailsRaw := fs.String("admin-emails", envString("QNETWARD_ADMIN_EMAILS", ""), "comma-separated platform admin emails")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: qnetward serve [flags]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(normalizeArgs(fs, args)); err != nil {
		os.Exit(2)
	}

	defer func() {
		appStore = nil
		appCache = nil
		appObjectStore = nil
		appMailer = nil
		appPublicBaseURL = ""
		appAllowedOrigins = nil
		appAdminEmails = nil
	}()

	allowedOrigins, err := parseAllowedOrigins(*allowedOriginsRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	appAllowedOrigins = allowedOrigins
	adminEmails, err := parseAdminEmails(*adminEmailsRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	appAdminEmails = adminEmails

	st, err := store.OpenConfigured(store.Config{
		DSN:              *dbDSN,
		BatchSize:        *dbBatch,
		FlushInterval:    *dbFlush,
		Retention:        *dbRetention,
		CleanupInterval:  *dbCleanup,
		MaxOpenConns:     *dbMaxOpen,
		MaxIdleConns:     *dbMaxIdle,
		ConnMaxLifetime:  *dbConnMaxLifetime,
		RequireTimescale: *timescale,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	appStore = st
	defer func() {
		_ = st.Close()
	}()

	rc, err := cache.Open(cache.Config{
		Addr:     *redisAddr,
		Password: *redisPassword,
		DB:       *redisDB,
		Prefix:   *redisPrefix,
		TTL:      *redisTTL,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	appCache = rc
	if rc != nil {
		defer func() {
			_ = rc.Close()
		}()
	}

	objStore, err := objectstore.Open(objectstore.Config{
		Endpoint:  *s3Endpoint,
		AccessKey: *s3AccessKey,
		SecretKey: *s3SecretKey,
		Region:    *s3Region,
		Bucket:    *s3Bucket,
		Prefix:    *s3Prefix,
		UseSSL:    *s3UseSSL,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	appObjectStore = objStore
	if objStore != nil {
		defer func() {
			_ = objStore.Close()
		}()
	}

	emailClient, err := mailer.Open(mailer.Config{
		APIKey:  *resendAPIKey,
		From:    *emailFrom,
		ReplyTo: *emailReplyTo,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	appMailer = emailClient
	appPublicBaseURL = strings.TrimRight(strings.TrimSpace(*publicBaseURL), "/")
	appDemoEnabled = *demoEnabled
	appDemoName = strings.TrimSpace(*demoName)
	appTargetPolicy = netguard.Policy{AllowPrivate: *allowPrivateTargets}
	appDemoEmail = strings.TrimSpace(*demoEmail)
	appDemoCreatedAt = time.Now().UTC()
	if appDemoEnabled {
		if appDemoName == "" {
			appDemoName = "Contributor Demo"
		}
		if appDemoEmail == "" {
			appDemoEmail = "demo@qnetward.local"
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/api/healthz", handleHealthz)
	mux.HandleFunc("/api/auth/session", handleAuthSession)
	mux.HandleFunc("/api/auth/account-context", handleAuthAccountContext)
	mux.HandleFunc("/api/auth/demo-login", handleAuthDemoLogin)
	mux.HandleFunc("/api/auth/signup", handleAuthSignup)
	mux.HandleFunc("/api/auth/login", handleAuthLogin)
	mux.HandleFunc("/api/auth/logout", handleAuthLogout)
	mux.HandleFunc("/api/auth/profile", handleAuthProfile)
	mux.HandleFunc("/api/auth/verify-email", handleAuthVerifyEmail)
	mux.HandleFunc("/api/auth/resend-verification", handleAuthResendVerification)
	mux.HandleFunc("/api/auth/forgot-password", handleAuthForgotPassword)
	mux.HandleFunc("/api/auth/reset-password", handleAuthResetPassword)
	mux.HandleFunc("/api/workspaces", handleWorkspaces)
	mux.HandleFunc("/api/workspace/team", handleWorkspaceTeam)
	mux.HandleFunc("/api/workspace/invitations", handleWorkspaceInvitations)
	mux.HandleFunc("/api/admin/overview", handleAdminOverview)
	mux.HandleFunc("/api/admin/users", handleAdminUsers)
	mux.HandleFunc("/api/admin/workspaces", handleAdminWorkspaces)
	mux.HandleFunc("/api/admin/system", handleAdminSystem)
	mux.HandleFunc("/api/ping", handlePing)
	mux.HandleFunc("/api/tcpping", handleTCPPing)
	mux.HandleFunc("/api/portscan", handlePortScan)
	mux.HandleFunc("/api/dnslookup", handleDNSLookup)
	mux.HandleFunc("/api/history", handleHistory)
	mux.HandleFunc("/api/history/recent-targets", handleRecentHistoryTargets)
	mux.HandleFunc("/api/export/history", handleHistoryExport)
	mux.HandleFunc("/api/alerts/rules", handleAlertRules)
	mux.HandleFunc("/ws", handleWS)
	mux.Handle("/", newBackendFallbackHandler())

	srv := &http.Server{
		Addr:              *listen,
		Handler:           withSecurityHeaders(mux),
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	fmt.Printf("Qnetward UI listening on http://%s\n", *listen)
	fmt.Printf("Database: postgres/timescale\n")
	fmt.Printf("DB batch=%d flush=%s retention=%s max-open=%d max-idle=%d timescale=%t\n", *dbBatch, dbFlush.String(), dbRetention.String(), *dbMaxOpen, *dbMaxIdle, *timescale)
	if rc != nil {
		fmt.Printf("Redis cache: %s db=%d ttl=%s\n", *redisAddr, *redisDB, redisTTL.String())
	}
	if objStore != nil {
		fmt.Printf("Object storage: %s bucket=%s prefix=%s\n", *s3Endpoint, *s3Bucket, *s3Prefix)
	}
	if emailClient != nil {
		fmt.Printf("Email delivery: Resend from=%s\n", *emailFrom)
	}

	// Handle graceful shutdown on SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Shutdown error: %v\n", err)
		}
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newBackendFallbackHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":      true,
				"service": "qnetward-api",
				"ui":      "serve the Next.js frontend separately or use the frontend-primary Docker stack",
			})
			return
		}
		writeJSON(w, http.StatusNotFound, apiError{Error: "not found"})
	})
}

var appStore *store.Store

var appCache *cache.Cache

var appObjectStore *objectstore.Client

var appMailer *mailer.Client

var appPublicBaseURL string

var appAllowedOrigins map[string]struct{}

var appAdminEmails map[string]struct{}

var appDemoEnabled bool

var appDemoName string

var appDemoEmail string

var appDemoCreatedAt time.Time

// appTargetPolicy decides which addresses API-initiated probes may reach.
// The CLI does not use it: it runs on the operator's own machine.
var appTargetPolicy netguard.Policy

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}

	type healthDep struct {
		Enabled bool   `json:"enabled"`
		Healthy bool   `json:"healthy"`
		Driver  string `json:"driver,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	type healthResponse struct {
		OK          bool      `json:"ok"`
		Time        time.Time `json:"time"`
		Database    healthDep `json:"database"`
		Redis       healthDep `json:"redis"`
		ObjectStore healthDep `json:"object_storage"`
	}

	resp := healthResponse{
		OK:   true,
		Time: time.Now().UTC(),
		Database: healthDep{
			Enabled: appStore != nil,
		},
		Redis: healthDep{
			Enabled: appCache != nil,
		},
		ObjectStore: healthDep{
			Enabled: appObjectStore != nil,
		},
	}

	if appStore != nil {
		resp.Database.Driver = appStore.Driver()
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		err := appStore.Ping(ctx)
		cancel()
		resp.Database.Healthy = err == nil
		if err != nil {
			resp.Database.Error = err.Error()
			resp.OK = false
		}
	}
	if appCache != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		err := appCache.Ping(ctx)
		cancel()
		resp.Redis.Healthy = err == nil
		if err != nil {
			resp.Redis.Error = err.Error()
			resp.OK = false
		}
	}
	if appObjectStore != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		err := appObjectStore.Ping(ctx)
		cancel()
		resp.ObjectStore.Healthy = err == nil
		if err != nil {
			resp.ObjectStore.Error = err.Error()
			resp.OK = false
		}
	}

	status := http.StatusOK
	if !resp.OK {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, resp)
}
