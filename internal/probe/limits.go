package probe

import "time"

// Defaults and bounds shared by the CLI and the API for probe requests.
const (
	DefaultCount       = 5
	DefaultInterval    = time.Second
	DefaultTimeout     = 2 * time.Second
	DefaultTCPPort     = 80
	MaxCount           = 100
	DefaultScanWorkers = 64
	defaultPingSize    = 32
	minProbeInterval   = 100 * time.Millisecond
	maxProbeInterval   = 60 * time.Second
	minProbeTimeout    = 100 * time.Millisecond
	maxProbeTimeout    = 60 * time.Second
	minPingSize        = 8
	maxPingSize        = 65535
	minScanConcurrency = 1
	maxScanConcurrency = 256
	maxScanPorts       = 4096
	maxHostLength      = 255
)
