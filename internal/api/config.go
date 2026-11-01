package api

import (
	"fmt"
	"qnetward/internal/store"
	"strings"
)

// ParseAllowedOrigins parses a comma-separated list of extra browser origins.
func ParseAllowedOrigins(raw string) (map[string]struct{}, error) {
	origins := make(map[string]struct{})
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		normalized, err := normalizeOrigin(item)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed origin %q: %w", item, err)
		}
		origins[normalized] = struct{}{}
	}
	return origins, nil
}

// ParseAdminEmails parses a comma-separated list of platform admin emails.
func ParseAdminEmails(raw string) (map[string]struct{}, error) {
	adminEmails := make(map[string]struct{})
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		email, err := store.NormalizeEmail(item)
		if err != nil {
			return nil, fmt.Errorf("invalid admin email %q: %w", item, err)
		}
		adminEmails[email] = struct{}{}
	}
	return adminEmails, nil
}
