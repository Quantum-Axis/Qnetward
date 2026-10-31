// Package netguard decides which network addresses user-requested probes may
// reach. It exists to stop the hosted service from being steered at its own
// network: loopback, private ranges, link-local, and cloud metadata endpoints.
package netguard

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"syscall"
)

// ErrBlocked is returned (wrapped) when a target resolves to a disallowed address.
var ErrBlocked = errors.New("target address is not allowed")

// Cloud instance-metadata endpoints. These are blocked even when private
// targets are allowed, because reaching them can leak credentials.
var metadataAddrs = []netip.Addr{
	netip.MustParseAddr("169.254.169.254"), // AWS, GCP, Azure, DigitalOcean, OpenStack
	netip.MustParseAddr("169.254.170.2"),   // AWS ECS task metadata
	netip.MustParseAddr("100.100.100.200"), // Alibaba Cloud
	netip.MustParseAddr("fd00:ec2::254"),   // AWS IPv6
}

// Ranges that are not public internet destinations.
var nonPublicPrefixes = mustPrefixes(
	"0.0.0.0/8",       // "this network"
	"10.0.0.0/8",      // RFC 1918
	"100.64.0.0/10",   // carrier-grade NAT
	"127.0.0.0/8",     // loopback
	"169.254.0.0/16",  // link-local
	"172.16.0.0/12",   // RFC 1918
	"192.0.0.0/24",    // IETF protocol assignments
	"192.0.2.0/24",    // TEST-NET-1
	"192.168.0.0/16",  // RFC 1918
	"198.18.0.0/15",   // benchmarking
	"198.51.100.0/24", // TEST-NET-2
	"203.0.113.0/24",  // TEST-NET-3
	"224.0.0.0/4",     // multicast
	"240.0.0.0/4",     // reserved, includes broadcast
	"::/128",          // unspecified
	"::1/128",         // loopback
	"64:ff9b:1::/48",  // local-use NAT64
	"100::/64",        // discard-only
	"2001:db8::/32",   // documentation
	"fc00::/7",        // unique local
	"fe80::/10",       // link-local
	"ff00::/8",        // multicast
)

func mustPrefixes(values ...string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(values))
	for _, v := range values {
		out = append(out, netip.MustParsePrefix(v))
	}
	return out
}

// Policy controls which destinations are allowed.
type Policy struct {
	// AllowPrivate permits loopback, private, and link-local targets. Enable it
	// for self-hosted installs that monitor their own LAN; keep it off for the
	// hosted service. Metadata endpoints stay blocked either way.
	AllowPrivate bool
}

// CheckAddr reports whether addr may be probed under p.
func (p Policy) CheckAddr(addr netip.Addr) error {
	if !addr.IsValid() {
		return fmt.Errorf("%w: invalid address", ErrBlocked)
	}
	addr = addr.Unmap().WithZone("")
	for _, m := range metadataAddrs {
		if addr == m {
			return fmt.Errorf("%w: %s is a cloud metadata endpoint", ErrBlocked, addr)
		}
	}
	if p.AllowPrivate {
		return nil
	}
	for _, prefix := range nonPublicPrefixes {
		if prefix.Contains(addr) {
			return fmt.Errorf("%w: %s is not a public address", ErrBlocked, addr)
		}
	}
	return nil
}

// CheckIP is CheckAddr for a net.IP, the form the probe engines work with.
func (p Policy) CheckIP(ip net.IP) error {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return fmt.Errorf("%w: invalid address", ErrBlocked)
	}
	return p.CheckAddr(addr)
}

// CheckHost resolves host and fails if any resulting address is disallowed.
// Prefer DialControl where a dial happens, since it checks the address that
// is actually used and so cannot be bypassed by DNS rebinding.
func (p Policy) CheckHost(ctx context.Context, host string) error {
	if addr, err := netip.ParseAddr(host); err == nil {
		return p.CheckAddr(addr)
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return fmt.Errorf("no addresses found for %s", host)
	}
	for _, ip := range ips {
		if err := p.CheckAddr(ip); err != nil {
			return err
		}
	}
	return nil
}

// DialControl returns a net.Dialer Control hook that rejects connections to
// disallowed addresses after DNS resolution, immediately before connect.
func (p Policy) DialControl() func(network, address string, c syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			host = address
		}
		addr, err := netip.ParseAddr(host)
		if err != nil {
			return fmt.Errorf("%w: unresolved address %q", ErrBlocked, host)
		}
		return p.CheckAddr(addr)
	}
}
