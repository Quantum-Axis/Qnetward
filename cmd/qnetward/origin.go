package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func isLocalRequestHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if strings.Contains(host, ":") {
		if parsedHost, _, err := net.SplitHostPort(host); err == nil {
			host = parsedHost
		}
	}
	host = strings.Trim(host, "[]")
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func isAllowedRequestOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	return isAllowedWSOrigin(r, origin)
}

func isAllowedWSOrigin(r *http.Request, origin string) bool {
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}

	originHost, originPort := splitHostAndOptionalPort(u.Host)
	requestHost, requestPort := splitHostAndOptionalPort(r.Host)
	if originPort == "" {
		originPort = defaultPortForScheme(u.Scheme)
	}
	if requestPort == "" {
		requestPort = defaultPortForRequest(r)
	}

	if hostsEquivalentForOrigin(originHost, requestHost) && originPort == requestPort {
		return true
	}

	if len(appAllowedOrigins) == 0 {
		return false
	}

	normalized, err := normalizeOrigin(origin)
	if err != nil {
		return false
	}
	_, ok := appAllowedOrigins[normalized]
	return ok
}

func hostsEquivalentForOrigin(left, right string) bool {
	if strings.EqualFold(left, right) {
		return true
	}
	return isLoopbackOriginHost(left) && isLoopbackOriginHost(right)
}

func isLoopbackOriginHost(raw string) bool {
	host := strings.TrimSpace(strings.Trim(raw, "[]"))
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func normalizeOrigin(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("scheme and host are required")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("query and fragment are not allowed")
	}
	if path := strings.TrimSpace(u.Path); path != "" && path != "/" {
		return "", fmt.Errorf("path is not allowed")
	}

	host, port := splitHostAndOptionalPort(u.Host)
	if host == "" {
		return "", fmt.Errorf("host is required")
	}
	if port == "" {
		port = defaultPortForScheme(u.Scheme)
	}
	if port == "" {
		return "", fmt.Errorf("port is required")
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(net.JoinHostPort(host, port)), nil
}

func splitHostAndOptionalPort(raw string) (string, string) {
	if raw == "" {
		return "", ""
	}
	if host, port, err := net.SplitHostPort(raw); err == nil {
		return host, port
	}
	return raw, ""
}

func defaultPortForScheme(scheme string) string {
	switch strings.ToLower(scheme) {
	case "https", "wss":
		return "443"
	case "http", "ws":
		return "80"
	default:
		return ""
	}
}

func defaultPortForRequest(r *http.Request) string {
	if r.TLS != nil {
		return "443"
	}
	return "80"
}
