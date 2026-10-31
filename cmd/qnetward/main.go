package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"

	"qnetward/internal/engine/dnslookup"
	"qnetward/internal/engine/ping"
	"qnetward/internal/engine/portscan"
	"qnetward/internal/engine/report"
	"qnetward/internal/engine/tcpping"
	"qnetward/internal/platform/cache"
	"qnetward/internal/platform/mailer"
	"qnetward/internal/platform/objectstore"
	"qnetward/internal/store"
)

const (
	defaultProbeCount    = 5
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
	authCookieName       = "qnetward_session"
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
	fmt.Fprintln(os.Stderr, "Qnetward - Networking toolkit")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  qnetward ping [flags] <host>")
	fmt.Fprintln(os.Stderr, "  qnetward tcpping [flags] <host>[:port]")
	fmt.Fprintln(os.Stderr, "  qnetward portscan [flags] <host>")
	fmt.Fprintln(os.Stderr, "  qnetward dns [flags] <host-or-ip>")
	fmt.Fprintln(os.Stderr, "  qnetward serve [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  ping     ICMP echo (requires root or CAP_NET_RAW)")
	fmt.Fprintln(os.Stderr, "  tcpping  TCP connect latency")
	fmt.Fprintln(os.Stderr, "  portscan TCP connect port scan")
	fmt.Fprintln(os.Stderr, "  dns      DNS lookup / reverse DNS")
	fmt.Fprintln(os.Stderr, "  serve    Web UI + REST API")
}

