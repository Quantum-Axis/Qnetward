package probe

import (
	"testing"
	"time"
)

func TestNormalizePingConfigDefaultsAndBounds(t *testing.T) {
	cfg, err := NormalizePingConfig(0, 0, 0, 0, true, false)
	if err != nil {
		t.Fatalf("NormalizePingConfig returned error: %v", err)
	}

	if cfg.Count != DefaultCount {
		t.Fatalf("expected default count %d, got %d", DefaultCount, cfg.Count)
	}
	if cfg.Interval != DefaultInterval {
		t.Fatalf("expected default interval %s, got %s", DefaultInterval, cfg.Interval)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Fatalf("expected default timeout %s, got %s", DefaultTimeout, cfg.Timeout)
	}
	if cfg.Size != defaultPingSize {
		t.Fatalf("expected default size %d, got %d", defaultPingSize, cfg.Size)
	}
	if !cfg.ForceIPv6 {
		t.Fatal("expected ForceIPv6 to be preserved")
	}
	if cfg.Continuous {
		t.Fatal("expected continuous mode to remain disabled")
	}

	continuousCfg, err := NormalizePingConfig(5, time.Second, 2*time.Second, defaultPingSize, false, true)
	if err != nil {
		t.Fatalf("NormalizePingConfig with continuous returned error: %v", err)
	}
	if !continuousCfg.Continuous {
		t.Fatal("expected continuous mode to be preserved")
	}
	continuousOverflowCfg, err := NormalizePingConfig(MaxCount+1, time.Second, 2*time.Second, defaultPingSize, false, true)
	if err != nil {
		t.Fatalf("NormalizePingConfig should ignore oversized count in continuous mode: %v", err)
	}
	if continuousOverflowCfg.Count != DefaultCount {
		t.Fatalf("expected continuous oversized count to normalize to %d, got %d", DefaultCount, continuousOverflowCfg.Count)
	}

	if _, err := NormalizePingConfig(MaxCount+1, time.Second, time.Second, defaultPingSize, false, false); err == nil {
		t.Fatal("expected count validation error")
	}
	if _, err := NormalizePingConfig(DefaultCount, minProbeInterval-time.Millisecond, time.Second, defaultPingSize, false, false); err == nil {
		t.Fatal("expected interval validation error")
	}
	if _, err := NormalizePingConfig(DefaultCount, time.Second, minProbeTimeout-time.Millisecond, defaultPingSize, false, false); err == nil {
		t.Fatal("expected timeout validation error")
	}
	if _, err := NormalizePingConfig(DefaultCount, time.Second, time.Second, minPingSize-1, false, false); err == nil {
		t.Fatal("expected size validation error")
	}
}

func TestNormalizeTCPPingRequestDefaultsAndBounds(t *testing.T) {
	host, port, cfg, err := NormalizeTCPPingRequest("  example.com  ", 0, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("NormalizeTCPPingRequest returned error: %v", err)
	}

	if host != "example.com" {
		t.Fatalf("expected trimmed host, got %q", host)
	}
	if port != DefaultTCPPort {
		t.Fatalf("expected default port %d, got %d", DefaultTCPPort, port)
	}
	if cfg.Count != DefaultCount {
		t.Fatalf("expected default count %d, got %d", DefaultCount, cfg.Count)
	}

	_, _, continuousCfg, err := NormalizeTCPPingRequest("example.com", 443, 5, time.Second, 2*time.Second, true)
	if err != nil {
		t.Fatalf("NormalizeTCPPingRequest with continuous returned error: %v", err)
	}
	if !continuousCfg.Continuous {
		t.Fatal("expected continuous mode to be preserved")
	}
	_, _, continuousOverflowCfg, err := NormalizeTCPPingRequest("example.com", 443, MaxCount+1, time.Second, 2*time.Second, true)
	if err != nil {
		t.Fatalf("NormalizeTCPPingRequest should ignore oversized count in continuous mode: %v", err)
	}
	if continuousOverflowCfg.Count != DefaultCount {
		t.Fatalf("expected continuous oversized count to normalize to %d, got %d", DefaultCount, continuousOverflowCfg.Count)
	}

	if _, _, _, err := NormalizeTCPPingRequest("", 443, 1, time.Second, time.Second, false); err == nil {
		t.Fatal("expected host validation error")
	}
	if _, _, _, err := NormalizeTCPPingRequest("example.com", 70000, 1, time.Second, time.Second, false); err == nil {
		t.Fatal("expected port validation error")
	}
}

func TestNormalizePortScanRequestDefaultsAndBounds(t *testing.T) {
	host, ports, cfg, err := NormalizePortScanRequest("  example.com  ", "22,80,443", 0, 0)
	if err != nil {
		t.Fatalf("NormalizePortScanRequest returned error: %v", err)
	}

	if host != "example.com" {
		t.Fatalf("expected trimmed host, got %q", host)
	}
	if len(ports) != 3 {
		t.Fatalf("expected 3 ports, got %d", len(ports))
	}
	if cfg.Timeout != DefaultTimeout {
		t.Fatalf("expected default timeout %s, got %s", DefaultTimeout, cfg.Timeout)
	}
	if cfg.Concurrency != 3 {
		t.Fatalf("expected concurrency capped to port count 3, got %d", cfg.Concurrency)
	}

	if _, _, _, err := NormalizePortScanRequest("example.com", "", time.Second, 10); err == nil {
		t.Fatal("expected ports validation error")
	}
	if _, _, _, err := NormalizePortScanRequest("example.com", "1-5000", time.Second, 10); err == nil {
		t.Fatal("expected max port count validation error")
	}
	if _, _, _, err := NormalizePortScanRequest("example.com", "80,443", time.Second, maxScanConcurrency+1); err == nil {
		t.Fatal("expected concurrency validation error")
	}
}

func TestNormalizeDNSLookupTimeoutBounds(t *testing.T) {
	got, err := NormalizeDNSLookupTimeout(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got <= 0 {
		t.Fatalf("expected positive timeout, got %s", got)
	}

	if _, err := NormalizeDNSLookupTimeout(50 * time.Millisecond); err == nil {
		t.Fatal("expected error for too-small timeout")
	}
}
