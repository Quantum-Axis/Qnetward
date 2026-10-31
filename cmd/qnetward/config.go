package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"qnetward/internal/store"
)

// lookupEnv reads key and, for QNETWARD_* keys, falls back to the legacy
// NETWATCHER_* name so deployments configured before the rename keep working.
func lookupEnv(key string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	if suffix, ok := strings.CutPrefix(key, "QNETWARD_"); ok {
		return strings.TrimSpace(os.Getenv("NETWATCHER_" + suffix))
	}
	return ""
}

func envString(key, fallback string) string {
	if value := lookupEnv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := lookupEnv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := lookupEnv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := lookupEnv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseAllowedOrigins(raw string) (map[string]struct{}, error) {
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

func parseAdminEmails(raw string) (map[string]struct{}, error) {
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