func runPing(args []string) {
	fs := flag.NewFlagSet("ping", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	count := fs.Int("count", defaultProbeCount, "number of probes")
	interval := fs.Duration("interval", time.Second, "interval between probes")
	timeout := fs.Duration("timeout", 2*time.Second, "per-probe timeout")
	size := fs.Int("size", 32, "ICMP payload size in bytes")
	ipv6 := fs.Bool("ipv6", false, "force IPv6")
	jsonOut := fs.Bool("json", false, "JSON output")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: qnetward ping [flags] <host>")
		fs.PrintDefaults()
	}

	if err := fs.Parse(normalizeArgs(fs, args)); err != nil {
		os.Exit(2)
	}

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}

	host := fs.Arg(0)
	cfg, err := normalizePingConfig(*count, *interval, *timeout, *size, *ipv6, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	rep, err := ping.Run(host, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *jsonOut {
		printJSON(rep)
		return
	}
	printPing(rep)
}

func runTCPPing(args []string) {
	fs := flag.NewFlagSet("tcpping", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	count := fs.Int("count", defaultProbeCount, "number of probes")
	interval := fs.Duration("interval", time.Second, "interval between probes")
	timeout := fs.Duration("timeout", 2*time.Second, "per-probe timeout")
	port := fs.Int("port", 80, "TCP port")
	jsonOut := fs.Bool("json", false, "JSON output")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: qnetward tcpping [flags] <host>[:port]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(normalizeArgs(fs, args)); err != nil {
		os.Exit(2)
	}

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}

	host := fs.Arg(0)
	portFromFlag := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			portFromFlag = true
		}
	})

	if !portFromFlag {
		parsedHost, parsedPort, ok := splitHostPort(host)
		if ok {
			host = parsedHost
			*port = parsedPort
		}
	}

	host, resolvedPort, cfg, err := normalizeTCPPingRequest(host, *port, *count, *interval, *timeout, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	rep, err := tcpping.Run(host, resolvedPort, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *jsonOut {
		printJSON(rep)
		return
	}
	printTCPPing(rep)
}

func runPortScan(args []string) {
	fs := flag.NewFlagSet("portscan", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	portsSpec := fs.String("ports", defaultScanPorts, "comma-separated ports and ranges (for example 22,80,443,8000-8100)")
	timeout := fs.Duration("timeout", defaultProbeTimeout, "per-port timeout")
	concurrency := fs.Int("concurrency", defaultScanWorkers, "maximum parallel connection attempts")
	jsonOut := fs.Bool("json", false, "JSON output")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: qnetward portscan [flags] <host>")
		fs.PrintDefaults()
	}

	if err := fs.Parse(normalizeArgs(fs, args)); err != nil {
		os.Exit(2)
	}

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}

	host, ports, cfg, err := normalizePortScanRequest(fs.Arg(0), *portsSpec, *timeout, *concurrency)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	rep, err := portscan.Run(host, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *jsonOut {
		printPortScanJSON(rep)
		return
	}
	printPortScan(rep, ports)
}

func runDNSLookup(args []string) {
	fs := flag.NewFlagSet("dns", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	timeout := fs.Duration("timeout", dnslookup.DefaultTimeout, "per-query timeout")
	recordType := fs.String("type", "all", "record type: all, a, aaaa, cname, ns, mx, txt, dmarc, soa, srv, caa, dnskey, ds, rrsig, ptr")
	resolver := fs.String("resolver", "auto", "resolver: auto, cloudflare, google, quad9, or host[:port]")
	jsonOut := fs.Bool("json", false, "JSON output")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: qnetward dns [flags] <host-or-ip>")
		fs.PrintDefaults()
	}

	if err := fs.Parse(normalizeArgs(fs, args)); err != nil {
		os.Exit(2)
	}

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}

	target, err := normalizeHost(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	normalizedTimeout, err := normalizeDNSLookupTimeout(*timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	rep, err := dnslookup.Run(target, dnslookup.Config{
		Timeout:    normalizedTimeout,
		RecordType: *recordType,
		Resolver:   *resolver,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *jsonOut {
		printDNSLookupJSON(rep)
		return
	}
	printDNSLookup(rep)
}

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

func splitHostPort(target string) (string, int, bool) {
	if !strings.Contains(target, ":") {
		return "", 0, false
	}
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		return "", 0, false
	}
	port, err := parsePort(portStr)
	if err != nil {
		return "", 0, false
	}
	return host, port, true
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

func normalizePingConfig(count int, interval, timeout time.Duration, size int, forceIPv6, continuous bool) (ping.Config, error) {
	normalizedCount, err := normalizeProbeCountForMode(count, continuous)
	if err != nil {
		return ping.Config{}, err
	}
	normalizedInterval, err := normalizeProbeInterval(interval)
	if err != nil {
		return ping.Config{}, err
	}
	normalizedTimeout, err := normalizeProbeTimeout(timeout)
	if err != nil {
		return ping.Config{}, err
	}
	normalizedSize, err := normalizePingSize(size)
	if err != nil {
		return ping.Config{}, err
	}

	return ping.Config{
		Count:      normalizedCount,
		Interval:   normalizedInterval,
		Timeout:    normalizedTimeout,
		Size:       normalizedSize,
		ForceIPv6:  forceIPv6,
		Continuous: continuous,
	}, nil
}

func normalizeTCPPingRequest(host string, port, count int, interval, timeout time.Duration, continuous bool) (string, int, tcpping.Config, error) {
	var err error
	host, err = normalizeHost(host)
	if err != nil {
		return "", 0, tcpping.Config{}, err
	}
	if port == 0 {
		port = defaultTCPPort
	}
	if port < 1 || port > 65535 {
		return "", 0, tcpping.Config{}, fmt.Errorf("port must be 1-65535")
	}

	normalizedCount, err := normalizeProbeCountForMode(count, continuous)
	if err != nil {
		return "", 0, tcpping.Config{}, err
	}
	normalizedInterval, err := normalizeProbeInterval(interval)
	if err != nil {
		return "", 0, tcpping.Config{}, err
	}
	normalizedTimeout, err := normalizeProbeTimeout(timeout)
	if err != nil {
		return "", 0, tcpping.Config{}, err
	}

	return host, port, tcpping.Config{
		Count:      normalizedCount,
		Interval:   normalizedInterval,
		Timeout:    normalizedTimeout,
		Continuous: continuous,
	}, nil
}

func normalizePortScanRequest(host, portsSpec string, timeout time.Duration, concurrency int) (string, []int, portscan.Config, error) {
	var err error
	host, err = normalizeHost(host)
	if err != nil {
		return "", nil, portscan.Config{}, err
	}

	ports, err := portscan.ParsePorts(portsSpec)
	if err != nil {
		return "", nil, portscan.Config{}, err
	}
	if len(ports) > maxScanPorts {
		return "", nil, portscan.Config{}, fmt.Errorf("scan is limited to %d ports per request", maxScanPorts)
	}

	normalizedTimeout, err := normalizeProbeTimeout(timeout)
	if err != nil {
		return "", nil, portscan.Config{}, err
	}

	if concurrency == 0 {
		concurrency = defaultScanWorkers
	}
	if concurrency < minScanConcurrency || concurrency > maxScanConcurrency {
		return "", nil, portscan.Config{}, fmt.Errorf("concurrency must be %d-%d", minScanConcurrency, maxScanConcurrency)
	}
	if concurrency > len(ports) {
		concurrency = len(ports)
	}

	return host, ports, portscan.Config{
		Ports:       ports,
		Timeout:     normalizedTimeout,
		Concurrency: concurrency,
	}, nil
}

func normalizeDNSLookupTimeout(timeout time.Duration) (time.Duration, error) {
	normalizedTimeout, err := normalizeProbeTimeout(timeout)
	if err != nil {
		return 0, err
	}
	return normalizedTimeout, nil
}

func normalizeHost(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("host is required")
	}
	if len(host) > maxHostLength {
		return "", fmt.Errorf("host is too long")
	}
	for _, b := range []byte(host) {
		if b < 32 || b == 127 {
			return "", fmt.Errorf("host contains invalid characters")
		}
	}
	return host, nil
}

func normalizeProbeCount(count int) (int, error) {
	if count == 0 {
		count = defaultProbeCount
	}
	if count < 1 || count > maxProbeCount {
		return 0, fmt.Errorf("count must be 1-%d", maxProbeCount)
	}
	return count, nil
}

func normalizeProbeCountForMode(count int, continuous bool) (int, error) {
	if !continuous {
		return normalizeProbeCount(count)
	}
	if count < 0 {
		return 0, fmt.Errorf("count cannot be negative")
	}
	if count == 0 || count > maxProbeCount {
		return defaultProbeCount, nil
	}
	return count, nil
}

func normalizeProbeInterval(interval time.Duration) (time.Duration, error) {
	if interval <= 0 {
		interval = defaultProbeInterval
	}
	if interval < minProbeInterval || interval > maxProbeInterval {
		return 0, fmt.Errorf("interval must be %s-%s", minProbeInterval, maxProbeInterval)
	}
	return interval, nil
}

func normalizeProbeTimeout(timeout time.Duration) (time.Duration, error) {
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}
	if timeout < minProbeTimeout || timeout > maxProbeTimeout {
		return 0, fmt.Errorf("timeout must be %s-%s", minProbeTimeout, maxProbeTimeout)
	}
	return timeout, nil
}

func normalizePingSize(size int) (int, error) {
	if size == 0 {
		size = defaultPingSize
	}
	if size < minPingSize || size > maxPingSize {
		return 0, fmt.Errorf("size must be %d-%d bytes", minPingSize, maxPingSize)
	}
	return size, nil
}

func parsePort(portStr string) (int, error) {
	var port int
	_, err := fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return 0, err
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port out of range")
	}
	return port, nil
}

func parseHistoryQuery(values url.Values) (string, string, int, int, error) {
	reqType := strings.ToLower(strings.TrimSpace(values.Get("type")))
	host := strings.TrimSpace(values.Get("host"))
	portStr := strings.TrimSpace(values.Get("port"))
	limitStr := strings.TrimSpace(values.Get("limit"))

	if reqType != "ping" && reqType != "tcpping" {
		return "", "", 0, 0, fmt.Errorf("type must be ping or tcpping")
	}
	if host == "" {
		return "", "", 0, 0, fmt.Errorf("host is required")
	}

	port := 0
	if reqType == "tcpping" {
		if parsedHost, parsedPort, ok := splitHostPort(host); ok {
			host = parsedHost
			if port == 0 {
				port = parsedPort
			}
		}
	}
	if normalizedHost, err := normalizeHost(host); err != nil {
		return "", "", 0, 0, err
	} else {
		host = normalizedHost
	}

	if portStr != "" {
		p, err := parsePort(portStr)
		if err != nil {
			return "", "", 0, 0, fmt.Errorf("invalid port")
		}
		port = p
	}

	limit := 120
	if limitStr != "" {
		var parsed int
		if _, err := fmt.Sscanf(limitStr, "%d", &parsed); err != nil || parsed < 1 || parsed > maxHistoryLimit {
			return "", "", 0, 0, fmt.Errorf("limit must be 1-%d", maxHistoryLimit)
		}
		limit = parsed
	}

	return reqType, host, port, limit, nil
}

func parseRecentHistoryTargetsLimit(values url.Values) (int, error) {
	limitStr := strings.TrimSpace(values.Get("limit"))
	if limitStr == "" {
		return 12, nil
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		return 0, fmt.Errorf("limit must be 1-50")
	}
	return limit, nil
}

func parseHistoryExportRequest(req historyExportRequest) (string, string, int, int, string, string, error) {
	values := url.Values{}
	values.Set("type", req.Type)
	values.Set("host", req.Host)
	if req.Port > 0 {
		values.Set("port", strconv.Itoa(req.Port))
	}
	if req.Limit > 0 {
		values.Set("limit", strconv.Itoa(req.Limit))
	}
	reqType, host, port, limit, err := parseHistoryQuery(values)
	if err != nil {
		return "", "", 0, 0, "", "", err
	}

	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "csv" {
		return "", "", 0, 0, "", "", fmt.Errorf("format must be json or csv")
	}

	destination := strings.ToLower(strings.TrimSpace(req.Destination))
	if destination == "" {
		destination = "download"
	}
	if destination != "download" && destination != "s3" {
		return "", "", 0, 0, "", "", fmt.Errorf("destination must be download or s3")
	}

	return reqType, host, port, limit, format, destination, nil
}

func historyCacheKey(userID int64, reqType, host string, port, limit int) string {
	return fmt.Sprintf("history:%d:%s:%s:%d:%d", userID, reqType, host, port, limit)
}

func exportFileName(reqType, host string, port int, format string) string {
	baseHost := strings.NewReplacer(":", "_", "/", "_", "\\", "_", " ", "_").Replace(host)
	if port > 0 {
		baseHost = fmt.Sprintf("%s_%d", baseHost, port)
	}
	return fmt.Sprintf("%s_%s.%s", reqType, baseHost, format)
}

func buildExportObjectKey(userID int64, reqType, host string, port int, format string) string {
	return fmt.Sprintf("exports/%d/%s/%d/%s", userID, reqType, time.Now().UTC().UnixNano(), exportFileName(reqType, host, port, format))
}

func historyCSVRows(points []store.HistoryPoint) [][]string {
	rows := make([][]string, 0, len(points)+1)
	rows = append(rows, []string{"timestamp_ms", "sequence", "rtt_ms", "error"})
	for _, point := range points {
		rtt := ""
		if point.RttMs != nil {
			rtt = fmt.Sprintf("%.3f", *point.RttMs)
		}
		rows = append(rows, []string{
			strconv.FormatInt(point.Ts, 10),
			strconv.Itoa(point.Seq),
			rtt,
			point.Error,
		})
	}
	return rows
}

// lookupEnv reads key and, for QNETWARD_* keys, falls back to the legacy
// NETWATCHER_* name so deployments configured before the rename keep working.
func lookupEnv(key string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	if suffix, ok := strings.CutPrefix(key, "QNETWARD_"); ok {
		return strings.TrimSpace(os.Getenv("NETWATCHER_" + suffix))
	}
	return ""
}

func envString(key, fallback string) string {
	if value := lookupEnv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := lookupEnv(key)
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
	value := lookupEnv(key)
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
	value := lookupEnv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func printPing(rep report.Report) {
	fmt.Printf("PING %s (%s)\n", rep.Target, rep.Addr)
	for _, r := range rep.Results {
		if r.Error != "" {
			fmt.Printf("seq=%d %s\n", r.Seq, r.Error)
			continue
		}
		addr := r.Addr
		if addr == "" {
			addr = rep.Addr
		}
		fmt.Printf("reply from %s: seq=%d time=%s\n", addr, r.Seq, formatDuration(r.RTT))
	}
	printSummary(rep)
}

func printTCPPing(rep report.Report) {
	fmt.Printf("TCP PING %s (%s)\n", rep.Target, rep.Addr)
	for _, r := range rep.Results {
		if r.Error != "" {
			fmt.Printf("seq=%d %s\n", r.Seq, r.Error)
			continue
		}
		fmt.Printf("seq=%d time=%s\n", r.Seq, formatDuration(r.RTT))
	}
	printSummary(rep)
}

func printPortScan(rep portscan.Report, ports []int) {
	fmt.Printf("TCP PORT SCAN %s", rep.Target)
	if rep.Addr != "" {
		fmt.Printf(" (%s)", rep.Addr)
	}
	fmt.Println()

	for _, result := range rep.Results {
		if result.State == "open" {
			fmt.Printf("port=%d state=open time=%s", result.Port, formatDuration(result.RTT))
		} else {
			fmt.Printf("port=%d state=%s", result.Port, result.State)
		}
		if result.Addr != "" {
			fmt.Printf(" addr=%s", result.Addr)
		}
		if result.Error != "" && result.State != "open" {
			fmt.Printf(" error=%s", result.Error)
		}
		fmt.Println()
	}

	fmt.Printf("--- %s tcp-portscan statistics ---\n", rep.Target)
	fmt.Printf("%d ports scanned, %d open, %d closed, %d timeout, completed in %s\n",
		rep.Summary.Scanned,
		rep.Summary.Open,
		rep.Summary.Closed,
		rep.Summary.Timeout,
		rep.Summary.Duration.Round(time.Millisecond),
	)
	if len(ports) > 0 {
		fmt.Printf("%d requested ports\n", len(ports))
	}
}

func printSummary(rep report.Report) {
	fmt.Printf("--- %s %s statistics ---\n", rep.Target, rep.Protocol)
	fmt.Printf("%d probes, %d replies, %.1f%% loss\n", rep.Summary.Sent, rep.Summary.Recv, rep.Summary.Loss)
	if rep.Summary.Recv > 0 {
		fmt.Printf("rtt min/avg/max/median/stddev = %s/%s/%s/%s/%s\n",
			formatDuration(rep.Summary.Min),
			formatDuration(rep.Summary.Avg),
			formatDuration(rep.Summary.Max),
			formatDuration(rep.Summary.Median),
			formatDuration(rep.Summary.StdDev),
		)
		fmt.Printf("jitter = %s\n", formatDuration(rep.Summary.Jitter))
	}
	if rep.Summary.LossStreak > 0 {
		fmt.Printf("max consecutive loss = %d\n", rep.Summary.LossStreak)
	}
	if rep.Summary.Duration > 0 {
		fmt.Printf("duration = %s\n", formatDuration(rep.Summary.Duration))
	}
}

type jsonResult struct {
	Seq   int      `json:"seq,omitempty"`
	Port  int      `json:"port,omitempty"`
	State string   `json:"state,omitempty"`
	RTT   string   `json:"rtt,omitempty"`
	RTTMs *float64 `json:"rtt_ms,omitempty"`
	Error string   `json:"error,omitempty"`
	Addr  string   `json:"addr,omitempty"`
}

type jsonSummary struct {
	Sent       int     `json:"sent"`
	Recv       int     `json:"recv"`
	Loss       float64 `json:"loss"`
	Min        string  `json:"min,omitempty"`
	Avg        string  `json:"avg,omitempty"`
	Max        string  `json:"max,omitempty"`
	Median     string  `json:"median,omitempty"`
	StdDev     string  `json:"stddev,omitempty"`
	Jitter     string  `json:"jitter,omitempty"`
	LossStreak int     `json:"loss_streak,omitempty"`
	Scanned    int     `json:"scanned,omitempty"`
	Open       int     `json:"open,omitempty"`
	Closed     int     `json:"closed,omitempty"`
	Timeout    int     `json:"timeout,omitempty"`
	Duration   string  `json:"duration,omitempty"`
}

type jsonReport struct {
	Protocol string       `json:"protocol"`
	Target   string       `json:"target"`
	Addr     string       `json:"addr"`
	Port     int          `json:"port,omitempty"`
	Results  []jsonResult `json:"results"`
	Summary  jsonSummary  `json:"summary"`
}

func printJSON(rep report.Report) {
	jr := toJSONReport(rep)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(jr)
}

func printPortScanJSON(rep portscan.Report) {
	jr := toJSONPortScanReport(rep)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(jr)
}

func printDNSLookup(rep dnslookup.Report) {
	fmt.Printf("DNS LOOKUP %s\n", rep.Target)
	fmt.Printf("kind=%s duration=%.1fms\n", rep.Kind, rep.Summary.DurationMs)

	printDNSSection := func(label string, section dnslookup.StringSection) {
		if len(section.Values) == 0 && section.Error == "" {
			return
		}
		fmt.Printf("%s:\n", label)
		if len(section.Values) > 0 {
			for _, value := range section.Values {
				fmt.Printf("  - %s\n", value)
			}
		}
		if section.Error != "" {
			fmt.Printf("  ! %s\n", section.Error)
		}
	}

	printDNSSection("A", rep.A)
	printDNSSection("AAAA", rep.AAAA)
	printDNSSection("CNAME", rep.CNAME)
	printDNSSection("NS", rep.NS)
	if len(rep.MX.Records) > 0 || rep.MX.Error != "" {
		fmt.Println("MX:")
		for _, record := range rep.MX.Records {
			fmt.Printf("  - %d %s\n", record.Pref, record.Host)
		}
		if rep.MX.Error != "" {
			fmt.Printf("  ! %s\n", rep.MX.Error)
		}
	}
	printDNSSection("TXT", rep.TXT)
	printDNSSection("DMARC", rep.DMARC)
	if rep.SOA.Record != nil || rep.SOA.Error != "" {
		fmt.Println("SOA:")
		if rep.SOA.Record != nil {
			fmt.Printf("  - ns=%s mbox=%s serial=%d refresh=%d retry=%d expire=%d minttl=%d\n",
				rep.SOA.Record.NS,
				rep.SOA.Record.MBox,
				rep.SOA.Record.Serial,
				rep.SOA.Record.Refresh,
				rep.SOA.Record.Retry,
				rep.SOA.Record.Expire,
				rep.SOA.Record.MinTTL,
			)
		}
		if rep.SOA.Error != "" {
			fmt.Printf("  ! %s\n", rep.SOA.Error)
		}
	}
	if len(rep.SRV.Records) > 0 || rep.SRV.Error != "" {
		fmt.Println("SRV:")
		for _, record := range rep.SRV.Records {
			fmt.Printf("  - %d %d %d %s\n", record.Priority, record.Weight, record.Port, record.Target)
		}
		if rep.SRV.Error != "" {
			fmt.Printf("  ! %s\n", rep.SRV.Error)
		}
	}
	if len(rep.CAA.Records) > 0 || rep.CAA.Error != "" {
		fmt.Println("CAA:")
		for _, record := range rep.CAA.Records {
			fmt.Printf("  - flag=%d tag=%s value=%s\n", record.Flag, record.Tag, record.Value)
		}
		if rep.CAA.Error != "" {
			fmt.Printf("  ! %s\n", rep.CAA.Error)
		}
	}
	printDNSSection("DNSKEY", rep.DNSKEY)
	printDNSSection("DS", rep.DS)
	printDNSSection("RRSIG", rep.RRSIG)
	if len(rep.PTR) > 0 {
		fmt.Println("PTR:")
		for _, record := range rep.PTR {
			if len(record.Names) > 0 {
				fmt.Printf("  %s\n", record.Address)
				for _, name := range record.Names {
					fmt.Printf("    - %s\n", name)
				}
				continue
			}
			fmt.Printf("  %s ! %s\n", record.Address, record.Error)
		}
	}
	fmt.Printf("analysis: mail_ready=%t spf=%t dmarc=%t mx=%t caa=%t dnssec=%t\n",
		rep.Analysis.MailReady,
		rep.Analysis.SPFPresent,
		rep.Analysis.DMARCPresent,
		rep.Analysis.MXPresent,
		rep.Analysis.CAAPresent,
		rep.Analysis.DNSSECPresent,
	)
}

func printDNSLookupJSON(rep dnslookup.Report) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(rep)
}

func formatDuration(d time.Duration) string {
	ms := float64(d) / float64(time.Millisecond)
	return fmt.Sprintf("%.3fms", ms)
}

type apiError struct {
	Error string `json:"error"`
}

type authRequest struct {
	Name            string `json:"name,omitempty"`
	FirstName       string `json:"first_name,omitempty"`
	LastName        string `json:"last_name,omitempty"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password,omitempty"`
}

type authForgotPasswordRequest struct {
	Email string `json:"email"`
}

type authProfileUpdateRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type workspaceInvitationRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type workspaceMemberUpdateRequest struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

type workspaceMemberRemoveRequest struct {
	UserID       int64 `json:"user_id"`
	InvitationID int64 `json:"invitation_id,omitempty"`
}

type workspaceInvitationActionRequest struct {
	InvitationID int64  `json:"invitation_id"`
	Action       string `json:"action"`
}

type authResetPasswordRequest struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password,omitempty"`
}

type authVerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type alertRuleRequest struct {
	ID                   int64   `json:"id,omitempty"`
	Name                 string  `json:"name,omitempty"`
	Protocol             string  `json:"protocol"`
	Target               string  `json:"target"`
	Port                 int     `json:"port,omitempty"`
	RecipientEmail       string  `json:"recipient_email,omitempty"`
	LatencyThresholdMs   float64 `json:"latency_threshold_ms,omitempty"`
	LossThresholdPercent float64 `json:"loss_threshold_percent,omitempty"`
	ConsecutiveBreaches  int     `json:"consecutive_breaches,omitempty"`
	CooldownMinutes      int     `json:"cooldown_minutes,omitempty"`
	NotifyOnRecovery     *bool   `json:"notify_on_recovery,omitempty"`
	Enabled              *bool   `json:"enabled,omitempty"`
}

type authUser struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type authWorkspace struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Plan          string    `json:"plan"`
	BillingStatus string    `json:"billing_status"`
	Role          string    `json:"role"`
	MemberCount   int       `json:"member_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type authWorkspaceMember struct {
	UserID    int64     `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
	IsCurrent bool      `json:"is_current"`
}

type authWorkspaceInvitation struct {
	ID            int64     `json:"id"`
	WorkspaceID   int64     `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name,omitempty"`
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	Status        string    `json:"status"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
	InviterName   string    `json:"inviter_name"`
}

type authAdminUser struct {
	ID                  int64      `json:"id"`
	Name                string     `json:"name"`
	Email               string     `json:"email"`
	EmailVerifiedAt     *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	OwnedWorkspaceCount int        `json:"owned_workspace_count"`
	WorkspaceCount      int        `json:"workspace_count"`
	ActiveSessionCount  int        `json:"active_session_count"`
	PlatformRole        string     `json:"platform_role"`
}

type authAdminWorkspace struct {
	ID                     int64     `json:"id"`
	Name                   string    `json:"name"`
	Plan                   string    `json:"plan"`
	BillingStatus          string    `json:"billing_status"`
	CreatedAt              time.Time `json:"created_at"`
	OwnerUserID            int64     `json:"owner_user_id"`
	OwnerName              string    `json:"owner_name"`
	OwnerEmail             string    `json:"owner_email"`
	MemberCount            int       `json:"member_count"`
	PendingInvitationCount int       `json:"pending_invitation_count"`
	ActiveSessionCount     int       `json:"active_session_count"`
}

type authAdminSystemRuntime struct {
	DatabaseDriver string `json:"database_driver,omitempty"`
	RedisEnabled   bool   `json:"redis_enabled"`
	ObjectStorage  bool   `json:"object_storage_enabled"`
	EmailDelivery  bool   `json:"email_delivery_enabled"`
}

type authAdminSystemFeatures struct {
	PublicBaseURL    string `json:"public_base_url,omitempty"`
	DemoEnabled      bool   `json:"demo_enabled"`
	SessionTTLHours  int    `json:"session_ttl_hours"`
	ConfiguredAdmins int    `json:"configured_admins"`
}

type authAdminSystemResponse struct {
	Overview store.AdminOverview     `json:"overview"`
	Runtime  authAdminSystemRuntime  `json:"runtime"`
	Features authAdminSystemFeatures `json:"features"`
}

type authSessionResponse struct {
	Authenticated     bool      `json:"authenticated"`
	User              *authUser `json:"user,omitempty"`
	Mode              string    `json:"mode,omitempty"`
	ReadOnly          bool      `json:"read_only,omitempty"`
	PlatformRole      string    `json:"platform_role,omitempty"`
	ActiveWorkspaceID int64     `json:"active_workspace_id,omitempty"`
}

type authMessageResponse struct {
	OK                   bool      `json:"ok"`
	Message              string    `json:"message"`
	PreviewURL           string    `json:"preview_url,omitempty"`
	PreviewCode          string    `json:"preview_code,omitempty"`
	VerificationRequired bool      `json:"verification_required,omitempty"`
	Authenticated        bool      `json:"authenticated,omitempty"`
	User                 *authUser `json:"user,omitempty"`
	Email                string    `json:"email,omitempty"`
	Mode                 string    `json:"mode,omitempty"`
	ReadOnly             bool      `json:"read_only,omitempty"`
}

type pingRequest struct {
	Host       string `json:"host"`
	Count      int    `json:"count"`
	IntervalMs int    `json:"interval_ms"`
	TimeoutMs  int    `json:"timeout_ms"`
	Size       int    `json:"size"`
	IPv6       bool   `json:"ipv6"`
	Continuous bool   `json:"continuous,omitempty"`
}

type tcpPingRequest struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Count      int    `json:"count"`
	IntervalMs int    `json:"interval_ms"`
	TimeoutMs  int    `json:"timeout_ms"`
	Continuous bool   `json:"continuous,omitempty"`
}

type portScanRequest struct {
	Host        string `json:"host"`
	Ports       string `json:"ports"`
	TimeoutMs   int    `json:"timeout_ms"`
	Concurrency int    `json:"concurrency"`
}

type dnsLookupRequest struct {
	Target     string `json:"target"`
	TimeoutMs  int    `json:"timeout_ms"`
	RecordType string `json:"record_type,omitempty"`
	Resolver   string `json:"resolver,omitempty"`
}

type historyExportRequest struct {
	Type        string `json:"type"`
	Host        string `json:"host"`
	Port        int    `json:"port,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Format      string `json:"format,omitempty"`
	Destination string `json:"destination,omitempty"`
}

type wsRequest struct {
	Type        string `json:"type"`
	Host        string `json:"host"`
	Port        int    `json:"port,omitempty"`
	Ports       string `json:"ports,omitempty"`
	Count       int    `json:"count"`
	IntervalMs  int    `json:"interval_ms"`
	TimeoutMs   int    `json:"timeout_ms"`
	Size        int    `json:"size"`
	IPv6        bool   `json:"ipv6"`
	Continuous  bool   `json:"continuous,omitempty"`
	Concurrency int    `json:"concurrency,omitempty"`
}

type wsEvent struct {
	Type     string       `json:"type"`
	Protocol string       `json:"protocol,omitempty"`
	Target   string       `json:"target,omitempty"`
	Addr     string       `json:"addr,omitempty"`
	Port     int          `json:"port,omitempty"`
	Ts       int64        `json:"ts,omitempty"`
	Result   *jsonResult  `json:"result,omitempty"`
	Summary  *jsonSummary `json:"summary,omitempty"`
	Error    string       `json:"error,omitempty"`
}

type wsWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type wsSession struct {
	writer *wsWriter

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func (w *wsWriter) WriteEvent(evt wsEvent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err != nil {
		return err
	}
	return w.conn.WriteJSON(evt)
}

func (s *wsSession) start(user store.User, req wsRequest) bool {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	s.mu.Unlock()

	go func() {
		defer close(done)
		err := runWSRequest(ctx, s.writer, user, req)
		if err != nil && !errors.Is(err, context.Canceled) {
			_ = s.writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
		}
		s.clear(done)
	}()
	return true
}

func (s *wsSession) stop() bool {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (s *wsSession) clear(done chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done != done {
		return
	}
	s.cancel = nil
	s.done = nil
}

func (s *wsSession) close() {
	s.stop()
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return isAllowedWSOrigin(r, origin)
	},
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
var probeLimiter = make(chan struct{}, maxConcurrentProbes)
var rejectedProbeRequests atomic.Uint64
var authRateLimiter = newRateLimiter()

type rateLimitConfig struct {
	Limit  int
	Window time.Duration
}

type rateLimitBucket struct {
	hits     []time.Time
	lastSeen time.Time
}

type rateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*rateLimitBucket
	lastSweep time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		buckets:   make(map[string]*rateLimitBucket),
		lastSweep: time.Now(),
	}
}

func handleAuthSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if appStore != nil {
		user, _, activeWorkspaceID, err := currentSessionState(r)
		if err == nil {
			mode := "authenticated"
			if isDemoUser(user) {
				mode = "demo"
			}
			writeJSON(w, http.StatusOK, authSessionResponse{
				Authenticated:     true,
				User:              toAuthUser(user),
				Mode:              mode,
				PlatformRole:      platformRoleForUser(user),
				ActiveWorkspaceID: activeWorkspaceID,
			})
			return
		}
		if !errors.Is(err, store.ErrSessionNotFound) && !errors.Is(err, http.ErrNoCookie) {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, authSessionResponse{Authenticated: false})
}

func handleAuthAccountContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	_, _, activeWorkspaceID, _ := currentSessionState(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	workspace, err := resolveActiveWorkspace(ctx, user, activeWorkspaceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, authWorkspaceContextResponse{
		Workspace: toAuthWorkspace(workspace),
	})
}

func handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	user, tokenHash, activeWorkspaceID, err := currentSessionState(r)
	if err != nil {
		if errors.Is(err, store.ErrSessionNotFound) || errors.Is(err, http.ErrNoCookie) {
			writeJSON(w, http.StatusUnauthorized, apiError{Error: "authentication required"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	switch r.Method {
	case http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		workspaces, err := appStore.ListWorkspaceContextsByUserID(ctx, user.ID)
		if err != nil {
			if errors.Is(err, store.ErrUserNotFound) {
				workspace, ensureErr := appStore.EnsureDefaultWorkspaceForUser(ctx, user)
				if ensureErr != nil {
					writeJSON(w, http.StatusInternalServerError, apiError{Error: ensureErr.Error()})
					return
				}
				workspaces = []store.WorkspaceContext{workspace}
			} else {
				writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
				return
			}
		}

		if activeWorkspaceID == 0 && len(workspaces) > 0 {
			activeWorkspaceID = workspaces[0].ID
		}
		resp := workspaceListResponse{ActiveWorkspaceID: activeWorkspaceID}
		for _, workspace := range workspaces {
			resp.Workspaces = append(resp.Workspaces, toAuthWorkspace(workspace))
		}
		writeJSON(w, http.StatusOK, resp)
	case http.MethodPost:
		if !isAllowedRequestOrigin(r) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
			return
		}
		var req struct {
			WorkspaceID int64 `json:"workspace_id"`
		}
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			return
		}
		if req.WorkspaceID <= 0 {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "workspace id is required"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := appStore.SetSessionActiveWorkspace(ctx, tokenHash, user.ID, req.WorkspaceID); err != nil {
			if errors.Is(err, store.ErrWorkspaceMemberNotFound) {
				writeJSON(w, http.StatusNotFound, apiError{Error: "workspace not available for this user"})
				return
			}
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, authMessageResponse{
			OK:      true,
			Message: "Active workspace updated.",
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
	}
}

func handleWorkspaceTeam(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	_, _, activeWorkspaceID, _ := currentSessionState(r)

	switch r.Method {
	case http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		workspace, err := resolveActiveWorkspace(ctx, user, activeWorkspaceID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}

		members, err := appStore.ListWorkspaceMembersByWorkspaceIDAndUserID(ctx, workspace.ID, user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}
		invitations, err := appStore.ListWorkspaceInvitationsByWorkspaceIDAndUserID(ctx, workspace.ID, user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, workspaceTeamResponse{
			Workspace:   toAuthWorkspace(workspace),
			Members:     toAuthWorkspaceMembers(members),
			Invitations: toAuthWorkspaceInvitations(invitations),
		})
	case http.MethodPost:
		if !isAllowedRequestOrigin(r) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
			return
		}
		if isDemoUser(user) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "demo access cannot issue workspace invitations"})
			return
		}

		var req workspaceInvitationRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		workspace, err := resolveActiveWorkspace(ctx, user, activeWorkspaceID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}

		invitation, err := appStore.CreateWorkspaceInvitationByWorkspaceIDAndUserID(ctx, workspace.ID, user.ID, req.Email, req.Role, time.Now().Add(7*24*time.Hour))
		if err != nil {
			switch {
			case errors.Is(err, store.ErrWorkspaceInviteConflict):
				writeJSON(w, http.StatusConflict, apiError{Error: "user is already a member of this workspace"})
			case errors.Is(err, store.ErrWorkspacePermissionDenied):
				writeJSON(w, http.StatusForbidden, apiError{Error: "workspace invitation role is not allowed for this operator"})
			case errors.Is(err, store.ErrWorkspaceRoleInvalid):
				writeJSON(w, http.StatusBadRequest, apiError{Error: "workspace role must be owner, admin, member, or viewer"})
			default:
				writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			}
			return
		}

		writeJSON(w, http.StatusCreated, authMessageResponse{
			OK:      true,
			Message: fmt.Sprintf("Invitation queued for %s as %s.", invitation.Email, invitation.Role),
			Email:   invitation.Email,
		})
	case http.MethodPatch:
		if !isAllowedRequestOrigin(r) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
			return
		}
		if isDemoUser(user) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "demo access cannot modify workspace members"})
			return
		}

		var req workspaceMemberUpdateRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			return
		}
		if req.UserID <= 0 {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "workspace member user id is required"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		workspace, err := resolveActiveWorkspace(ctx, user, activeWorkspaceID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}

		member, err := appStore.UpdateWorkspaceMemberRoleByWorkspaceIDAndUserID(ctx, workspace.ID, user.ID, req.UserID, req.Role)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrWorkspaceMemberNotFound):
				writeJSON(w, http.StatusNotFound, apiError{Error: "workspace member not found"})
			case errors.Is(err, store.ErrWorkspacePermissionDenied):
				writeJSON(w, http.StatusForbidden, apiError{Error: "workspace member update is not allowed"})
			case errors.Is(err, store.ErrWorkspaceRoleInvalid):
				writeJSON(w, http.StatusBadRequest, apiError{Error: "workspace role must be admin, member, or viewer"})
			default:
				writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			}
			return
		}

		writeJSON(w, http.StatusOK, authMessageResponse{
			OK:      true,
			Message: fmt.Sprintf("Updated %s to %s.", member.Email, member.Role),
			Email:   member.Email,
		})
	case http.MethodDelete:
		if !isAllowedRequestOrigin(r) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
			return
		}
		if isDemoUser(user) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "demo access cannot modify workspace members"})
			return
		}

		var req workspaceMemberRemoveRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			return
		}
		if req.UserID <= 0 && req.InvitationID <= 0 {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "workspace member user id or invitation id is required"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		workspace, err := resolveActiveWorkspace(ctx, user, activeWorkspaceID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}

		if req.InvitationID > 0 {
			invitation, err := appStore.CancelWorkspaceInvitationByWorkspaceIDAndUserID(ctx, workspace.ID, user.ID, req.InvitationID)
			if err != nil {
				switch {
				case errors.Is(err, store.ErrWorkspaceInvitationNotFound):
					writeJSON(w, http.StatusNotFound, apiError{Error: "workspace invitation not found"})
				case errors.Is(err, store.ErrWorkspacePermissionDenied):
					writeJSON(w, http.StatusForbidden, apiError{Error: "workspace invitation revoke is not allowed"})
				default:
					writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
				}
				return
			}

			writeJSON(w, http.StatusOK, authMessageResponse{
				OK:      true,
				Message: fmt.Sprintf("Revoked invitation for %s.", invitation.Email),
				Email:   invitation.Email,
			})
			return
		}

		if err := appStore.RemoveWorkspaceMemberByWorkspaceIDAndUserID(ctx, workspace.ID, user.ID, req.UserID); err != nil {
			switch {
			case errors.Is(err, store.ErrWorkspaceMemberNotFound):
				writeJSON(w, http.StatusNotFound, apiError{Error: "workspace member not found"})
			case errors.Is(err, store.ErrWorkspacePermissionDenied):
				writeJSON(w, http.StatusForbidden, apiError{Error: "workspace member removal is not allowed"})
			default:
				writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			}
			return
		}

		writeJSON(w, http.StatusOK, authMessageResponse{
			OK:      true,
			Message: "Workspace member removed.",
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
	}
}

func handleWorkspaceInvitations(w http.ResponseWriter, r *http.Request) {
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		invitations, err := appStore.ListIncomingWorkspaceInvitationsByEmail(ctx, user.Email)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, workspaceInvitationsResponse{
			Invitations: toAuthWorkspaceInvitations(invitations),
		})
	case http.MethodPost:
		if !isAllowedRequestOrigin(r) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
			return
		}

		var req workspaceInvitationActionRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			return
		}
		if strings.TrimSpace(strings.ToLower(req.Action)) != "accept" {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "workspace invitation action must be accept"})
			return
		}
		if req.InvitationID <= 0 {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "workspace invitation id is required"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		invitation, err := appStore.AcceptWorkspaceInvitationByUserID(ctx, user.ID, req.InvitationID)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrWorkspaceInvitationNotFound):
				writeJSON(w, http.StatusNotFound, apiError{Error: "workspace invitation not found"})
			case errors.Is(err, store.ErrWorkspaceInvitationExpired):
				writeJSON(w, http.StatusConflict, apiError{Error: "workspace invitation is expired"})
			default:
				writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			}
			return
		}

		writeJSON(w, http.StatusOK, authMessageResponse{
			OK:      true,
			Message: fmt.Sprintf("Joined %s as %s.", invitation.WorkspaceName, invitation.Role),
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
	}
}

func handleAdminOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}
	if _, ok := requirePlatformAdmin(w, r); !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	overview, err := appStore.GetAdminOverview(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, overview)
}

func handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}
	if _, ok := requirePlatformAdmin(w, r); !ok {
		return
	}

	limit := parseAdminListLimit(r.URL.Query().Get("limit"))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	users, err := appStore.ListAdminUsers(ctx, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, adminUsersResponse{
		Users: toAuthAdminUsers(users),
	})
}

func handleAdminWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}
	if _, ok := requirePlatformAdmin(w, r); !ok {
		return
	}

	limit := parseAdminListLimit(r.URL.Query().Get("limit"))
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	workspaces, err := appStore.ListAdminWorkspaces(ctx, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, adminWorkspacesResponse{
		Workspaces: toAuthAdminWorkspaces(workspaces),
	})
}

func handleAdminSystem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}
	if _, ok := requirePlatformAdmin(w, r); !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	overview, err := appStore.GetAdminOverview(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, authAdminSystemResponse{
		Overview: overview,
		Runtime: authAdminSystemRuntime{
			DatabaseDriver: appStore.Driver(),
			RedisEnabled:   appCache != nil,
			ObjectStorage:  appObjectStore != nil,
			EmailDelivery:  appMailer != nil,
		},
		Features: authAdminSystemFeatures{
			PublicBaseURL:    appPublicBaseURL,
			DemoEnabled:      appDemoEnabled,
			SessionTTLHours:  int(defaultSessionTTL / time.Hour),
			ConfiguredAdmins: len(appAdminEmails),
		},
	})
}

func handleAuthDemoLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !appDemoEnabled {
		writeJSON(w, http.StatusNotFound, apiError{Error: "demo access is disabled"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}

	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	user, err := getOrCreateDemoUser(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if err := issueSession(w, r, ctx, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	clearLegacyDemoCookie(w, r)
	clearLegacyPreviewCookie(w, r)
	writeJSON(w, http.StatusOK, authMessageResponse{
		OK:            true,
		Message:       "Demo access granted.",
		Authenticated: true,
		User:          toAuthUser(user),
		Mode:          "demo",
	})
}

func handleAuthSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req authRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	name, err := normalizeSignupDisplayName(req.Name, req.FirstName, req.LastName)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	email, err := store.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if !enforceRateLimit(w, r, "auth-signup", email, rateLimitConfig{Limit: 5, Window: 15 * time.Minute}) {
		return
	}
	if err := validatePassword(req.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if err := validatePasswordConfirmation(req.Password, req.ConfirmPassword); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "failed to hash password"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_ = appStore.DeleteExpiredSessions(ctx)
	_ = appStore.DeleteExpiredEmailVerificationCodes(ctx)
	user, err := appStore.CreateUser(ctx, name, email, string(passwordHash), true)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrUserExists):
			writeJSON(w, http.StatusConflict, apiError{Error: "user already exists"})
		default:
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		}
		return
	}

	resp, err := prepareVerificationChallenge(ctx, r, user)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, apiError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, resp)
}

func handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req authRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	email, err := store.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if !enforceRateLimit(w, r, "auth-login", email, rateLimitConfig{Limit: 8, Window: 15 * time.Minute}) {
		return
	}
	if strings.TrimSpace(req.Password) == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "password is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_ = appStore.DeleteExpiredSessions(ctx)
	user, passwordHash, err := appStore.AuthenticateUser(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, apiError{Error: "invalid email or password"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, apiError{Error: "invalid email or password"})
		return
	}
	if user.VerificationRequired {
		writeJSON(w, http.StatusForbidden, authMessageResponse{
			OK:                   false,
			Message:              "Email verification required. Enter the OTP sent to your inbox.",
			VerificationRequired: true,
			Email:                user.Email,
		})
		return
	}

	if err := issueSession(w, r, ctx, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, authSessionResponse{
		Authenticated: true,
		User:          toAuthUser(user),
	})
}

func handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}
	if tokenHash, err := sessionTokenHashFromRequest(r); err == nil && tokenHash != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		_ = appStore.DeleteSession(ctx, tokenHash)
		cancel()
	}
	clearSessionCookie(w, r)
	clearLegacyDemoCookie(w, r)
	clearLegacyPreviewCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleAuthProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if isDemoUser(user) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "demo access cannot modify the shared profile"})
		return
	}

	var req authProfileUpdateRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	name, err := normalizeDisplayName(req.Name)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	email, err := store.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	updatedUser, emailChanged, err := appStore.UpdateUserProfile(ctx, user.ID, name, email)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrUserExists):
			writeJSON(w, http.StatusConflict, apiError{Error: "user already exists"})
		case errors.Is(err, store.ErrUserNotFound):
			writeJSON(w, http.StatusNotFound, apiError{Error: "user not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		}
		return
	}

	if emailChanged {
		resp, challengeErr := prepareVerificationChallenge(ctx, r, updatedUser)
		if challengeErr != nil {
			log.Printf("profile verification challenge failed for %s: %v", updatedUser.Email, challengeErr)
			writeJSON(w, http.StatusOK, authMessageResponse{
				OK:                   true,
				Authenticated:        true,
				User:                 toAuthUser(updatedUser),
				Email:                updatedUser.Email,
				VerificationRequired: updatedUser.VerificationRequired,
				Message:              "Profile updated. Verification delivery could not be issued automatically; request a new OTP from this page.",
			})
			return
		}
		resp.Authenticated = true
		resp.User = toAuthUser(updatedUser)
		resp.Email = updatedUser.Email
		resp.Message = "Profile updated. Enter the OTP sent to your new email address to re-verify the account."
		writeJSON(w, http.StatusOK, resp)
		return
	}

	writeJSON(w, http.StatusOK, authMessageResponse{
		OK:            true,
		Authenticated: true,
		User:          toAuthUser(updatedUser),
		Email:         updatedUser.Email,
		Message:       "Profile updated.",
	})
}

func handleAuthVerifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req authVerifyEmailRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	email, err := store.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if !enforceRateLimit(w, r, "auth-verify-email", email, rateLimitConfig{Limit: 6, Window: 10 * time.Minute}) {
		return
	}
	code := normalizeVerificationCode(req.Code)
	if code == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "verification code is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_ = appStore.DeleteExpiredEmailVerificationCodes(ctx)
	user, err := appStore.VerifyUserEmailCode(ctx, email, hashOpaqueToken(code))
	if err != nil {
		if errors.Is(err, store.ErrVerificationCode) {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "verification code is invalid or expired"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	if err := issueSession(w, r, ctx, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	if appMailer != nil {
		welcomeCtx, welcomeCancel := context.WithTimeout(context.Background(), 8*time.Second)
		if err := appMailer.SendWelcome(welcomeCtx, mailer.WelcomeEmailInput{
			To:           user.Email,
			Name:         user.Name,
			DashboardURL: buildDashboardURL(r, "alerts"),
		}); err != nil {
			log.Printf("welcome email failed for %s: %v", user.Email, err)
		}
		welcomeCancel()
	}

	writeJSON(w, http.StatusOK, authMessageResponse{
		OK:            true,
		Authenticated: true,
		User:          toAuthUser(user),
		Message:       "Email verified. Access granted.",
		Email:         user.Email,
	})
}

func handleAuthResendVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req authForgotPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	email, err := store.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if !enforceRateLimit(w, r, "auth-resend-verification", email, rateLimitConfig{Limit: 3, Window: 10 * time.Minute}) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	user, err := appStore.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			writeJSON(w, http.StatusOK, authMessageResponse{
				OK:      true,
				Message: "If an account is pending verification, a new OTP has been issued.",
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if !user.VerificationRequired {
		writeJSON(w, http.StatusOK, authMessageResponse{
			OK:      true,
			Message: "This email is already verified. You can sign in now.",
			Email:   user.Email,
		})
		return
	}
	recent, err := appStore.HasRecentEmailVerificationCode(ctx, user.ID, time.Now().Add(-verifyResendCooldown))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if recent {
		writeJSON(w, http.StatusOK, authMessageResponse{
			OK:      true,
			Message: "A recent verification code is still active. Check your inbox before requesting another.",
			Email:   user.Email,
		})
		return
	}

	resp, err := prepareVerificationChallenge(ctx, r, user)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, apiError{Error: err.Error()})
		return
	}
	resp.Message = "A fresh verification code has been sent."
	writeJSON(w, http.StatusOK, resp)
}

func handleAuthForgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req authForgotPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	email, err := store.NormalizeEmail(req.Email)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if !enforceRateLimit(w, r, "auth-forgot-password", email, rateLimitConfig{Limit: 3, Window: 15 * time.Minute}) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	_ = appStore.DeleteExpiredPasswordResetTokens(ctx)

	resp := authMessageResponse{
		OK:      true,
		Message: "If an account exists, a reset link has been issued.",
	}

	user, err := appStore.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			writeJSON(w, http.StatusOK, resp)
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	recent, err := appStore.HasRecentPasswordResetToken(ctx, user.ID, time.Now().Add(-resetIssueCooldown))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if recent {
		writeJSON(w, http.StatusOK, resp)
		return
	}

	token, tokenHash, err := generatePasswordResetToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "failed to create reset token"})
		return
	}
	expiresAt := time.Now().Add(defaultResetTTL)
	if err := appStore.CreatePasswordResetToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	resetURL := buildPasswordResetURL(r, token)
	if appMailer != nil {
		if err := appMailer.SendPasswordReset(ctx, mailer.PasswordResetInput{
			To:        user.Email,
			Name:      user.Name,
			ResetURL:  resetURL,
			ExpiresIn: defaultResetTTL,
		}); err != nil {
			if cleanupErr := appStore.DeletePasswordResetToken(ctx, user.ID); cleanupErr != nil {
				log.Printf("password reset token cleanup failed for %s: %v", user.Email, cleanupErr)
			}
			log.Printf("password reset email failed for %s: %v", user.Email, err)
			writeJSON(w, http.StatusBadGateway, apiError{Error: "email delivery is temporarily unavailable"})
			return
		}
		resp.Message = "If an account exists, a reset link has been sent to the inbox."
	} else {
		log.Printf("password reset requested for %s: %s (expires %s)", user.Email, resetURL, expiresAt.UTC().Format(time.RFC3339))
		resp.Message = "If an account exists, a reset link has been issued locally."
	}
	if appMailer == nil && isLocalRequestHost(r.Host) {
		resp.PreviewURL = resetURL
	}
	writeJSON(w, http.StatusOK, resp)
}

func handleAuthResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req authResetPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	token := strings.TrimSpace(req.Token)
	if token == "" {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "reset token is required"})
		return
	}
	if err := validatePassword(req.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if err := validatePasswordConfirmation(req.Password, req.ConfirmPassword); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "failed to hash password"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := appStore.DeleteExpiredPasswordResetTokens(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if err := appStore.ResetPasswordWithToken(ctx, hashOpaqueToken(token), string(passwordHash)); err != nil {
		if errors.Is(err, store.ErrResetTokenNotFound) {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "reset link is invalid or expired"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	clearSessionCookie(w, r)
	writeJSON(w, http.StatusOK, authMessageResponse{
		OK:      true,
		Message: "Password updated. Sign in with your new password.",
	})
}

func handleAlertRules(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		rules, err := appStore.ListAlertRules(ctx, user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, rules)
	case http.MethodPost:
		if !isAllowedRequestOrigin(r) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
			return
		}
		var req alertRuleRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			return
		}
		rule, err := normalizeAlertRuleRequest(user, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		saved, err := appStore.SaveAlertRule(ctx, rule)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrAlertRuleNotFound):
				writeJSON(w, http.StatusNotFound, apiError{Error: "alert rule not found"})
			default:
				writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			}
			return
		}
		status := http.StatusOK
		if req.ID == 0 {
			status = http.StatusCreated
		}
		writeJSON(w, status, saved)
	case http.MethodDelete:
		if !isAllowedRequestOrigin(r) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
			return
		}
		ruleID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
		if err != nil || ruleID <= 0 {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "valid rule id is required"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := appStore.DeleteAlertRule(ctx, user.ID, ruleID); err != nil {
			switch {
			case errors.Is(err, store.ErrAlertRuleNotFound):
				writeJSON(w, http.StatusNotFound, apiError{Error: "alert rule not found"})
			default:
				writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": ruleID})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
	}
}

func normalizeAlertRuleRequest(user store.User, req alertRuleRequest) (store.AlertRule, error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Protocol))
	target := strings.TrimSpace(req.Target)
	port := req.Port

	switch protocol {
	case "ping":
		normalizedTarget, err := normalizeHost(target)
		if err != nil {
			return store.AlertRule{}, err
		}
		target = normalizedTarget
		port = 0
	case "tcpping":
		if parsedHost, parsedPort, ok := splitHostPort(target); ok {
			target = parsedHost
			if port == 0 {
				port = parsedPort
			}
		}
		normalizedTarget, normalizedPort, _, err := normalizeTCPPingRequest(target, port, 1, defaultProbeInterval, defaultProbeTimeout, false)
		if err != nil {
			return store.AlertRule{}, err
		}
		target = normalizedTarget
		port = normalizedPort
	default:
		return store.AlertRule{}, fmt.Errorf("protocol must be ping or tcpping")
	}

	var latencyThreshold *float64
	if req.LatencyThresholdMs > 0 {
		v := req.LatencyThresholdMs
		latencyThreshold = &v
	}
	var lossThreshold *float64
	if req.LossThresholdPercent > 0 {
		v := req.LossThresholdPercent
		lossThreshold = &v
	}

	recipientEmail := strings.TrimSpace(req.RecipientEmail)
	if recipientEmail == "" {
		recipientEmail = user.Email
	}

	notifyOnRecovery := true
	if req.NotifyOnRecovery != nil {
		notifyOnRecovery = *req.NotifyOnRecovery
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := store.AlertRule{
		ID:                   req.ID,
		UserID:               user.ID,
		Name:                 strings.TrimSpace(req.Name),
		Protocol:             protocol,
		Target:               target,
		RecipientEmail:       recipientEmail,
		LatencyThresholdMs:   latencyThreshold,
		LossThresholdPercent: lossThreshold,
		ConsecutiveBreaches:  req.ConsecutiveBreaches,
		CooldownMinutes:      req.CooldownMinutes,
		NotifyOnRecovery:     notifyOnRecovery,
		Enabled:              enabled,
	}
	if protocol == "tcpping" {
		rule.Port = &port
	}
	return rule, nil
}

func requireAuthenticatedUser(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return store.User{}, false
	}
	user, _, err := currentSessionUser(r)
	if err != nil {
		if errors.Is(err, store.ErrSessionNotFound) || errors.Is(err, http.ErrNoCookie) {
			writeJSON(w, http.StatusUnauthorized, apiError{Error: "authentication required"})
			return store.User{}, false
		}
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return store.User{}, false
	}
	return user, true
}

func requirePlatformAdmin(w http.ResponseWriter, r *http.Request) (store.User, bool) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return store.User{}, false
	}
	if !isPlatformAdmin(user) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "platform admin access required"})
		return store.User{}, false
	}
	return user, true
}

func parseAdminListLimit(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 50
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func currentSessionState(r *http.Request) (store.User, string, int64, error) {
	tokenHash, err := sessionTokenHashFromRequest(r)
	if err != nil {
		return store.User{}, "", 0, err
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	session, err := appStore.GetSessionDetailsByToken(ctx, tokenHash)
	if err != nil {
		return store.User{}, "", 0, err
	}
	return session.User, tokenHash, session.ActiveWorkspaceID, nil
}

func currentSessionUser(r *http.Request) (store.User, string, error) {
	user, tokenHash, _, err := currentSessionState(r)
	if err != nil {
		return store.User{}, "", err
	}
	return user, tokenHash, nil
}

func sessionTokenHashFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(cookie.Value)
	if token == "" {
		return "", http.ErrNoCookie
	}
	return hashSessionToken(token), nil
}

func issueSession(w http.ResponseWriter, r *http.Request, ctx context.Context, user store.User) error {
	token, tokenHash, err := generateSessionToken()
	if err != nil {
		return fmt.Errorf("failed to create session token")
	}
	expiresAt := time.Now().Add(defaultSessionTTL)
	if err := appStore.CreateSession(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return err
	}
	setSessionCookie(w, r, token, expiresAt)
	return nil
}

func getOrCreateDemoUser(ctx context.Context) (store.User, error) {
	user, err := appStore.GetUserByEmail(ctx, appDemoEmail)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, store.ErrUserNotFound) {
		return store.User{}, err
	}

	passwordToken, _, genErr := generateOpaqueToken()
	if genErr != nil {
		return store.User{}, fmt.Errorf("failed to generate demo credentials")
	}
	passwordHash, hashErr := bcrypt.GenerateFromPassword([]byte(passwordToken), bcrypt.DefaultCost)
	if hashErr != nil {
		return store.User{}, fmt.Errorf("failed to prepare demo account")
	}
	return appStore.CreateUser(ctx, appDemoName, appDemoEmail, string(passwordHash), false)
}

