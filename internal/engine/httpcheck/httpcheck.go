// Package httpcheck fetches a URL once and reports the status, the time to a
// complete response, and the TLS certificate's expiry.
package httpcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

const (
	DefaultTimeout = 10 * time.Second
	MaxURLLength   = 2048
	maxRedirects   = 5
	// maxBodyBytes is how much of the body is read before the connection is
	// closed. The body itself is not inspected.
	maxBodyBytes = 64 << 10
	userAgent    = "Qnetward-Monitor/1.0 (+https://github.com/Quantum-Axis/Qnetward)"
)

type Config struct {
	Timeout time.Duration
	// DialControl, when set, vets each address right before connecting. It is
	// called for redirects too, so a redirect can't reach a blocked address.
	DialControl func(network, address string, c syscall.RawConn) error
	// RootCAs overrides the system roots; tests use it for their own server.
	RootCAs *x509.CertPool
}

type Result struct {
	URL          string
	StatusCode   int
	Duration     time.Duration
	TLSExpiresAt *time.Time
	TLSIssuer    string
}

// NormalizeURL accepts a URL with or without a scheme and returns it with
// http or https. Credentials in the URL are refused, because monitors are
// stored and shown to the whole workspace.
func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("URL is required")
	}
	if len(raw) > MaxURLLength {
		return "", fmt.Errorf("URL must be %d characters or fewer", MaxURLLength)
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("URL must start with http:// or https://")
	}
	if u.User != nil {
		return "", errors.New("URL must not contain a username or password")
	}
	if u.Hostname() == "" {
		return "", errors.New("URL must include a host")
	}
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	return u.String(), nil
}

// Run requests rawURL with GET, following up to five redirects.
func Run(ctx context.Context, rawURL string, cfg Config) (Result, error) {
	target, err := NormalizeURL(rawURL)
	if err != nil {
		return Result{}, err
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dialer := &net.Dialer{Timeout: timeout, Control: cfg.DialControl}
	transport := &http.Transport{
		// No proxy: the dial check must see the real destination.
		Proxy:                  nil,
		DialContext:            dialer.DialContext,
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: cfg.RootCAs},
		TLSHandshakeTimeout:    timeout,
		ResponseHeaderTimeout:  timeout,
		DisableKeepAlives:      true,
		MaxResponseHeaderBytes: 64 << 10,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return Result{URL: target}, cleanError(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))

	result := Result{URL: target, StatusCode: resp.StatusCode, Duration: time.Since(start)}
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		leaf := resp.TLS.PeerCertificates[0]
		expires := leaf.NotAfter.UTC()
		result.TLSExpiresAt = &expires
		result.TLSIssuer = leaf.Issuer.CommonName
	}
	return result, nil
}

// cleanError drops the "Get \"url\":" prefix net/http adds; the URL is
// already shown next to the error.
func cleanError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
