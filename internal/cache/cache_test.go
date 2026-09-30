package cache

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestOpenWithEmptyAddrIsANoop(t *testing.T) {
	c, err := Open(Config{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c != nil {
		t.Fatalf("expected a nil *Cache when no address is configured, got %+v", c)
	}
}

func TestOpenWithUnreachableAddrFails(t *testing.T) {
	// Reserve a TCP port and close it immediately so it is refused quickly
	// rather than filtered/dropped, keeping the test fast.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve a port: %v", err)
	}
	addr := ln.Addr().String()
	ln.Close()

	c, err := Open(Config{Addr: addr})
	if err == nil {
		t.Fatal("expected an error connecting to a closed port, got nil")
	}
	if c != nil {
		t.Fatalf("expected a nil *Cache on error, got %+v", c)
	}
}

// A nil *Cache is what callers get back whenever Redis isn't configured
// (Open returns nil, nil above); every method must stay a safe no-op so the
// rest of the application doesn't need to nil-check the cache everywhere.
func TestNilCacheMethodsAreSafeNoops(t *testing.T) {
	var c *Cache

	if err := c.Close(); err != nil {
		t.Errorf("Close() on nil cache = %v, want nil", err)
	}
	if err := c.Ping(context.Background()); err != nil {
		t.Errorf("Ping() on nil cache = %v, want nil", err)
	}

	found, err := c.GetJSON(context.Background(), "key", &struct{}{})
	if err != nil || found {
		t.Errorf("GetJSON() on nil cache = (%v, %v), want (false, nil)", found, err)
	}

	if err := c.SetJSON(context.Background(), "key", map[string]string{"a": "b"}); err != nil {
		t.Errorf("SetJSON() on nil cache = %v, want nil", err)
	}
}

func TestKeyPrefixing(t *testing.T) {
	c := &Cache{cfg: Config{Prefix: "netwatcher", TTL: time.Second}}
	if got := c.key("history:tcpping:example.com"); got != "netwatcher:history:tcpping:example.com" {
		t.Fatalf("key() = %q, want %q", got, "netwatcher:history:tcpping:example.com")
	}
}
