// Package probe holds what the CLI and the API share about probe requests:
// input validation and normalization, default limits, and the JSON shape of
// probe reports. It does no network I/O itself; that is internal/engine.
package probe
