package objectstore

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOpenWithEmptyEndpointIsANoop(t *testing.T) {
	c, err := Open(Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c != nil {
		t.Fatalf("expected a nil *Client when no endpoint is configured, got %+v", c)
	}
}

func TestOpenRequiresBucketWhenEndpointIsSet(t *testing.T) {
	_, err := Open(Config{Endpoint: "s3.example.com"})
	if err == nil {
		t.Fatal("expected an error when an endpoint is set without a bucket")
	}
}

// A nil *Client is what callers get back whenever object storage isn't
// configured (Open returns nil, nil above); the upload/URL methods must
// report a clear "not configured" error rather than panicking.
func TestNilClientMethodsReportNotConfigured(t *testing.T) {
	var c *Client

	if _, err := c.UploadJSON(context.Background(), "key", map[string]string{"a": "b"}); err == nil {
		t.Error("UploadJSON() on nil client = nil error, want an error")
	}
	if _, err := c.UploadCSV(context.Background(), "key", [][]string{{"a", "b"}}); err == nil {
		t.Error("UploadCSV() on nil client = nil error, want an error")
	}
	if _, err := c.PresignedGetURL(context.Background(), "key", 0); err == nil {
		t.Error("PresignedGetURL() on nil client = nil error, want an error")
	}
	if err := c.Ping(context.Background()); err != nil {
		t.Errorf("Ping() on nil client = %v, want nil", err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("Close() on nil client = %v, want nil", err)
	}
}

func TestObjectKeyPrefixing(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
		key    string
		want   string
	}{
		{"no prefix", "", "exports/report.json", "exports/report.json"},
		{"with prefix", "qnetward", "exports/report.json", "qnetward/exports/report.json"},
		{"prefix with slashes", "/qnetward/", "exports/report.json", "qnetward/exports/report.json"},
		{"leading slash on key is stripped", "qnetward", "/exports/report.json", "qnetward/exports/report.json"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{cfg: Config{Prefix: tc.prefix}}
			if got := c.objectKey(tc.key); got != tc.want {
				t.Fatalf("objectKey(%q) with prefix %q = %q, want %q", tc.key, tc.prefix, got, tc.want)
			}
		})
	}
}

// TestUploadThenPresignedGetURLRoundTrips exercises the exact sequence
// cmd/qnetward/history_handlers.go uses: upload a CSV, then presign a GET URL
// for the key UploadCSV returned. That key already has Config.Prefix
// applied, so PresignedGetURL must use it as-is rather than re-applying
// the prefix (which previously produced a URL pointing at a key that was
// never written, and a 404 on download).
func TestUploadThenPresignedGetURLRoundTrips(t *testing.T) {
	endpoint := strings.TrimSpace(os.Getenv("QNETWARD_TEST_S3_ENDPOINT"))
	if endpoint == "" {
		t.Skip("QNETWARD_TEST_S3_ENDPOINT is not set")
	}

	c, err := Open(Config{
		Endpoint:  endpoint,
		AccessKey: envOr("QNETWARD_TEST_S3_ACCESS_KEY", "qnetward"),
		SecretKey: envOr("QNETWARD_TEST_S3_SECRET_KEY", "change-me-too"),
		Bucket:    envOr("QNETWARD_TEST_S3_BUCKET", "qnetward-artifacts"),
		Prefix:    "qnetward",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	ctx := context.Background()
	rows := [][]string{{"seq", "rtt_ms"}, {"1", "12.5"}}
	objectKey, err := c.UploadCSV(ctx, "exports/roundtrip-test.csv", rows)
	if err != nil {
		t.Fatalf("UploadCSV: %v", err)
	}
	if objectKey != "qnetward/exports/roundtrip-test.csv" {
		t.Fatalf("UploadCSV returned key %q, want the prefixed key", objectKey)
	}

	url, err := c.PresignedGetURL(ctx, objectKey, time.Minute)
	if err != nil {
		t.Fatalf("PresignedGetURL: %v", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET presigned URL: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET presigned URL: status %d, body: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "12.5") {
		t.Fatalf("downloaded body missing expected content: %s", body)
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
