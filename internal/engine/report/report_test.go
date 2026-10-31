package report

import (
	"testing"
	"time"
)

func TestSummarizeIncludesAdvancedMetrics(t *testing.T) {
	results := []Result{
		{Seq: 1, RTT: 10 * time.Millisecond},
		{Seq: 2, RTT: 20 * time.Millisecond},
		{Seq: 3, Error: "timeout"},
		{Seq: 4, RTT: 40 * time.Millisecond},
		{Seq: 5, Error: "timeout"},
		{Seq: 6, Error: "timeout"},
	}

	summary := Summarize(results, len(results))

	if summary.Sent != 6 {
		t.Fatalf("expected sent=6, got %d", summary.Sent)
	}
	if summary.Recv != 3 {
		t.Fatalf("expected recv=3, got %d", summary.Recv)
	}
	if summary.Min != 10*time.Millisecond {
		t.Fatalf("expected min=10ms, got %s", summary.Min)
	}
	if summary.Max != 40*time.Millisecond {
		t.Fatalf("expected max=40ms, got %s", summary.Max)
	}
	if summary.Median != 20*time.Millisecond {
		t.Fatalf("expected median=20ms, got %s", summary.Median)
	}
	if summary.Jitter != 15*time.Millisecond {
		t.Fatalf("expected jitter=15ms, got %s", summary.Jitter)
	}
	if summary.LossStreak != 2 {
		t.Fatalf("expected loss streak=2, got %d", summary.LossStreak)
	}
}
