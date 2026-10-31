package netguard

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
)

func TestCheckAddrBlocksNonPublicByDefault(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "10.1.2.3", "172.16.0.9", "192.168.1.1", "100.64.0.1",
		"169.254.10.10", "0.0.0.0", "224.0.0.1", "255.255.255.255",
		"::1", "fe80::1", "fd12:3456::1", "::ffff:10.0.0.1", "::ffff:127.0.0.1",
	}
	var p Policy
	for _, raw := range blocked {
		if err := p.CheckAddr(netip.MustParseAddr(raw)); !errors.Is(err, ErrBlocked) {
			t.Errorf("CheckAddr(%s) = %v, want ErrBlocked", raw, err)
		}
	}
}

func TestCheckAddrAllowsPublic(t *testing.T) {
	var p Policy
	for _, raw := range []string{"1.1.1.1", "8.8.8.8", "93.184.216.34", "2606:4700:4700::1111"} {
		if err := p.CheckAddr(netip.MustParseAddr(raw)); err != nil {
			t.Errorf("CheckAddr(%s) = %v, want nil", raw, err)
		}
	}
}

func TestAllowPrivateStillBlocksMetadata(t *testing.T) {
	p := Policy{AllowPrivate: true}
	if err := p.CheckAddr(netip.MustParseAddr("192.168.1.1")); err != nil {
		t.Fatalf("private address should be allowed: %v", err)
	}
	if err := p.CheckAddr(netip.MustParseAddr("127.0.0.1")); err != nil {
		t.Fatalf("loopback should be allowed: %v", err)
	}
	for _, raw := range []string{"169.254.169.254", "::ffff:169.254.169.254", "fd00:ec2::254", "100.100.100.200"} {
		if err := p.CheckAddr(netip.MustParseAddr(raw)); !errors.Is(err, ErrBlocked) {
			t.Errorf("metadata %s should stay blocked, got %v", raw, err)
		}
	}
}

func TestCheckIP(t *testing.T) {
	var p Policy
	if err := p.CheckIP(net.ParseIP("10.0.0.1")); !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected ErrBlocked, got %v", err)
	}
	if err := p.CheckIP(net.ParseIP("1.1.1.1")); err != nil {
		t.Fatalf("expected public IP to pass, got %v", err)
	}
	if err := p.CheckIP(nil); !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected nil IP to be blocked, got %v", err)
	}
}

func TestCheckHostLiteralAndLocalhost(t *testing.T) {
	var p Policy
	if err := p.CheckHost(context.Background(), "127.0.0.1"); !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected literal loopback to be blocked, got %v", err)
	}
	if err := p.CheckHost(context.Background(), "localhost"); !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected localhost to be blocked, got %v", err)
	}
}

func TestDialControlRejectsBlockedAddress(t *testing.T) {
	control := Policy{}.DialControl()
	if err := control("tcp", "127.0.0.1:80", nil); !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected ErrBlocked, got %v", err)
	}
	if err := control("tcp", "[2606:4700:4700::1111]:443", nil); err != nil {
		t.Fatalf("expected public address to pass, got %v", err)
	}
}

func TestDialControlBlocksRealDial(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen: %v", err)
	}
	defer ln.Close()

	dialer := &net.Dialer{Control: Policy{}.DialControl()}
	conn, err := dialer.Dial("tcp", ln.Addr().String())
	if err == nil {
		conn.Close()
		t.Fatal("expected dial to loopback to be rejected")
	}
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected ErrBlocked, got %v", err)
	}

	allowed := &net.Dialer{Control: Policy{AllowPrivate: true}.DialControl()}
	conn, err = allowed.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("expected dial to succeed with AllowPrivate: %v", err)
	}
	conn.Close()
}
