package ping

import (
	"context"
	"errors"
	"testing"
	"time"

	"netwatcher/internal/report"
)

func TestRunStreamContextRejectsNonPositiveCount(t *testing.T) {
	_, err := RunStreamContext(context.Background(), "127.0.0.1", Config{Count: 0}, nil)
	if err == nil {
		t.Fatal("expected an error for Count <= 0, got nil")
	}
}

func TestRunLoopback(t *testing.T) {
	rep, err := Run("127.0.0.1", Config{Count: 2, Timeout: time.Second})
	if err != nil {
		if isPermissionErr(err) {
			t.Skipf("skipping: ICMP requires root or CAP_NET_RAW in this environment: %v", err)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	if rep.Protocol != "icmp" {
		t.Fatalf("Protocol = %q, want %q", rep.Protocol, "icmp")
	}
	if rep.Summary.Sent != 2 {
		t.Fatalf("Summary.Sent = %d, want 2", rep.Summary.Sent)
	}
	if rep.Summary.Recv != 2 {
		t.Fatalf("Summary.Recv = %d, want 2 (loopback ping should succeed): %+v", rep.Summary.Recv, rep)
	}
	if len(rep.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(rep.Results))
	}
}

func TestRunStreamCallsOnResult(t *testing.T) {
	var seen []int
	_, err := RunStream("127.0.0.1", Config{Count: 3, Timeout: time.Second}, func(r report.Result) error {
		seen = append(seen, r.Seq)
		return nil
	})
	if err != nil {
		if isPermissionErr(err) {
			t.Skipf("skipping: ICMP requires root or CAP_NET_RAW in this environment: %v", err)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	if len(seen) != 3 {
		t.Fatalf("onResult was called %d times, want 3 (seen=%v)", len(seen), seen)
	}
	for i, seq := range seen {
		if seq != i+1 {
			t.Fatalf("seen[%d] = %d, want %d", i, seq, i+1)
		}
	}
}

func TestRunStreamOnResultErrorStopsEarly(t *testing.T) {
	stop := errors.New("stop after first result")
	calls := 0
	_, err := RunStream("127.0.0.1", Config{Count: 5, Timeout: time.Second}, func(r report.Result) error {
		calls++
		return stop
	})
	if err == nil {
		if calls == 0 {
			t.Skip("skipping: no ICMP replies were received to trigger the callback in this environment")
		}
		t.Fatal("expected the onResult error to propagate, got nil")
	}
	if isPermissionErr(err) {
		t.Skipf("skipping: ICMP requires root or CAP_NET_RAW in this environment: %v", err)
	}
	if !errors.Is(err, stop) {
		t.Fatalf("error = %v, want it to wrap/equal %v", err, stop)
	}
	if calls != 1 {
		t.Fatalf("onResult was called %d times, want exactly 1 (should stop after the callback errors)", calls)
	}
}

func TestResolveAddrIPv4Literal(t *testing.T) {
	addr, isIPv6, err := resolveAddr("127.0.0.1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isIPv6 {
		t.Fatal("expected isIPv6=false for an IPv4 literal")
	}
	if addr.String() != "127.0.0.1" {
		t.Fatalf("addr = %q, want %q", addr.String(), "127.0.0.1")
	}
}

func TestResolveAddrIPv6Literal(t *testing.T) {
	addr, isIPv6, err := resolveAddr("::1", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isIPv6 {
		t.Fatal("expected isIPv6=true when ForceIPv6 is set")
	}
	if addr.String() != "::1" {
		t.Fatalf("addr = %q, want %q", addr.String(), "::1")
	}
}

func TestResolveAddrPrefersIPv4WhenNotForced(t *testing.T) {
	addr, isIPv6, err := resolveAddr("127.0.0.1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isIPv6 {
		t.Fatal("expected an IPv4 literal to resolve as IPv4 even though ForceIPv6 was not requested")
	}
	if addr == nil {
		t.Fatal("expected a non-nil address")
	}
}

func TestIsPermissionErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"permission denied", errors.New("listen ip4:icmp: socket: permission denied"), true},
		{"operation not permitted", errors.New("operation not permitted"), true},
		{"unrelated error", errors.New("no such host"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isPermissionErr(tc.err); got != tc.want {
				t.Fatalf("isPermissionErr(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestSleepContextZeroDuration(t *testing.T) {
	if err := sleepContext(context.Background(), 0); err != nil {
		t.Fatalf("expected nil error for a zero duration, got %v", err)
	}
}

func TestSleepContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := sleepContext(ctx, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("sleepContext with a cancelled context = %v, want context.Canceled", err)
	}
}

func TestSleepContextCompletes(t *testing.T) {
	if err := sleepContext(context.Background(), 5*time.Millisecond); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
