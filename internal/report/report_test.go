package report

import (
	"testing"
	"time"
)

func TestSummarizeAllSuccess(t *testing.T) {
	results := []Result{
		{Seq: 1, RTT: 10 * time.Millisecond},
		{Seq: 2, RTT: 20 * time.Millisecond},
		{Seq: 3, RTT: 30 * time.Millisecond},
	}

	s := Summarize(results, 3)

	if s.Sent != 3 {
		t.Fatalf("Sent = %d, want 3", s.Sent)
	}
	if s.Recv != 3 {
		t.Fatalf("Recv = %d, want 3", s.Recv)
	}
	if s.Loss != 0 {
		t.Fatalf("Loss = %v, want 0", s.Loss)
	}
	if s.Min != 10*time.Millisecond {
		t.Fatalf("Min = %v, want 10ms", s.Min)
	}
	if s.Max != 30*time.Millisecond {
		t.Fatalf("Max = %v, want 30ms", s.Max)
	}
	if s.Avg != 20*time.Millisecond {
		t.Fatalf("Avg = %v, want 20ms", s.Avg)
	}
}

func TestSummarizeAllFailure(t *testing.T) {
	results := []Result{
		{Seq: 1, Error: "timeout"},
		{Seq: 2, Error: "timeout"},
	}

	s := Summarize(results, 2)

	if s.Recv != 0 {
		t.Fatalf("Recv = %d, want 0", s.Recv)
	}
	if s.Loss != 100 {
		t.Fatalf("Loss = %v, want 100", s.Loss)
	}
	if s.Min != 0 || s.Max != 0 || s.Avg != 0 || s.StdDev != 0 {
		t.Fatalf("expected zero-value RTT stats when nothing was received, got %+v", s)
	}
}

func TestSummarizeMixedResults(t *testing.T) {
	results := []Result{
		{Seq: 1, RTT: 10 * time.Millisecond},
		{Seq: 2, Error: "timeout"},
		{Seq: 3, RTT: 30 * time.Millisecond},
		{Seq: 4, Error: "timeout"},
	}

	s := Summarize(results, 4)

	if s.Sent != 4 {
		t.Fatalf("Sent = %d, want 4", s.Sent)
	}
	if s.Recv != 2 {
		t.Fatalf("Recv = %d, want 2", s.Recv)
	}
	if s.Loss != 50 {
		t.Fatalf("Loss = %v, want 50", s.Loss)
	}
	if s.Min != 10*time.Millisecond {
		t.Fatalf("Min = %v, want 10ms", s.Min)
	}
	if s.Max != 30*time.Millisecond {
		t.Fatalf("Max = %v, want 30ms", s.Max)
	}
}

func TestSummarizeSingleResult(t *testing.T) {
	results := []Result{{Seq: 1, RTT: 15 * time.Millisecond}}

	s := Summarize(results, 1)

	if s.Min != 15*time.Millisecond || s.Max != 15*time.Millisecond || s.Avg != 15*time.Millisecond {
		t.Fatalf("expected min=avg=max=15ms for a single sample, got %+v", s)
	}
	if s.StdDev != 0 {
		t.Fatalf("StdDev = %v, want 0 for a single sample", s.StdDev)
	}
}

func TestSummarizeZeroSent(t *testing.T) {
	s := Summarize(nil, 0)

	if s.Sent != 0 || s.Recv != 0 {
		t.Fatalf("expected zero sent/recv, got %+v", s)
	}
	if s.Loss != 0 {
		t.Fatalf("Loss = %v, want 0 when nothing was sent (avoids divide-by-zero)", s.Loss)
	}
}

func TestSummarizeStdDev(t *testing.T) {
	// RTTs of 10ms, 20ms, 30ms: mean 20ms, population stddev ~8.16ms.
	results := []Result{
		{Seq: 1, RTT: 10 * time.Millisecond},
		{Seq: 2, RTT: 20 * time.Millisecond},
		{Seq: 3, RTT: 30 * time.Millisecond},
	}

	s := Summarize(results, 3)

	want := 8163265 * time.Nanosecond // ~8.163265ms
	diff := s.StdDev - want
	if diff < 0 {
		diff = -diff
	}
	if diff > 2*time.Microsecond {
		t.Fatalf("StdDev = %v, want approximately %v", s.StdDev, want)
	}
}
