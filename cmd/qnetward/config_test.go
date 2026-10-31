package main

import (
	"testing"
)

func TestLookupEnvFallsBackToLegacyName(t *testing.T) {
	t.Setenv("QNETWARD_LOOKUP_TEST", "")
	t.Setenv("NETWATCHER_LOOKUP_TEST", "legacy")
	if got := lookupEnv("QNETWARD_LOOKUP_TEST"); got != "legacy" {
		t.Fatalf("expected legacy fallback, got %q", got)
	}

	t.Setenv("QNETWARD_LOOKUP_TEST", "current")
	if got := lookupEnv("QNETWARD_LOOKUP_TEST"); got != "current" {
		t.Fatalf("expected current value to win, got %q", got)
	}
}
