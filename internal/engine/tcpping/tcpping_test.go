package tcpping

import (
	"context"
	"errors"
	"net"
	"strconv"
	"testing"
	"time"

	"qnetward/internal/engine/report"
)

func TestRunContextRejectsNonPositiveCount(t *testing.T) {
	_, err := RunContext(context.Background(), "127.0.0.1", 80, Config{Count: 0})
	if err == nil {
		t.Fatal("expected an error for Count <= 0, got nil")
	}
}

func TestRunContextRejectsInvalidPort(t *testing.T) {
	cases := []int{0, -1, 65536, 100000}
	for _, port := range cases {
		if _, err := RunContext(context.Background(), "127.0.0.1", port, Config{Count: 1}); err == nil {
			t.Errorf("port %d: expected an error, got nil", port)
		}
	}
}

func TestRunAgainstOpenPortSucceeds(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to split listener address: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse listener port: %v", err)
	}

	rep, err := Run(host, port, Config{Count: 3, Interval: time.Millisecond, Timeout: time.Second})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rep.Protocol != "tcp" {
		t.Fatalf("Protocol = %q, want %q", rep.Protocol, "tcp")
	}
	if rep.Port != port {
		t.Fatalf("Port = %d, want %d", rep.Port, port)
	}
	if rep.Summary.Sent != 3 || rep.Summary.Recv != 3 {
		t.Fatalf("Summary = %+v, want Sent=3 Recv=3", rep.Summary)
	}
	if rep.Summary.Loss != 0 {
		t.Fatalf("Loss = %v, want 0", rep.Summary.Loss)
	}
	for _, r := range rep.Results {
		if r.Error != "" {
			t.Errorf("result %d: unexpected error %q", r.Seq, r.Error)
		}
	}
}

func TestRunAgainstClosedPortRecordsError(t *testing.T) {
	// Bind a listener just to reserve a free port, then close it immediately
	// so the port is very likely to be refused rather than filtered/dropped.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve a port: %v", err)
	}
	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to split listener address: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse listener port: %v", err)
	}
	ln.Close()

	rep, err := Run(host, port, Config{Count: 1, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	if rep.Summary.Recv != 0 {
		t.Fatalf("Summary.Recv = %d, want 0 against a closed port", rep.Summary.Recv)
	}
	if rep.Summary.Loss != 100 {
		t.Fatalf("Summary.Loss = %v, want 100", rep.Summary.Loss)
	}
	if len(rep.Results) != 1 || rep.Results[0].Error == "" {
		t.Fatalf("expected a recorded connection error, got %+v", rep.Results)
	}
}

func TestRunContextStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rep, err := RunContext(ctx, "127.0.0.1", 80, Config{Count: 5})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if len(rep.Results) != 0 {
		t.Fatalf("expected no results once the context was already cancelled, got %d", len(rep.Results))
	}
}

func TestRunStreamCallsOnResult(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test listener: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("failed to split listener address: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse listener port: %v", err)
	}

	var seen []int
	_, err = RunStream(host, port, Config{Count: 3, Interval: time.Millisecond, Timeout: time.Second}, func(r report.Result) error {
		seen = append(seen, r.Seq)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(seen) != 3 {
		t.Fatalf("onResult was called %d times, want 3 (seen=%v)", len(seen), seen)
	}
}
