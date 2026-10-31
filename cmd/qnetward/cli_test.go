package main

import (
	"testing"
)

func TestNormalizeArgsKeepsBoolFlagsFromConsumingHost(t *testing.T) {
	fs := newPingFlagSet()
	if err := fs.Parse(normalizeArgs(fs, []string{"--ipv6", "example.com"})); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if fs.NArg() != 1 || fs.Arg(0) != "example.com" {
		t.Fatalf("expected host to remain positional, got %q", fs.Arg(0))
	}
}

func TestNormalizeArgsMovesValueFlagsAheadOfPositionals(t *testing.T) {
	fs := newTCPPingFlagSet()
	if err := fs.Parse(normalizeArgs(fs, []string{"example.com", "--port", "443"})); err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	portValue := fs.Lookup("port").Value.String()
	if portValue != "443" {
		t.Fatalf("expected port flag to parse as 443, got %s", portValue)
	}
	if fs.NArg() != 1 || fs.Arg(0) != "example.com" {
		t.Fatalf("expected host positional arg, got %q", fs.Arg(0))
	}
}
