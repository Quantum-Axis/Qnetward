package main

import (
	"testing"
	"time"
)

func TestRateLimiterSweepPrunesStaleBuckets(t *testing.T) {
	rl := newRateLimiter()
	old := time.Now().Add(-(rateLimitBucketTTL + time.Minute))
	rl.buckets["stale"] = &rateLimitBucket{
		hits:     []time.Time{old},
		lastSeen: old,
	}
	rl.lastSweep = old

	allowed, retryAfter := rl.Allow("fresh", rateLimitConfig{Limit: 2, Window: time.Minute})
	if !allowed || retryAfter != 0 {
		t.Fatalf("expected fresh request to be allowed, got allowed=%v retryAfter=%s", allowed, retryAfter)
	}
	if _, ok := rl.buckets["stale"]; ok {
		t.Fatal("expected stale bucket to be pruned during sweep")
	}
}
