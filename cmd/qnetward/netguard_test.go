package main

import (
	"context"
	"errors"
	"testing"

	"qnetward/internal/platform/netguard"
)

func withTargetPolicy(t *testing.T, p netguard.Policy) {
	t.Helper()
	old := appTargetPolicy
	appTargetPolicy = p
	t.Cleanup(func() { appTargetPolicy = old })
}

func TestCheckCustomResolverAllowsPresets(t *testing.T) {
	withTargetPolicy(t, netguard.Policy{})
	for _, r := range []string{"", "auto", "cloudflare", "Google", "quad9", "1.1.1.1", "8.8.8.8:53", "[2606:4700:4700::1111]:53"} {
		if err := checkCustomResolver(context.Background(), r); err != nil {
			t.Errorf("checkCustomResolver(%q) = %v, want nil", r, err)
		}
	}
}

func TestCheckCustomResolverBlocksPrivate(t *testing.T) {
	withTargetPolicy(t, netguard.Policy{})
	for _, r := range []string{"127.0.0.1", "10.0.0.2:53", "169.254.169.254", "[::1]:53", "localhost"} {
		if err := checkCustomResolver(context.Background(), r); !errors.Is(err, netguard.ErrBlocked) {
			t.Errorf("checkCustomResolver(%q) = %v, want ErrBlocked", r, err)
		}
	}
}
