package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
)

func TestSetupExportsToConfiguredCollector(t *testing.T) {
	var mu sync.Mutex
	hit := map[string]bool{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hit[r.URL.Path] = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx := context.Background()
	shutdown, err := Setup(ctx, Config{Endpoint: srv.URL, ServiceVersion: "test"})
	if err != nil {
		t.Fatalf("Setup returned error: %v", err)
	}

	_, span := otel.Tracer(ServiceName).Start(ctx, "test-span")
	span.End()
	RecordProbe(ctx, "ping", true, 5*time.Millisecond)

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown returned error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !hit["/v1/traces"] {
		t.Errorf("expected an export to /v1/traces, got requests: %v", hit)
	}
	if !hit["/v1/metrics"] {
		t.Errorf("expected an export to /v1/metrics, got requests: %v", hit)
	}
}

func TestSetupDisabledWhenEndpointEmpty(t *testing.T) {
	shutdown, err := Setup(context.Background(), Config{})
	if err != nil {
		t.Fatalf("Setup returned error: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("no-op shutdown returned error: %v", err)
	}
}

func TestSplitEndpoint(t *testing.T) {
	cases := []struct {
		endpoint     string
		wantHost     string
		wantInsecure bool
		wantErr      bool
	}{
		{"localhost:4318", "localhost:4318", true, false},
		{"http://localhost:4318", "localhost:4318", true, false},
		{"https://collector.example.com", "collector.example.com", false, false},
		{"ftp://bad.example.com", "", false, true},
	}
	for _, tc := range cases {
		host, insecure, err := splitEndpoint(tc.endpoint)
		if tc.wantErr {
			if err == nil {
				t.Errorf("splitEndpoint(%q): expected error, got none", tc.endpoint)
			}
			continue
		}
		if err != nil {
			t.Errorf("splitEndpoint(%q): unexpected error: %v", tc.endpoint, err)
			continue
		}
		if host != tc.wantHost || insecure != tc.wantInsecure {
			t.Errorf("splitEndpoint(%q) = (%q, %v), want (%q, %v)", tc.endpoint, host, insecure, tc.wantHost, tc.wantInsecure)
		}
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("NETWATCHER_OTEL_ENDPOINT", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("NETWATCHER_OTEL_SERVICE_VERSION", "")

	if cfg := ConfigFromEnv(); cfg.Endpoint != "" {
		t.Fatalf("expected empty endpoint by default, got %q", cfg.Endpoint)
	}

	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://standard:4318")
	if cfg := ConfigFromEnv(); cfg.Endpoint != "http://standard:4318" {
		t.Fatalf("expected standard OTEL_EXPORTER_OTLP_ENDPOINT to be honored, got %q", cfg.Endpoint)
	}

	t.Setenv("NETWATCHER_OTEL_ENDPOINT", "http://project-specific:4318")
	if cfg := ConfigFromEnv(); cfg.Endpoint != "http://project-specific:4318" {
		t.Fatalf("expected NETWATCHER_OTEL_ENDPOINT to take precedence, got %q", cfg.Endpoint)
	}
}
