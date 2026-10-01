package telemetry

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// meter is created against the global MeterProvider proxy, so it's safe to
// call before Setup runs: otel upgrades it in place once a real provider is
// installed, and it stays a harmless no-op if telemetry is never enabled.
var meter = otel.Meter(ServiceName)

var (
	probesTotal     metric.Int64Counter
	probeDuration   metric.Float64Histogram
	alertsEvaluated metric.Int64Counter
	alertsTriggered metric.Int64Counter
)

func init() {
	var err error
	probesTotal, err = meter.Int64Counter(
		"netwatcher.probes.total",
		metric.WithDescription("Number of network probes executed"),
		metric.WithUnit("{probe}"),
	)
	if err != nil {
		log.Printf("telemetry: create probes.total counter: %v", err)
	}
	probeDuration, err = meter.Float64Histogram(
		"netwatcher.probe.duration",
		metric.WithDescription("Duration of a single network probe request"),
		metric.WithUnit("s"),
	)
	if err != nil {
		log.Printf("telemetry: create probe.duration histogram: %v", err)
	}
	alertsEvaluated, err = meter.Int64Counter(
		"netwatcher.alerts.evaluated",
		metric.WithDescription("Number of alert rule evaluations"),
		metric.WithUnit("{evaluation}"),
	)
	if err != nil {
		log.Printf("telemetry: create alerts.evaluated counter: %v", err)
	}
	alertsTriggered, err = meter.Int64Counter(
		"netwatcher.alerts.triggered",
		metric.WithDescription("Number of alert rule evaluations that produced a triggered or recovered decision"),
		metric.WithUnit("{alert}"),
	)
	if err != nil {
		log.Printf("telemetry: create alerts.triggered counter: %v", err)
	}
}

// RecordProbe records that one network probe of the given protocol
// finished, successfully or not, after d.
func RecordProbe(ctx context.Context, protocol string, success bool, d time.Duration) {
	attrs := metric.WithAttributes(
		attribute.String("protocol", protocol),
		attribute.Bool("success", success),
	)
	probesTotal.Add(ctx, 1, attrs)
	probeDuration.Record(ctx, d.Seconds(), attrs)
}

// RecordAlertEvaluation records that a set of alert rules was evaluated for
// one probe result, and how many of them produced a triggered or recovered
// decision.
func RecordAlertEvaluation(ctx context.Context, protocol string, decisionCount int) {
	attrs := metric.WithAttributes(attribute.String("protocol", protocol))
	alertsEvaluated.Add(ctx, 1, attrs)
	if decisionCount > 0 {
		alertsTriggered.Add(ctx, int64(decisionCount), attrs)
	}
}