func isDemoUser(user store.User) bool {
	if !appDemoEnabled || appDemoEmail == "" {
		return false
	}
	return strings.EqualFold(user.Email, appDemoEmail)
}

func isPlatformAdmin(user store.User) bool {
	if len(appAdminEmails) == 0 {
		return false
	}
	_, ok := appAdminEmails[strings.ToLower(strings.TrimSpace(user.Email))]
	return ok
}

func platformRoleForUser(user store.User) string {
	if isPlatformAdmin(user) {
		return "admin"
	}
	return "user"
}

func resolveActiveWorkspace(ctx context.Context, user store.User, activeWorkspaceID int64) (store.WorkspaceContext, error) {
	if activeWorkspaceID > 0 {
		workspace, err := appStore.GetWorkspaceContextByWorkspaceIDAndUserID(ctx, activeWorkspaceID, user.ID)
		if err == nil {
			return workspace, nil
		}
		if !errors.Is(err, store.ErrWorkspaceMemberNotFound) {
			return store.WorkspaceContext{}, err
		}
	}

	workspace, err := appStore.GetWorkspaceContextByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return appStore.EnsureDefaultWorkspaceForUser(ctx, user)
		}
		return store.WorkspaceContext{}, err
	}
	return workspace, nil
}

func generateSessionToken() (string, string, error) {
	return generateOpaqueToken()
}

func generatePasswordResetToken() (string, string, error) {
	return generateOpaqueToken()
}

func generateVerificationCode() (string, string, error) {
	const digits = "0123456789"
	buf := make([]byte, 6)
	for i := range buf {
		n, err := rand.Int(rand.Reader, bigTen)
		if err != nil {
			return "", "", err
		}
		buf[i] = digits[n.Int64()]
	}
	code := string(buf)
	return code, hashOpaqueToken(code), nil
}

func generateOpaqueToken() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	return token, hashOpaqueToken(token), nil
}

func hashSessionToken(token string) string {
	return hashOpaqueToken(token)
}

func hashOpaqueToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

var bigTen = func() *big.Int {
	return big.NewInt(10)
}()

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func clearLegacyDemoCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "qnetward_demo",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func clearLegacyPreviewCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "qnetward_preview",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func requestIsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https") {
		return true
	}
	if strings.Contains(strings.ToLower(r.Header.Get("Cf-Visitor")), "\"https\"") {
		return true
	}
	return false
}

func requestHost(r *http.Request) string {
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		return forwardedHost
	}
	return r.Host
}

func requestScheme(r *http.Request) string {
	if requestIsSecure(r) {
		return "https"
	}
	return "http"
}

func buildPasswordResetURL(r *http.Request, token string) string {
	if appPublicBaseURL != "" {
		return fmt.Sprintf("%s/reset-password?token=%s", appPublicBaseURL, url.QueryEscape(token))
	}
	return fmt.Sprintf("%s://%s/reset-password?token=%s", requestScheme(r), requestHost(r), url.QueryEscape(token))
}

func buildDashboardURL(r *http.Request, panel string) string {
	base := buildConfiguredDashboardURL(panel)
	if base != "" {
		return base
	}
	return fmt.Sprintf("%s://%s%s", requestScheme(r), requestHost(r), dashboardRoutePath(panel))
}

func buildConfiguredDashboardURL(panel string) string {
	base := strings.TrimRight(strings.TrimSpace(appPublicBaseURL), "/")
	if base == "" {
		return ""
	}
	return base + dashboardRoutePath(panel)
}

func dashboardRoutePath(panel string) string {
	switch strings.TrimSpace(panel) {
	case "", "overview":
		return "/"
	case "ping":
		return "/icmp-ping"
	case "tcpping":
		return "/tcp-ping"
	case "portscan":
		return "/port-scan"
	case "dnslookup":
		return "/dns-lookup"
	case "history":
		return "/history"
	case "alerts":
		return "/alerts"
	case "settings":
		return "/settings"
	case "profile":
		return "/profile"
	default:
		return "/"
	}
}

func enqueueAlertEvaluation(user store.User, protocol, target string, port int, summary report.Summary, dashboardURL string) {
	if appStore == nil {
		return
	}

	sample := store.AlertSample{
		LossPercent: summary.Loss,
		Sent:        summary.Sent,
		Recv:        summary.Recv,
	}
	if summary.Recv > 0 {
		avgMs := float64(summary.Avg) / float64(time.Millisecond)
		sample.LatencyAvgMs = &avgMs
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		decisions, err := appStore.EvaluateAlertRules(ctx, user.ID, protocol, target, port, sample)
		if err != nil {
			log.Printf("alert evaluation failed for %s/%s: %v", protocol, target, err)
			return
		}
		if appMailer == nil {
			return
		}
		for _, decision := range decisions {
			alertTitle := decision.Rule.Name
			if strings.TrimSpace(alertTitle) == "" {
				alertTitle = "Monitoring Alert"
			}
			if err := appMailer.SendAlert(ctx, mailer.AlertEmailInput{
				To:           decision.Rule.RecipientEmail,
				Name:         user.Name,
				Severity:     decision.Severity,
				Title:        alertTitle,
				Summary:      decision.Summary,
				Target:       alertTargetLabel(decision.Rule),
				TriggeredAt:  time.Now().UTC(),
				DashboardURL: dashboardURL,
			}); err != nil {
				log.Printf("alert email failed for rule %d: %v", decision.Rule.ID, err)
			}
		}
	}()
}

func alertTargetLabel(rule store.AlertRule) string {
	if rule.Protocol == "tcpping" && rule.Port != nil {
		return fmt.Sprintf("%s:%d", rule.Target, *rule.Port)
	}
	return rule.Target
}

func isLocalRequestHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if strings.Contains(host, ":") {
		if parsedHost, _, err := net.SplitHostPort(host); err == nil {
			host = parsedHost
		}
	}
	host = strings.Trim(host, "[]")
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func normalizeVerificationCode(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	return value
}

func prepareVerificationChallenge(ctx context.Context, r *http.Request, user store.User) (authMessageResponse, error) {
	code, codeHash, err := generateVerificationCode()
	if err != nil {
		return authMessageResponse{}, fmt.Errorf("failed to generate verification code")
	}
	expiresAt := time.Now().Add(defaultVerifyCodeTTL)
	if err := appStore.CreateEmailVerificationCode(ctx, user.ID, codeHash, expiresAt); err != nil {
		return authMessageResponse{}, err
	}

	resp := authMessageResponse{
		OK:                   true,
		Message:              "Account created. Enter the OTP sent to your email to activate access.",
		VerificationRequired: true,
		Email:                user.Email,
	}

	if appMailer != nil {
		if err := appMailer.SendVerificationOTP(ctx, mailer.VerificationOTPInput{
			To:        user.Email,
			Name:      user.Name,
			Code:      code,
			ExpiresIn: defaultVerifyCodeTTL,
		}); err != nil {
			if cleanupErr := appStore.DeleteEmailVerificationCode(ctx, user.ID); cleanupErr != nil {
				log.Printf("verification code cleanup failed for %s: %v", user.Email, cleanupErr)
			}
			log.Printf("verification email failed for %s: %v", user.Email, err)
			resp.Message = "Account created, but email delivery failed. Request a new OTP to continue."
			return resp, nil
		}
		return resp, nil
	}

	log.Printf("email verification requested for %s: otp=%s (expires %s)", user.Email, code, expiresAt.UTC().Format(time.RFC3339))
	resp.PreviewCode = code
	resp.Message = "Account created. Enter the OTP below to activate access."
	return resp, nil
}

func normalizeDisplayName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("name is required")
	}
	if len(value) > maxNameLength {
		return "", fmt.Errorf("name must be %d characters or fewer", maxNameLength)
	}
	return value, nil
}

func normalizeSignupDisplayName(name, firstName, lastName string) (string, error) {
	name = strings.TrimSpace(name)
	if name != "" {
		return normalizeDisplayName(name)
	}

	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	if firstName == "" {
		return "", fmt.Errorf("first name is required")
	}
	if lastName == "" {
		return "", fmt.Errorf("last name is required")
	}
	return normalizeDisplayName(firstName + " " + lastName)
}

func validatePassword(value string) error {
	if len(value) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	if len(value) > maxPasswordLength {
		return fmt.Errorf("password must be %d characters or fewer", maxPasswordLength)
	}
	return nil
}

func validatePasswordConfirmation(password, confirmPassword string) error {
	confirmPassword = strings.TrimSpace(confirmPassword)
	if confirmPassword == "" {
		return nil
	}
	if password != confirmPassword {
		return fmt.Errorf("passwords do not match")
	}
	return nil
}

func toAuthUser(user store.User) *authUser {
	return &authUser{
		ID:              user.ID,
		Name:            user.Name,
		Email:           user.Email,
		EmailVerifiedAt: user.EmailVerifiedAt,
		CreatedAt:       user.CreatedAt,
	}
}

type authWorkspaceContextResponse struct {
	Workspace *authWorkspace `json:"workspace,omitempty"`
}

type workspaceListResponse struct {
	ActiveWorkspaceID int64            `json:"active_workspace_id,omitempty"`
	Workspaces        []*authWorkspace `json:"workspaces,omitempty"`
}

type workspaceTeamResponse struct {
	Workspace   *authWorkspace            `json:"workspace,omitempty"`
	Members     []authWorkspaceMember     `json:"members,omitempty"`
	Invitations []authWorkspaceInvitation `json:"invitations,omitempty"`
}

type workspaceInvitationsResponse struct {
	Invitations []authWorkspaceInvitation `json:"invitations,omitempty"`
}

type adminUsersResponse struct {
	Users []authAdminUser `json:"users,omitempty"`
}

type adminWorkspacesResponse struct {
	Workspaces []authAdminWorkspace `json:"workspaces,omitempty"`
}

func toAuthWorkspace(workspace store.WorkspaceContext) *authWorkspace {
	return &authWorkspace{
		ID:            workspace.ID,
		Name:          workspace.Name,
		Plan:          workspace.Plan,
		BillingStatus: workspace.BillingStatus,
		Role:          workspace.Role,
		MemberCount:   workspace.MemberCount,
		CreatedAt:     workspace.CreatedAt,
	}
}

func toAuthWorkspaceMembers(members []store.WorkspaceMember) []authWorkspaceMember {
	resp := make([]authWorkspaceMember, 0, len(members))
	for _, member := range members {
		resp = append(resp, authWorkspaceMember{
			UserID:    member.UserID,
			Name:      member.Name,
			Email:     member.Email,
			Role:      member.Role,
			JoinedAt:  member.JoinedAt,
			IsCurrent: member.IsCurrent,
		})
	}
	return resp
}

