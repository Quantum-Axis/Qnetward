package objectstore

import (
	"context"
	"testing"
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
		{"with prefix", "netwatcher", "exports/report.json", "netwatcher/exports/report.json"},
		{"prefix with slashes", "/netwatcher/", "exports/report.json", "netwatcher/exports/report.json"},
		{"leading slash on key is stripped", "netwatcher", "/exports/report.json", "netwatcher/exports/report.json"},
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
