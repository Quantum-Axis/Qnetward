package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"qnetward/internal/platform/mailer"
	"qnetward/internal/store"
)

func newPingFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("ping", flag.ContinueOnError)
	fs.Bool("ipv6", false, "")
	fs.Int("count", defaultProbeCount, "")
	return fs
}

func newTCPPingFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("tcpping", flag.ContinueOnError)
	fs.Int("port", defaultTCPPort, "")
	return fs
}

func openTestStore(t *testing.T) *store.Store {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("QNETWARD_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("QNETWARD_TEST_POSTGRES_DSN is not set")
	}

	st, err := store.OpenConfigured(store.Config{
		DSN:           dsn,
		BatchSize:     1,
		FlushInterval: 5 * time.Millisecond,
		TableName:     testTableName(t.Name()),
	})
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	return st
}

func installTestStore(t *testing.T) *store.Store {
	t.Helper()

	st := openTestStore(t)
	oldStore := appStore
	oldCache := appCache
	oldObjectStore := appObjectStore
	appStore = st
	appCache = nil
	appObjectStore = nil
	t.Cleanup(func() {
		appStore = oldStore
		appCache = oldCache
		appObjectStore = oldObjectStore
	})
	return st
}

func installFailingTestMailer(t *testing.T) {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"mailer unavailable"}`))
	}))
	t.Cleanup(srv.Close)

	client, err := mailer.Open(mailer.Config{
		APIKey:   "test-key",
		From:     "Qnetward <alerts@example.com>",
		ReplyTo:  "support@example.com",
		Endpoint: srv.URL,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("open test mailer: %v", err)
	}

	previous := appMailer
	appMailer = client
	t.Cleanup(func() {
		appMailer = previous
	})
}

func issueTestSessionCookie(t *testing.T, st *store.Store) *http.Cookie {
	t.Helper()
	_, cookie := issueTestSession(t, st)
	return cookie
}

func issueTestSession(t *testing.T, st *store.Store) (store.User, *http.Cookie) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	email := fmt.Sprintf("session_%d@example.com", time.Now().UnixNano())
	user, err := st.CreateUser(ctx, "Session Test User", email, "test-password-hash", false)
	if err != nil {
		t.Fatalf("create session test user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1", nil)
	rr := httptest.NewRecorder()
	if err := issueSession(rr, req, ctx, user); err != nil {
		t.Fatalf("issue session: %v", err)
	}

	return user, extractAuthCookie(t, rr.Result())
}

func resetAuthRateLimiterForTest(t *testing.T) {
	t.Helper()

	previous := authRateLimiter
	authRateLimiter = newRateLimiter()
	t.Cleanup(func() {
		authRateLimiter = previous
	})
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func float64Ptr(v float64) *float64 {
	return &v
}

func fillProbeLimiter() {
	for i := 0; i < maxConcurrentProbes; i++ {
		probeLimiter <- struct{}{}
	}
}

func drainProbeLimiter() {
	for {
		select {
		case <-probeLimiter:
		default:
			return
		}
	}
}

func extractAuthCookie(t *testing.T, resp *http.Response) *http.Cookie {
	t.Helper()

	for _, cookie := range resp.Cookies() {
		if cookie.Name == authCookieName {
			return cookie
		}
	}
	t.Fatal("expected auth cookie")
	return nil
}

func testTableName(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	b.WriteString("test_")
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	b.WriteString("_")
	b.WriteString(fmt.Sprintf("%d", time.Now().UnixNano()))
	return b.String()
}