func toAuthWorkspaceInvitations(invitations []store.WorkspaceInvitation) []authWorkspaceInvitation {
	resp := make([]authWorkspaceInvitation, 0, len(invitations))
	for _, invitation := range invitations {
		resp = append(resp, authWorkspaceInvitation{
			ID:            invitation.ID,
			WorkspaceID:   invitation.WorkspaceID,
			WorkspaceName: invitation.WorkspaceName,
			Email:         invitation.Email,
			Role:          invitation.Role,
			Status:        invitation.Status,
			ExpiresAt:     invitation.ExpiresAt,
			CreatedAt:     invitation.CreatedAt,
			InviterName:   invitation.InviterName,
		})
	}
	return resp
}

func toAuthAdminUsers(users []store.AdminUserSummary) []authAdminUser {
	resp := make([]authAdminUser, 0, len(users))
	for _, user := range users {
		resp = append(resp, authAdminUser{
			ID:                  user.ID,
			Name:                user.Name,
			Email:               user.Email,
			EmailVerifiedAt:     user.EmailVerifiedAt,
			CreatedAt:           user.CreatedAt,
			OwnedWorkspaceCount: user.OwnedWorkspaceCount,
			WorkspaceCount:      user.WorkspaceCount,
			ActiveSessionCount:  user.ActiveSessionCount,
			PlatformRole:        platformRoleForUser(store.User{Email: user.Email}),
		})
	}
	return resp
}

func toAuthAdminWorkspaces(workspaces []store.AdminWorkspaceSummary) []authAdminWorkspace {
	resp := make([]authAdminWorkspace, 0, len(workspaces))
	for _, workspace := range workspaces {
		resp = append(resp, authAdminWorkspace{
			ID:                     workspace.ID,
			Name:                   workspace.Name,
			Plan:                   workspace.Plan,
			BillingStatus:          workspace.BillingStatus,
			CreatedAt:              workspace.CreatedAt,
			OwnerUserID:            workspace.OwnerUserID,
			OwnerName:              workspace.OwnerName,
			OwnerEmail:             workspace.OwnerEmail,
			MemberCount:            workspace.MemberCount,
			PendingInvitationCount: workspace.PendingInvitationCount,
			ActiveSessionCount:     workspace.ActiveSessionCount,
		})
	}
	return resp
}

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

func handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}

	var req pingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if normalizedHost, err := normalizeHost(req.Host); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	} else {
		req.Host = normalizedHost
	}

	count := req.Count
	interval := time.Second
	if req.IntervalMs > 0 {
		interval = time.Duration(req.IntervalMs) * time.Millisecond
	}
	timeout := 2 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	size := req.Size

	if req.Continuous {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "continuous mode requires realtime streaming"})
		return
	}
	cfg, err := normalizePingConfig(count, interval, timeout, size, req.IPv6, false)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	release, ok := tryAcquireProbeSlot()
	if !ok {
		writeJSON(w, http.StatusTooManyRequests, apiError{Error: "server busy, try again"})
		return
	}
	defer release()

	rep, err := ping.RunStreamContext(r.Context(), req.Host, cfg, func(res report.Result) error {
		storeResult(user.ID, "ping", req.Host, res.Addr, 0, res, time.Now())
		return nil
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	enqueueAlertEvaluation(user, "ping", req.Host, 0, rep.Summary, buildDashboardURL(r, "alerts"))

	writeJSON(w, http.StatusOK, toJSONReport(rep))
}

func handleTCPPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}

	var req tcpPingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	host := req.Host
	port := req.Port
	if parsedHost, parsedPort, ok := splitHostPort(host); ok {
		host = parsedHost
		if port == 0 {
			port = parsedPort
		}
	}
	if port == 0 {
		port = 80
	}

	count := req.Count
	interval := time.Second
	if req.IntervalMs > 0 {
		interval = time.Duration(req.IntervalMs) * time.Millisecond
	}
	timeout := 2 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}

	if req.Continuous {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "continuous mode requires realtime streaming"})
		return
	}
	host, port, cfg, err := normalizeTCPPingRequest(host, port, count, interval, timeout, false)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	release, ok := tryAcquireProbeSlot()
	if !ok {
		writeJSON(w, http.StatusTooManyRequests, apiError{Error: "server busy, try again"})
		return
	}
	defer release()

	rep, err := tcpping.RunStreamContext(r.Context(), host, port, cfg, func(res report.Result) error {
		storeResult(user.ID, "tcpping", host, res.Addr, port, res, time.Now())
		return nil
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	enqueueAlertEvaluation(user, "tcpping", host, port, rep.Summary, buildDashboardURL(r, "alerts"))

	writeJSON(w, http.StatusOK, toJSONReport(rep))
}

func handlePortScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if _, ok := requireAuthenticatedUser(w, r); !ok {
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}

	var req portScanRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	timeout := defaultProbeTimeout
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}

	host, _, cfg, err := normalizePortScanRequest(req.Host, req.Ports, timeout, req.Concurrency)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	release, ok := tryAcquireProbeSlot()
	if !ok {
		writeJSON(w, http.StatusTooManyRequests, apiError{Error: "server busy, try again"})
		return
	}
	defer release()

	rep, err := portscan.RunContext(r.Context(), host, cfg)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, toJSONPortScanReport(rep))
}

func handleDNSLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if _, ok := requireAuthenticatedUser(w, r); !ok {
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}

	var req dnsLookupRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	target, err := normalizeHost(req.Target)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	timeout := dnslookup.DefaultTimeout
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	timeout, err = normalizeDNSLookupTimeout(timeout)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	release, ok := tryAcquireProbeSlot()
	if !ok {
		writeJSON(w, http.StatusTooManyRequests, apiError{Error: "server busy, try again"})
		return
	}
	defer release()

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	rep, err := dnslookup.RunContext(ctx, target, dnslookup.Config{
		Timeout:    timeout,
		RecordType: req.RecordType,
		Resolver:   req.Resolver,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	reqType, host, port, limit, err := parseHistoryQuery(r.URL.Query())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	cacheKey := historyCacheKey(user.ID, reqType, host, port, limit)
	if appCache != nil {
		var cached []store.HistoryPoint
		ctx, cancel := context.WithTimeout(r.Context(), 250*time.Millisecond)
		hit, cacheErr := appCache.GetJSON(ctx, cacheKey, &cached)
		cancel()
		if cacheErr == nil && hit {
			writeJSON(w, http.StatusOK, cached)
			return
		}
	}

	points, err := appStore.QueryHistory(user.ID, reqType, host, port, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}
	if appCache != nil && len(points) > 0 {
		ctx, cancel := context.WithTimeout(r.Context(), 250*time.Millisecond)
		_ = appCache.SetJSON(ctx, cacheKey, points)
		cancel()
	}
	writeJSON(w, http.StatusOK, points)
}

func handleRecentHistoryTargets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}

	limit, err := parseRecentHistoryTargetsLimit(r.URL.Query())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	targets, err := appStore.QueryRecentHistoryTargets(user.ID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, targets)
}

func handleHistoryExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	var req historyExportRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	reqType, host, port, limit, format, destination, err := parseHistoryExportRequest(req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	points, err := appStore.QueryHistory(user.ID, reqType, host, port, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return
	}

	if destination == "s3" {
		if appObjectStore == nil {
			writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "object storage not configured"})
			return
		}
		key := buildExportObjectKey(user.ID, reqType, host, port, format)
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		var objectKey string
		if format == "csv" {
			objectKey, err = appObjectStore.UploadCSV(ctx, key, historyCSVRows(points))
		} else {
			objectKey, err = appObjectStore.UploadJSON(ctx, key, points)
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}
		presignedURL, err := appObjectStore.PresignedGetURL(ctx, objectKey, 15*time.Minute)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":     true,
			"key":    objectKey,
			"url":    presignedURL,
			"format": format,
			"count":  len(points),
		})
		return
	}

	filename := exportFileName(reqType, host, port, format)
	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		cw := csv.NewWriter(w)
		if err := cw.WriteAll(historyCSVRows(points)); err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(points)
	}
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WebSocket upgrade error: %v\n", err)
		return
	}
	defer conn.Close()

	session := &wsSession{writer: &wsWriter{conn: conn}}
	defer session.close()

	conn.SetReadLimit(maxWSMessageBytes)
	conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	})

	for {
		var req wsRequest
		if err := conn.ReadJSON(&req); err != nil {
			// Check if it's a normal close or an error
			if !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return // Normal close
			}
			fmt.Fprintf(os.Stderr, "WebSocket read error: %v\n", err)
			return // Abnormal close or error
		}
		req.Type = strings.ToLower(strings.TrimSpace(req.Type))
		if req.Type == "" {
			_ = session.writer.WriteEvent(wsEvent{Type: "error", Error: "type is required"})
			continue
		}

		conn.SetReadDeadline(time.Now().Add(wsReadTimeout))

		switch req.Type {
		case "stop":
			if !session.stop() {
				_ = session.writer.WriteEvent(wsEvent{Type: "error", Error: "no active run"})
			}
		case "ping":
			if !session.start(user, req) {
				_ = session.writer.WriteEvent(wsEvent{Type: "error", Error: "a run is already in progress"})
			}
		case "tcpping":
			if !session.start(user, req) {
				_ = session.writer.WriteEvent(wsEvent{Type: "error", Error: "a run is already in progress"})
			}
		case "portscan":
			if !session.start(user, req) {
				_ = session.writer.WriteEvent(wsEvent{Type: "error", Error: "a run is already in progress"})
			}
		default:
			_ = session.writer.WriteEvent(wsEvent{Type: "error", Error: "unknown type"})
		}
	}
}

func runWSRequest(ctx context.Context, writer *wsWriter, user store.User, req wsRequest) error {
	switch req.Type {
	case "ping":
		return streamPingWS(ctx, writer, user, req)
	case "tcpping":
		return streamTCPPingWS(ctx, writer, user, req)
	case "portscan":
		return streamPortScanWS(ctx, writer, req)
	default:
		return fmt.Errorf("unknown type")
	}
}

func streamPingWS(ctx context.Context, writer *wsWriter, user store.User, req wsRequest) error {
	release, ok := tryAcquireProbeSlot()
	if !ok {
		return writer.WriteEvent(wsEvent{Type: "error", Error: "server busy, try again"})
	}
	defer release()

	interval := time.Second
	if req.IntervalMs > 0 {
		interval = time.Duration(req.IntervalMs) * time.Millisecond
	}
	timeout := 2 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	size := req.Size
	var err error
	req.Host, err = normalizeHost(req.Host)
	if err != nil {
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}
	cfg, err := normalizePingConfig(req.Count, interval, timeout, size, req.IPv6, req.Continuous)
	if err != nil {
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}

	rep, err := ping.RunStreamContext(ctx, req.Host, cfg, func(res report.Result) error {
		ts := time.Now()
		storeResult(user.ID, "ping", req.Host, res.Addr, 0, res, ts)
		jr := toJSONResult(res)
		return writer.WriteEvent(wsEvent{
			Type:     "result",
			Protocol: "icmp",
			Target:   req.Host,
			Ts:       ts.UnixMilli(),
			Result:   &jr,
		})
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			summary := toJSONSummary(rep.Summary)
			return writer.WriteEvent(wsEvent{
				Type:     "stopped",
				Protocol: "icmp",
				Target:   req.Host,
				Addr:     rep.Addr,
				Summary:  &summary,
			})
		}
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}

	summary := toJSONSummary(rep.Summary)
	enqueueAlertEvaluation(user, "ping", req.Host, 0, rep.Summary, buildConfiguredDashboardURL("alerts"))
	return writer.WriteEvent(wsEvent{
		Type:     "summary",
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Summary:  &summary,
	})
}

