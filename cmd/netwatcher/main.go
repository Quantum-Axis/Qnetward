package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"netwatcher/internal/cache"
	"netwatcher/internal/mailer"
	"netwatcher/internal/objectstore"
	"netwatcher/internal/store"
	"netwatcher/internal/telemetry"
)

const (
	defaultProbeCount    = 4
	defaultProbeInterval = time.Second
	defaultProbeTimeout  = 2 * time.Second
	defaultPingSize      = 32
	defaultTCPPort       = 80
	defaultScanPorts     = "22,80,443,8080"
	defaultScanWorkers   = 64
	maxProbeCount        = 100
	minProbeInterval     = 100 * time.Millisecond
	maxProbeInterval     = 60 * time.Second
	minProbeTimeout      = 100 * time.Millisecond
	maxProbeTimeout      = 60 * time.Second
	minPingSize          = 8
	maxPingSize          = 65535
	minScanConcurrency   = 1
	maxScanConcurrency   = 256
	maxScanPorts         = 4096
	wsWriteTimeout       = 10 * time.Second
	wsReadTimeout        = 60 * time.Second
	maxJSONBodyBytes     = 8 << 10
	maxWSMessageBytes    = 4 << 10
	maxHostLength        = 255
	maxHistoryLimit      = 1000
	maxConcurrentProbes  = 8
	authCookieName       = "netwatcher_session"
	defaultSessionTTL    = 30 * 24 * time.Hour
	defaultResetTTL      = 30 * time.Minute
	defaultVerifyCodeTTL = 10 * time.Minute
	verifyResendCooldown = 60 * time.Second
	resetIssueCooldown   = 60 * time.Second
	rateLimitSweepEvery  = 5 * time.Minute
	rateLimitBucketTTL   = 30 * time.Minute
	maxRateLimitBuckets  = 8192
	minPasswordLength    = 8
	maxPasswordLength    = 72
	maxNameLength        = 80
)

