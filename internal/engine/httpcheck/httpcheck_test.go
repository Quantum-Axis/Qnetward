package httpcheck

import (
	"context"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"
)

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"example.com":                  "https://example.com",
		"HTTP://Example.com/health#x":  "http://example.com/health",
		" https://api.example.com/v1 ": "https://api.example.com/v1",
	}
	for in, want := range cases {
		got, err := NormalizeURL(in)
		if err != nil || got != want {
			t.Errorf("NormalizeURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "ftp://example.com", "https://user:pass@example.com", "https://"} {
		if _, err := NormalizeURL(bad); err == nil {
			t.Errorf("NormalizeURL(%q) should fail", bad)
		}
	}
}

func TestRunReportsStatusAndTLSExpiry(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/moved" {
			http.Redirect(w, r, "/ok", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	// The test server's certificate is self-signed, so verification fails.
	if _, err := Run(context.Background(), srv.URL+"/moved", Config{Timeout: 5 * time.Second}); err == nil {
		t.Fatal("expected an untrusted certificate to fail the check")
	}
	roots := x509.NewCertPool()
	roots.AddCert(srv.Certificate())
	secure, err := Run(context.Background(), srv.URL+"/moved", Config{Timeout: 5 * time.Second, RootCAs: roots})
	if err != nil {
		t.Fatalf("Run with trusted roots returned error: %v", err)
	}
	if secure.StatusCode != http.StatusTeapot || secure.TLSExpiresAt == nil || !secure.TLSExpiresAt.Equal(srv.Certificate().NotAfter.UTC()) {
		t.Fatalf("expected status and certificate expiry after the redirect, got %#v", secure)
	}

	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/moved" {
			http.Redirect(w, r, "/ok", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusTeapot)
	}))
	defer plain.Close()
	res, err := Run(context.Background(), plain.URL+"/moved", Config{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.StatusCode != http.StatusTeapot || res.Duration <= 0 || res.TLSExpiresAt != nil {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func TestRunHonoursDialControl(t *testing.T) {
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer plain.Close()
	blocked := errors.New("blocked")
	_, err := Run(context.Background(), plain.URL, Config{
		Timeout:     5 * time.Second,
		DialControl: func(string, string, syscall.RawConn) error { return blocked },
	})
	if !errors.Is(err, blocked) {
		t.Fatalf("expected the dial check to stop the request, got %v", err)
	}
}