func streamTCPPingWS(ctx context.Context, writer *wsWriter, user store.User, req wsRequest) error {
	release, ok := tryAcquireProbeSlot()
	if !ok {
		return writer.WriteEvent(wsEvent{Type: "error", Error: "server busy, try again"})
	}
	defer release()

	host := req.Host
	port := req.Port
	if parsedHost, parsedPort, ok := splitHostPort(host); ok {
		host = parsedHost
		if port == 0 {
			port = parsedPort
		}
	}
	if port == 0 {
		port = 80
	}

	interval := time.Second
	if req.IntervalMs > 0 {
		interval = time.Duration(req.IntervalMs) * time.Millisecond
	}
	timeout := 2 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}

	host, port, cfg, err := normalizeTCPPingRequest(host, port, req.Count, interval, timeout, req.Continuous)
	if err != nil {
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}

	rep, err := tcpping.RunStreamContext(ctx, host, port, cfg, func(res report.Result) error {
		ts := time.Now()
		storeResult(user.ID, "tcpping", host, res.Addr, port, res, ts)
		jr := toJSONResult(res)
		return writer.WriteEvent(wsEvent{
			Type:     "result",
			Protocol: "tcp",
			Target:   host,
			Port:     port,
			Ts:       ts.UnixMilli(),
			Result:   &jr,
		})
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			summary := toJSONSummary(rep.Summary)
			return writer.WriteEvent(wsEvent{
				Type:     "stopped",
				Protocol: "tcp",
				Target:   host,
				Addr:     rep.Addr,
				Port:     port,
				Summary:  &summary,
			})
		}
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}

	summary := toJSONSummary(rep.Summary)
	enqueueAlertEvaluation(user, "tcpping", host, port, rep.Summary, buildConfiguredDashboardURL("alerts"))
	return writer.WriteEvent(wsEvent{
		Type:     "summary",
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Port:     rep.Port,
		Summary:  &summary,
	})
}

func streamPortScanWS(ctx context.Context, writer *wsWriter, req wsRequest) error {
	release, ok := tryAcquireProbeSlot()
	if !ok {
		return writer.WriteEvent(wsEvent{Type: "error", Error: "server busy, try again"})
	}
	defer release()

	timeout := defaultProbeTimeout
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}

	host, _, cfg, err := normalizePortScanRequest(req.Host, req.Ports, timeout, req.Concurrency)
	if err != nil {
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}

	rep, err := portscan.RunStreamContext(ctx, host, cfg, func(res portscan.Result) error {
		ts := time.Now()
		jr := toJSONPortScanResult(res)
		return writer.WriteEvent(wsEvent{
			Type:     "result",
			Protocol: "tcp-portscan",
			Target:   host,
			Ts:       ts.UnixMilli(),
			Result:   &jr,
		})
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			summary := toJSONPortScanSummary(rep.Summary)
			return writer.WriteEvent(wsEvent{
				Type:     "stopped",
				Protocol: rep.Protocol,
				Target:   rep.Target,
				Addr:     rep.Addr,
				Summary:  &summary,
			})
		}
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}

	summary := toJSONPortScanSummary(rep.Summary)
	return writer.WriteEvent(wsEvent{
		Type:     "summary",
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Summary:  &summary,
	})
}

func tryAcquireProbeSlot() (func(), bool) {
	select {
	case probeLimiter <- struct{}{}:
		return func() { <-probeLimiter }, true
	default:
		rejected := rejectedProbeRequests.Add(1)
		if rejected == 1 || rejected%100 == 0 {
			fmt.Fprintf(os.Stderr, "probe limiter rejected %d request(s)\n", rejected)
		}
		return nil, false
	}
}

func isAllowedRequestOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	return isAllowedWSOrigin(r, origin)
}

func isAllowedWSOrigin(r *http.Request, origin string) bool {
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}

	originHost, originPort := splitHostAndOptionalPort(u.Host)
	requestHost, requestPort := splitHostAndOptionalPort(r.Host)
	if originPort == "" {
		originPort = defaultPortForScheme(u.Scheme)
	}
	if requestPort == "" {
		requestPort = defaultPortForRequest(r)
	}

	if hostsEquivalentForOrigin(originHost, requestHost) && originPort == requestPort {
		return true
	}

	if len(appAllowedOrigins) == 0 {
		return false
	}

	normalized, err := normalizeOrigin(origin)
	if err != nil {
		return false
	}
	_, ok := appAllowedOrigins[normalized]
	return ok
}

func parseAllowedOrigins(raw string) (map[string]struct{}, error) {
	origins := make(map[string]struct{})
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		normalized, err := normalizeOrigin(item)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed origin %q: %w", item, err)
		}
		origins[normalized] = struct{}{}
	}
	return origins, nil
}

func parseAdminEmails(raw string) (map[string]struct{}, error) {
	adminEmails := make(map[string]struct{})
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		email, err := store.NormalizeEmail(item)
		if err != nil {
			return nil, fmt.Errorf("invalid admin email %q: %w", item, err)
		}
		adminEmails[email] = struct{}{}
	}
	return adminEmails, nil
}

func hostsEquivalentForOrigin(left, right string) bool {
	if strings.EqualFold(left, right) {
		return true
	}
	return isLoopbackOriginHost(left) && isLoopbackOriginHost(right)
}

func isLoopbackOriginHost(raw string) bool {
	host := strings.TrimSpace(strings.Trim(raw, "[]"))
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func normalizeOrigin(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("scheme and host are required")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("query and fragment are not allowed")
	}
	if path := strings.TrimSpace(u.Path); path != "" && path != "/" {
		return "", fmt.Errorf("path is not allowed")
	}

	host, port := splitHostAndOptionalPort(u.Host)
	if host == "" {
		return "", fmt.Errorf("host is required")
	}
	if port == "" {
		port = defaultPortForScheme(u.Scheme)
	}
	if port == "" {
		return "", fmt.Errorf("port is required")
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(net.JoinHostPort(host, port)), nil
}

func splitHostAndOptionalPort(raw string) (string, string) {
	if raw == "" {
		return "", ""
	}
	if host, port, err := net.SplitHostPort(raw); err == nil {
		return host, port
	}
	return raw, ""
}

func defaultPortForScheme(scheme string) string {
	switch strings.ToLower(scheme) {
	case "https", "wss":
		return "443"
	case "http", "ws":
		return "80"
	default:
		return ""
	}
}

func defaultPortForRequest(r *http.Request) string {
	if r.TLS != nil {
		return "443"
	}
	return "80"
}

func (rl *rateLimiter) Allow(key string, cfg rateLimitConfig) (bool, time.Duration) {
	if rl == nil {
		return true, 0
	}
	now := time.Now()
	cutoff := now.Add(-cfg.Window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if len(rl.buckets) > maxRateLimitBuckets || now.Sub(rl.lastSweep) >= rateLimitSweepEvery {
		rl.sweep(now)
	}

	bucket := rl.buckets[key]
	if bucket == nil {
		bucket = &rateLimitBucket{}
		rl.buckets[key] = bucket
	}

	filtered := bucket.hits[:0]
	for _, hit := range bucket.hits {
		if hit.After(cutoff) {
			filtered = append(filtered, hit)
		}
	}
	bucket.hits = filtered
	bucket.lastSeen = now

	if len(bucket.hits) >= cfg.Limit {
		retryAfter := bucket.hits[0].Add(cfg.Window).Sub(now)
		if retryAfter < time.Second {
			retryAfter = time.Second
		}
		return false, retryAfter
	}

	bucket.hits = append(bucket.hits, now)
	return true, 0
}

func (rl *rateLimiter) sweep(now time.Time) {
	cutoff := now.Add(-rateLimitBucketTTL)
	for key, bucket := range rl.buckets {
		filtered := bucket.hits[:0]
		for _, hit := range bucket.hits {
			if hit.After(cutoff) {
				filtered = append(filtered, hit)
			}
		}
		bucket.hits = filtered
		if len(bucket.hits) == 0 && bucket.lastSeen.Before(cutoff) {
			delete(rl.buckets, key)
		}
	}
	rl.lastSweep = now
}

func enforceRateLimit(w http.ResponseWriter, r *http.Request, action, subject string, cfg rateLimitConfig) bool {
	key := buildRateLimitKey(r, action, subject)
	allowed, retryAfter := authRateLimiter.Allow(key, cfg)
	if allowed {
		return true
	}
	w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
	writeJSON(w, http.StatusTooManyRequests, apiError{Error: "too many requests, try again later"})
	return false
}

func buildRateLimitKey(r *http.Request, action, subject string) string {
	ip := clientIPFromRequest(r)
	subject = strings.ToLower(strings.TrimSpace(subject))
	if subject == "" {
		return action + "|" + ip
	}
	return action + "|" + ip + "|" + subject
}

func clientIPFromRequest(r *http.Request) string {
	if cfIP := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cfIP != "" {
		return cfIP
	}
	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("empty body")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			return fmt.Errorf("request body too large")
		}
		return err
	}
	var extra struct{}
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("request body must contain a single JSON object")
	}
	return nil
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func toJSONReport(rep report.Report) jsonReport {
	jr := jsonReport{
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Port:     rep.Port,
		Summary:  toJSONSummary(rep.Summary),
	}

	jr.Results = make([]jsonResult, 0, len(rep.Results))
	for _, r := range rep.Results {
		jr.Results = append(jr.Results, toJSONResult(r))
	}

	return jr
}

func toJSONPortScanReport(rep portscan.Report) jsonReport {
	jr := jsonReport{
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Summary:  toJSONPortScanSummary(rep.Summary),
	}

	jr.Results = make([]jsonResult, 0, len(rep.Results))
	for _, r := range rep.Results {
		jr.Results = append(jr.Results, toJSONPortScanResult(r))
	}

	return jr
}

func toJSONSummary(s report.Summary) jsonSummary {
	js := jsonSummary{
		Sent:       s.Sent,
		Recv:       s.Recv,
		Loss:       s.Loss,
		LossStreak: s.LossStreak,
	}
	if s.Recv > 0 {
		js.Min = formatDuration(s.Min)
		js.Avg = formatDuration(s.Avg)
		js.Max = formatDuration(s.Max)
		js.Median = formatDuration(s.Median)
		js.StdDev = formatDuration(s.StdDev)
		js.Jitter = formatDuration(s.Jitter)
	}
	if s.Duration > 0 {
		js.Duration = formatDuration(s.Duration)
	}
	return js
}

func toJSONPortScanSummary(s portscan.Summary) jsonSummary {
	return jsonSummary{
		Scanned:  s.Scanned,
		Open:     s.Open,
		Closed:   s.Closed,
		Timeout:  s.Timeout,
		Duration: s.Duration.Round(time.Millisecond).String(),
	}
}

func toJSONResult(r report.Result) jsonResult {
	jr := jsonResult{Seq: r.Seq}
	if r.Error != "" {
		jr.Error = r.Error
	} else {
		jr.RTT = formatDuration(r.RTT)
		ms := float64(r.RTT) / float64(time.Millisecond)
		jr.RTTMs = &ms
	}
	if r.Addr != "" {
		jr.Addr = r.Addr
	}
	return jr
}

func toJSONPortScanResult(r portscan.Result) jsonResult {
	jr := jsonResult{
		Port:  r.Port,
		State: r.State,
		Error: r.Error,
		Addr:  r.Addr,
	}
	if r.State == "open" {
		jr.RTT = formatDuration(r.RTT)
		ms := float64(r.RTT) / float64(time.Millisecond)
		jr.RTTMs = &ms
	}
	return jr
}

func storeResult(userID int64, protocol, target, addr string, port int, res report.Result, ts time.Time) {
	if appStore == nil {
		return
	}
	rec := store.ResultRecord{
		UserID:   userID,
		Ts:       ts,
		Protocol: protocol,
		Target:   target,
		Addr:     addr,
		Port:     port,
		Seq:      res.Seq,
		Error:    res.Error,
	}
	if res.Error == "" {
		ms := float64(res.RTT) / float64(time.Millisecond)
		rec.RttMs = &ms
	}
	appStore.InsertAsync(rec)
}
