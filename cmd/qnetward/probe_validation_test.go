package main

import (
	"testing"
	"time"
)

func TestNormalizePingConfigDefaultsAndBounds(t *testing.T) {
	cfg, err := normalizePingConfig(0, 0, 0, 0, true, false)
	if err != nil {
		t.Fatalf("normalizePingConfig returned error: %v", err)
	}

	if cfg.Count != defaultProbeCount {
		t.Fatalf("expected default count %d, got %d", defaultProbeCount, cfg.Count)
	}
	if cfg.Interval != defaultProbeInterval {
		t.Fatalf("expected default interval %s, got %s", defaultProbeInterval, cfg.Interval)
	}
	if cfg.Timeout != defaultProbeTimeout {
		t.Fatalf("expected default timeout %s, got %s", defaultProbeTimeout, cfg.Timeout)
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

	continuousCfg, err := normalizePingConfig(5, time.Second, 2*time.Second, defaultPingSize, false, true)
	if err != nil {
		t.Fatalf("normalizePingConfig with continuous returned error: %v", err)
	}
	if !continuousCfg.Continuous {
		t.Fatal("expected continuous mode to be preserved")
	}
	continuousOverflowCfg, err := normalizePingConfig(maxProbeCount+1, time.Second, 2*time.Second, defaultPingSize, false, true)
	if err != nil {
		t.Fatalf("normalizePingConfig should ignore oversized count in continuous mode: %v", err)
	}
	if continuousOverflowCfg.Count != defaultProbeCount {
		t.Fatalf("expected continuous oversized count to normalize to %d, got %d", defaultProbeCount, continuousOverflowCfg.Count)
	}

	if _, err := normalizePingConfig(maxProbeCount+1, time.Second, time.Second, defaultPingSize, false, false); err == nil {
		t.Fatal("expected count validation error")
	}
	if _, err := normalizePingConfig(defaultProbeCount, minProbeInterval-time.Millisecond, time.Second, defaultPingSize, false, false); err == nil {
		t.Fatal("expected interval validation error")
	}
	if _, err := normalizePingConfig(defaultProbeCount, time.Second, minProbeTimeout-time.Millisecond, defaultPingSize, false, false); err == nil {
		t.Fatal("expected timeout validation error")
	}
	if _, err := normalizePingConfig(defaultProbeCount, time.Second, time.Second, minPingSize-1, false, false); err == nil {
		t.Fatal("expected size validation error")
	}
}

func TestNormalizeTCPPingRequestDefaultsAndBounds(t *testing.T) {
	host, port, cfg, err := normalizeTCPPingRequest("  example.com  ", 0, 0, 0, 0, false)
	if err != nil {
		t.Fatalf("normalizeTCPPingRequest returned error: %v", err)
	}

	if host != "example.com" {
		t.Fatalf("expected trimmed host, got %q", host)
	}
	if port != defaultTCPPort {
		t.Fatalf("expected default port %d, got %d", defaultTCPPort, port)
	}
	if cfg.Count != defaultProbeCount {
		t.Fatalf("expected default count %d, got %d", defaultProbeCount, cfg.Count)
	}

	_, _, continuousCfg, err := normalizeTCPPingRequest("example.com", 443, 5, time.Second, 2*time.Second, true)
	if err != nil {
		t.Fatalf("normalizeTCPPingRequest with continuous returned error: %v", err)
	}
	if !continuousCfg.Continuous {
		t.Fatal("expected continuous mode to be preserved")
	}
	_, _, continuousOverflowCfg, err := normalizeTCPPingRequest("example.com", 443, maxProbeCount+1, time.Second, 2*time.Second, true)
	if err != nil {
		t.Fatalf("normalizeTCPPingRequest should ignore oversized count in continuous mode: %v", err)
	}
	if continuousOverflowCfg.Count != defaultProbeCount {
		t.Fatalf("expected continuous oversized count to normalize to %d, got %d", defaultProbeCount, continuousOverflowCfg.Count)
	}

	if _, _, _, err := normalizeTCPPingRequest("", 443, 1, time.Second, time.Second, false); err == nil {
		t.Fatal("expected host validation error")
	}
	if _, _, _, err := normalizeTCPPingRequest("example.com", 70000, 1, time.Second, time.Second, false); err == nil {
		t.Fatal("expected port validation error")
	}
}

func TestNormalizePortScanRequestDefaultsAndBounds(t *testing.T) {
	host, ports, cfg, err := normalizePortScanRequest("  example.com  ", "22,80,443", 0, 0)
	if err != nil {
		t.Fatalf("normalizePortScanRequest returned error: %v", err)
	}

	if host != "example.com" {
		t.Fatalf("expected trimmed host, got %q", host)
	}
	if len(ports) != 3 {
		t.Fatalf("expected 3 ports, got %d", len(ports))
	}
	if cfg.Timeout != defaultProbeTimeout {
		t.Fatalf("expected default timeout %s, got %s", defaultProbeTimeout, cfg.Timeout)
	}
	if cfg.Concurrency != 3 {
		t.Fatalf("expected concurrency capped to port count 3, got %d", cfg.Concurrency)
	}

	if _, _, _, err := normalizePortScanRequest("example.com", "", time.Second, 10); err == nil {
		t.Fatal("expected ports validation error")
	}
	if _, _, _, err := normalizePortScanRequest("example.com", "1-5000", time.Second, 10); err == nil {
		t.Fatal("expected max port count validation error")
	}
	if _, _, _, err := normalizePortScanRequest("example.com", "80,443", time.Second, maxScanConcurrency+1); err == nil {
		t.Fatal("expected concurrency validation error")
	}
}

func TestNormalizeDNSLookupTimeoutBounds(t *testing.T) {
	got, err := normalizeDNSLookupTimeout(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got <= 0 {
		t.Fatalf("expected positive timeout, got %s", got)
	}

	if _, err := normalizeDNSLookupTimeout(50 * time.Millisecond); err == nil {
		t.Fatal("expected error for too-small timeout")
	}
}
