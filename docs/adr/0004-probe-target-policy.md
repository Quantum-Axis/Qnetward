# 4. Block probes to private and metadata addresses by default

## Status

Accepted

## Context

Users choose the hosts that Qnetward pings, connects to, and scans. On a
hosted service, that lets a user point probes at the server's own network:
loopback services, other machines on the private network, or the cloud
instance-metadata endpoint at `169.254.169.254`, which can hand out cloud
credentials. This is server-side request forgery (SSRF).

Self-hosted installs have the opposite need: monitoring their own LAN is the
point.

## Decision

`internal/platform/netguard` defines a target policy. By default it rejects
loopback, RFC 1918, carrier-grade NAT, link-local, multicast, reserved and
documentation ranges, and IPv6 equivalents. Cloud metadata addresses are
always rejected.

The engines take an optional `AddrCheck` hook:

- TCP ping checks every resolved address before starting and again at connect
  time through `net.Dialer.Control`, so a hostname that re-resolves to a
  private address (DNS rebinding) is still refused.
- Port scan resolves once, checks the addresses, and dials those exact
  addresses.
- ICMP ping checks the resolved address before sending.

The policy applies to HTTP and WebSocket probes. Custom DNS resolvers are
checked too. The CLI is not restricted because it runs on the operator's own
machine.

`QNETWARD_ALLOW_PRIVATE_TARGETS=true` allows private targets for self-hosted
LAN monitoring. Metadata endpoints stay blocked.

## Consequences

- Hosted users cannot probe `localhost` or private IPs from the console.
- Local development that pings private addresses needs the flag set in `.env`.
- New probe types must wire `AddrCheck` (or the dial control) before they are
  exposed through the API.
- A public "try it" probe on the landing page becomes safe to build, together
  with stricter rate limits.