type boolFlag interface {
	IsBoolFlag() bool
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "ping":
		runPing(os.Args[2:])
	case "tcpping":
		runTCPPing(os.Args[2:])
	case "portscan":
		runPortScan(os.Args[2:])
	case "dns":
		runDNSLookup(os.Args[2:])
	case "serve":
		runServe(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "NetWatcher - Networking toolkit")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  netwatcher ping [flags] <host>")
	fmt.Fprintln(os.Stderr, "  netwatcher tcpping [flags] <host>[:port]")
	fmt.Fprintln(os.Stderr, "  netwatcher portscan [flags] <host>")
	fmt.Fprintln(os.Stderr, "  netwatcher dns [flags] <host-or-ip>")
	fmt.Fprintln(os.Stderr, "  netwatcher serve [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  ping     ICMP echo (requires root or CAP_NET_RAW)")
	fmt.Fprintln(os.Stderr, "  tcpping  TCP connect latency")
	fmt.Fprintln(os.Stderr, "  portscan TCP connect port scan")
	fmt.Fprintln(os.Stderr, "  dns      DNS lookup / reverse DNS")
	fmt.Fprintln(os.Stderr, "  serve    Web UI + REST API")
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	listen := fs.String("listen", envString("NETWATCHER_LISTEN", "127.0.0.1:8080"), "listen address")
	dbDSN := fs.String("db-dsn", envString("NETWATCHER_DB_DSN", ""), "postgres/timescaledb DSN")
	dbBatch := fs.Int("db-batch", envInt("NETWATCHER_DB_BATCH", 100), "database batch insert size")
	dbFlush := fs.Duration("db-flush", envDuration("NETWATCHER_DB_FLUSH", 1*time.Second), "database flush interval")
	dbRetention := fs.Duration("db-retention", envDuration("NETWATCHER_DB_RETENTION", 168*time.Hour), "retention duration (0 to disable)")
	dbCleanup := fs.Duration("db-cleanup", envDuration("NETWATCHER_DB_CLEANUP", 1*time.Hour), "retention cleanup interval")
	dbMaxOpen := fs.Int("db-max-open", envInt("NETWATCHER_DB_MAX_OPEN", 20), "database max open connections")
	dbMaxIdle := fs.Int("db-max-idle", envInt("NETWATCHER_DB_MAX_IDLE", 10), "database max idle connections")
	dbConnMaxLifetime := fs.Duration("db-conn-max-lifetime", envDuration("NETWATCHER_DB_CONN_MAX_LIFETIME", 30*time.Minute), "database connection max lifetime")
	timescale := fs.Bool("timescale", envBool("NETWATCHER_TIMESCALE", false), "require TimescaleDB extension when using postgres")
	redisAddr := fs.String("redis-addr", envString("NETWATCHER_REDIS_ADDR", ""), "Redis address for history cache")
	redisPassword := fs.String("redis-password", envString("NETWATCHER_REDIS_PASSWORD", ""), "Redis password")
	redisDB := fs.Int("redis-db", envInt("NETWATCHER_REDIS_DB", 0), "Redis logical DB")
	redisPrefix := fs.String("redis-prefix", envString("NETWATCHER_REDIS_PREFIX", "netwatcher"), "Redis key prefix")
	redisTTL := fs.Duration("redis-cache-ttl", envDuration("NETWATCHER_REDIS_CACHE_TTL", 3*time.Second), "Redis history cache TTL")
	s3Endpoint := fs.String("s3-endpoint", envString("NETWATCHER_S3_ENDPOINT", ""), "S3-compatible endpoint")
	s3AccessKey := fs.String("s3-access-key", envString("NETWATCHER_S3_ACCESS_KEY", ""), "S3 access key")
	s3SecretKey := fs.String("s3-secret-key", envString("NETWATCHER_S3_SECRET_KEY", ""), "S3 secret key")
	s3Bucket := fs.String("s3-bucket", envString("NETWATCHER_S3_BUCKET", ""), "S3 bucket for exports and artifacts")
	s3Region := fs.String("s3-region", envString("NETWATCHER_S3_REGION", "us-east-1"), "S3 region")
	s3Prefix := fs.String("s3-prefix", envString("NETWATCHER_S3_PREFIX", "netwatcher"), "S3 object prefix")
	s3UseSSL := fs.Bool("s3-ssl", envBool("NETWATCHER_S3_SSL", false), "use TLS for S3-compatible endpoint")
	resendAPIKey := fs.String("resend-api-key", envString("NETWATCHER_RESEND_API_KEY", ""), "Resend API key for transactional email")
	emailFrom := fs.String("email-from", envString("NETWATCHER_EMAIL_FROM", ""), "from address for transactional emails")
	emailReplyTo := fs.String("email-reply-to", envString("NETWATCHER_EMAIL_REPLY_TO", ""), "reply-to address for transactional emails")
	publicBaseURL := fs.String("public-base-url", envString("NETWATCHER_PUBLIC_BASE_URL", ""), "public base URL used in emails, for example https://netwatcher.example.com")
	allowedOriginsRaw := fs.String("allowed-origins", envString("NETWATCHER_ALLOWED_ORIGINS", ""), "comma-separated additional allowed browser origins, for example http://127.0.0.1:3000")
	demoEnabled := fs.Bool("demo-enabled", envBool("NETWATCHER_DEMO_ENABLED", false), "enable shared demo login")
	demoName := fs.String("demo-name", envString("NETWATCHER_DEMO_NAME", "Contributor Demo"), "display name for demo access")
	demoEmail := fs.String("demo-email", envString("NETWATCHER_DEMO_EMAIL", "demo@netwatcher.local"), "display email for demo access")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: netwatcher serve [flags]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(normalizeArgs(fs, args)); err != nil {
		os.Exit(2)
	}

	telemetryShutdown, err := telemetry.Setup(context.Background(), telemetry.ConfigFromEnv())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetryShutdown(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "telemetry shutdown error: %v\n", err)
		}
	}()

	defer func() {
		appStore = nil
		appCache = nil
		appObjectStore = nil
		appMailer = nil
		appPublicBaseURL = ""
		appAllowedOrigins = nil
	}()

	allowedOrigins, err := parseAllowedOrigins(*allowedOriginsRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	appAllowedOrigins = allowedOrigins

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
	appDemoEmail = strings.TrimSpace(*demoEmail)
	appDemoCreatedAt = time.Now().UTC()
	if appDemoEnabled {
		if appDemoName == "" {
			appDemoName = "Contributor Demo"
		}
		if appDemoEmail == "" {
			appDemoEmail = "demo@netwatcher.local"
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/api/healthz", handleHealthz)
	mux.HandleFunc("/api/auth/session", handleAuthSession)
	mux.HandleFunc("/api/auth/demo-login", handleAuthDemoLogin)
	mux.HandleFunc("/api/auth/signup", handleAuthSignup)
	mux.HandleFunc("/api/auth/login", handleAuthLogin)
	mux.HandleFunc("/api/auth/logout", handleAuthLogout)
	mux.HandleFunc("/api/auth/verify-email", handleAuthVerifyEmail)
	mux.HandleFunc("/api/auth/resend-verification", handleAuthResendVerification)
	mux.HandleFunc("/api/auth/forgot-password", handleAuthForgotPassword)
	mux.HandleFunc("/api/auth/reset-password", handleAuthResetPassword)
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

	instrumentedMux := otelhttp.NewHandler(mux, "netwatcher.http",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)

	srv := &http.Server{
		Addr:              *listen,
		Handler:           withSecurityHeaders(instrumentedMux),
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	fmt.Printf("NetWatcher UI listening on http://%s\n", *listen)
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
				"service": "netwatcher-api",
				"ui":      "serve the Next.js frontend separately or use the frontend-primary Docker stack",
			})
			return
		}
		writeJSON(w, http.StatusNotFound, apiError{Error: "not found"})
	})
}

func normalizeArgs(fs *flag.FlagSet, args []string) []string {
	if len(args) == 0 {
		return args
	}

	takesValue := make(map[string]bool)
	fs.VisitAll(func(f *flag.Flag) {
		_, isBool := f.Value.(boolFlag)
		takesValue[f.Name] = !isBool
	})

	flags := make([]string, 0, len(args))
	pos := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			name := strings.TrimLeft(a, "-")
			if idx := strings.Index(name, "="); idx >= 0 {
				name = name[:idx]
			}
			if takesValue[name] && !strings.Contains(a, "=") && i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		pos = append(pos, a)
	}

	return append(flags, pos...)
}

func envString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

var appStore *store.Store

var appCache *cache.Cache

var appObjectStore *objectstore.Client

var appMailer *mailer.Client

var appPublicBaseURL string

var appAllowedOrigins map[string]struct{}

var appDemoEnabled bool

var appDemoName string

var appDemoEmail string

var appDemoCreatedAt time.Time

var probeLimiter = make(chan struct{}, maxConcurrentProbes)

var rejectedProbeRequests atomic.Uint64
