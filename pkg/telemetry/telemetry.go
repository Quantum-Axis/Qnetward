// Package telemetry wires up OpenTelemetry tracing and metrics for
// NetWatcher. It is safe to import and use unconditionally: instruments and
// spans created via the global otel.Tracer/otel.Meter API are no-ops until
// Setup installs a real exporter, and Setup itself is a no-op (returns a
// no-op shutdown, touches no global state) when no OTLP endpoint is
// configured, so an operator who hasn't stood up a collector sees no
// behavior change at all.
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// ServiceName is used as the OpenTelemetry resource's service.name
// attribute, and as the default value for OTEL_SERVICE_NAME.
const ServiceName = "netwatcher"

// Config controls whether and how telemetry is exported. An empty
// Endpoint disables export entirely: Setup then leaves the global
// tracer/meter providers as OpenTelemetry's built-in no-ops.
type Config struct {
	// Endpoint is an OTLP/HTTP collector address, e.g. "localhost:4318" or
	// "https://otel-collector.example.com". A bare host:port is treated as
	// insecure (plain HTTP), matching most local collector setups; an
	// explicit "https://" scheme uses TLS.
	Endpoint string
	// ServiceVersion is reported as the service.version resource
	// attribute. Optional.
	ServiceVersion string
}

// ConfigFromEnv reads Config from the standard OTel_EXPORTER_OTLP_ENDPOINT
// environment variable (falling back to NETWATCHER_OTEL_ENDPOINT for
// consistency with this project's other NETWATCHER_* settings), plus
// NETWATCHER_OTEL_SERVICE_VERSION.
func ConfigFromEnv() Config {
	endpoint := strings.TrimSpace(os.Getenv("NETWATCHER_OTEL_ENDPOINT"))
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	}
	return Config{
		Endpoint:       endpoint,
		ServiceVersion: strings.TrimSpace(os.Getenv("NETWATCHER_OTEL_SERVICE_VERSION")),
	}
}

// Shutdown flushes and closes the telemetry pipeline set up by Setup. It is
// always safe to call, including the no-op returned when telemetry is
// disabled.
type Shutdown func(context.Context) error

// Setup configures global OpenTelemetry tracing and metrics from cfg. When
// cfg.Endpoint is empty, it does nothing and returns a no-op Shutdown.
func Setup(ctx context.Context, cfg Config) (Shutdown, error) {
	if cfg.Endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	host, insecure, err := splitEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("telemetry: %w", err)
	}

	res, err := buildResource(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("telemetry: build resource: %w", err)
	}

	traceOpts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(host)}
	metricOpts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(host)}
	if insecure {
		traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
		metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
	}

	traceExporter, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return nil, fmt.Errorf("telemetry: create trace exporter: %w", err)
	}

	metricExporter, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		return nil, fmt.Errorf("telemetry: create metric exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter)),
		metric.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(shutdownCtx context.Context) error {
		return errors.Join(
			tracerProvider.Shutdown(shutdownCtx),
			meterProvider.Shutdown(shutdownCtx),
		)
	}, nil
}

func buildResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	version := cfg.ServiceVersion
	if version == "" {
		version = "dev"
	}
	return resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(ServiceName),
			semconv.ServiceVersion(version),
		),
		resource.WithFromEnv(),
		resource.WithHost(),
		resource.WithProcess(),
	)
}

// splitEndpoint strips a scheme from endpoint (if any) and reports whether
// the connection should be made without TLS. A bare host:port with no
// scheme is treated as insecure, since that's the overwhelmingly common
// case for a collector running as a local sidecar.
func splitEndpoint(endpoint string) (host string, insecure bool, err error) {
	if !strings.Contains(endpoint, "://") {
		return endpoint, true, nil
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", false, fmt.Errorf("invalid OTLP endpoint %q: %w", endpoint, err)
	}
	switch u.Scheme {
	case "http":
		return u.Host, true, nil
	case "https":
		return u.Host, false, nil
	default:
		return "", false, fmt.Errorf("unsupported OTLP endpoint scheme %q", u.Scheme)
	}
}
