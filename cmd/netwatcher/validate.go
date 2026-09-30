package main

import (
	"fmt"
	"strings"
	"time"

	"netwatcher/internal/ping"
	"netwatcher/internal/portscan"
	"netwatcher/internal/tcpping"
)

func normalizePingConfig(count int, interval, timeout time.Duration, size int, forceIPv6 bool) (ping.Config, error) {
	normalizedCount, err := normalizeProbeCount(count)
	if err != nil {
		return ping.Config{}, err
	}
	normalizedInterval, err := normalizeProbeInterval(interval)
	if err != nil {
		return ping.Config{}, err
	}
	normalizedTimeout, err := normalizeProbeTimeout(timeout)
	if err != nil {
		return ping.Config{}, err
	}
	normalizedSize, err := normalizePingSize(size)
	if err != nil {
		return ping.Config{}, err
	}

	return ping.Config{
		Count:     normalizedCount,
		Interval:  normalizedInterval,
		Timeout:   normalizedTimeout,
		Size:      normalizedSize,
		ForceIPv6: forceIPv6,
	}, nil
}

func normalizeTCPPingRequest(host string, port, count int, interval, timeout time.Duration) (string, int, tcpping.Config, error) {
	var err error
	host, err = normalizeHost(host)
	if err != nil {
		return "", 0, tcpping.Config{}, err
	}
	if port == 0 {
		port = defaultTCPPort
	}
	if port < 1 || port > 65535 {
		return "", 0, tcpping.Config{}, fmt.Errorf("port must be 1-65535")
	}

	normalizedCount, err := normalizeProbeCount(count)
	if err != nil {
		return "", 0, tcpping.Config{}, err
	}
	normalizedInterval, err := normalizeProbeInterval(interval)
	if err != nil {
		return "", 0, tcpping.Config{}, err
	}
	normalizedTimeout, err := normalizeProbeTimeout(timeout)
	if err != nil {
		return "", 0, tcpping.Config{}, err
	}

	return host, port, tcpping.Config{
		Count:    normalizedCount,
		Interval: normalizedInterval,
		Timeout:  normalizedTimeout,
	}, nil
}

func normalizePortScanRequest(host, portsSpec string, timeout time.Duration, concurrency int) (string, []int, portscan.Config, error) {
	var err error
	host, err = normalizeHost(host)
	if err != nil {
		return "", nil, portscan.Config{}, err
	}

	ports, err := portscan.ParsePorts(portsSpec)
	if err != nil {
		return "", nil, portscan.Config{}, err
	}
	if len(ports) > maxScanPorts {
		return "", nil, portscan.Config{}, fmt.Errorf("scan is limited to %d ports per request", maxScanPorts)
	}

	normalizedTimeout, err := normalizeProbeTimeout(timeout)
	if err != nil {
		return "", nil, portscan.Config{}, err
	}

	if concurrency == 0 {
		concurrency = defaultScanWorkers
	}
	if concurrency < minScanConcurrency || concurrency > maxScanConcurrency {
		return "", nil, portscan.Config{}, fmt.Errorf("concurrency must be %d-%d", minScanConcurrency, maxScanConcurrency)
	}
	if concurrency > len(ports) {
		concurrency = len(ports)
	}

	return host, ports, portscan.Config{
		Ports:       ports,
		Timeout:     normalizedTimeout,
		Concurrency: concurrency,
	}, nil
}

func normalizeDNSLookupTimeout(timeout time.Duration) (time.Duration, error) {
	normalizedTimeout, err := normalizeProbeTimeout(timeout)
	if err != nil {
		return 0, err
	}
	return normalizedTimeout, nil
}

func normalizeHost(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("host is required")
	}
	if len(host) > maxHostLength {
		return "", fmt.Errorf("host is too long")
	}
	for _, b := range []byte(host) {
		if b < 32 || b == 127 {
			return "", fmt.Errorf("host contains invalid characters")
		}
	}
	return host, nil
}

func normalizeProbeCount(count int) (int, error) {
	if count == 0 {
		count = defaultProbeCount
	}
	if count < 1 || count > maxProbeCount {
		return 0, fmt.Errorf("count must be 1-%d", maxProbeCount)
	}
	return count, nil
}

func normalizeProbeInterval(interval time.Duration) (time.Duration, error) {
	if interval <= 0 {
		interval = defaultProbeInterval
	}
	if interval < minProbeInterval || interval > maxProbeInterval {
		return 0, fmt.Errorf("interval must be %s-%s", minProbeInterval, maxProbeInterval)
	}
	return interval, nil
}

func normalizeProbeTimeout(timeout time.Duration) (time.Duration, error) {
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}
	if timeout < minProbeTimeout || timeout > maxProbeTimeout {
		return 0, fmt.Errorf("timeout must be %s-%s", minProbeTimeout, maxProbeTimeout)
	}
	return timeout, nil
}

func normalizePingSize(size int) (int, error) {
	if size == 0 {
		size = defaultPingSize
	}
	if size < minPingSize || size > maxPingSize {
		return 0, fmt.Errorf("size must be %d-%d bytes", minPingSize, maxPingSize)
	}
	return size, nil
}

func parsePort(portStr string) (int, error) {
	var port int
	_, err := fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return 0, err
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port out of range")
	}
	return port, nil
}
