package throttle

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryLimiterAppliesIPIdentifierAndDeviceKeysAtomically(t *testing.T) {
	clockValue := time.Unix(100, 0)
	limiter, err := NewMemoryLimiter(time.Minute, 2, func() time.Time { return clockValue })
	if err != nil {
		t.Fatal(err)
	}
	rules := []Rule{{Key: Key("global", "login"), Window: time.Minute, Limit: 2, Global: true}, {Key: Key("identifier", "alice@example.invalid"), Window: time.Minute, Limit: 2}, {Key: Key("device", "device-a"), Window: time.Minute, Limit: 2}}
	if err := limiter.Allow(context.Background(), rules...); err != nil {
		t.Fatal(err)
	}
	if err := limiter.Allow(context.Background(), rules...); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(limiter.Allow(context.Background(), rules...), ErrRateLimited) {
		t.Fatal("third request was not rate limited")
	}
	clockValue = clockValue.Add(time.Minute)
	if err := limiter.Allow(context.Background(), rules...); err != nil {
		t.Fatalf("expired bucket was not reset: %v", err)
	}
}

func TestLimiterFailureIsFailClosed(t *testing.T) {
	limiter, err := NewMemoryLimiter(time.Minute, 2, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	limiter.SetUnavailable(true)
	if !errors.Is(limiter.Allow(context.Background(), Rule{Key: Key("global", "login"), Window: time.Minute, Limit: 2, Global: true}), ErrLimiterUnavailable) {
		t.Fatal("limiter failure was not fail-closed")
	}
}

func TestGlobalDenialDoesNotCreateVariableBucket(t *testing.T) {
	clockValue := time.Unix(200, 0)
	limiter, err := NewMemoryLimiter(time.Minute, 1, func() time.Time { return clockValue })
	if err != nil {
		t.Fatal(err)
	}
	global := Rule{Key: Key("global", "login"), Window: time.Minute, Limit: 1, Global: true}
	if err := limiter.Allow(context.Background(), global, Rule{Key: Key("browser", "first"), Window: time.Minute, Limit: 5}); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(limiter.Allow(context.Background(), global, Rule{Key: Key("browser", "second"), Window: time.Minute, Limit: 5}), ErrRateLimited) {
		t.Fatal("global rule did not deny the second request")
	}
	limiter.mu.Lock()
	_, secondExists := limiter.buckets[Key("browser", "second")]
	limiter.mu.Unlock()
	if secondExists {
		t.Fatal("global denial created the variable bucket before the request was admitted")
	}
	clockValue = clockValue.Add(time.Minute)
	if err := limiter.Allow(context.Background(), global, Rule{Key: Key("browser", "second"), Window: time.Minute, Limit: 5}); err != nil {
		t.Fatalf("global window did not expire: %v", err)
	}
}
