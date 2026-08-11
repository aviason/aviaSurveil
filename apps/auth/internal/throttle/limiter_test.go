package throttle

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"
)

func TestMemoryLimiterAppliesIPIdentifierAndDeviceKeysAtomically(t *testing.T) {
	clockValue := time.Unix(100, 0)
	limiter, err := NewMemoryLimiter(time.Minute, 2, func() time.Time { return clockValue })
	if err != nil {
		t.Fatal(err)
	}
	keys := []string{Key("ip", "192.0.2.1"), Key("identifier", "alice@example.invalid"), Key("device", "device-a")}
	if err := limiter.Allow(context.Background(), keys...); err != nil {
		t.Fatal(err)
	}
	if err := limiter.Allow(context.Background(), keys...); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(limiter.Allow(context.Background(), keys...), ErrRateLimited) {
		t.Fatal("third request was not rate limited")
	}
	clockValue = clockValue.Add(time.Minute)
	if err := limiter.Allow(context.Background(), keys...); err != nil {
		t.Fatalf("expired bucket was not reset: %v", err)
	}
}

func TestLimiterFailureIsFailClosed(t *testing.T) {
	limiter, err := NewMemoryLimiter(time.Minute, 2, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	limiter.SetUnavailable(true)
	if !errors.Is(limiter.Allow(context.Background(), Key("ip", "192.0.2.1")), ErrLimiterUnavailable) {
		t.Fatal("limiter failure was not fail-closed")
	}
}

func TestForwardedClientIdentityRequiresTrustedGateway(t *testing.T) {
	trusted := []netip.Prefix{netip.MustParsePrefix("198.51.100.0/24")}
	remote, err := ResolveClientIP(ForwardedHeaders{
		RemoteAddr:    "203.0.113.9:443",
		XForwardedFor: "192.0.2.4",
	}, trusted)
	if err != nil || remote.String() != "203.0.113.9" {
		t.Fatalf("untrusted forwarded address = %s/%v", remote, err)
	}
	forwarded, err := ResolveClientIP(ForwardedHeaders{
		RemoteAddr:    "198.51.100.9:443",
		XForwardedFor: "192.0.2.4, 198.51.100.8",
	}, trusted)
	if err != nil || forwarded.String() != "192.0.2.4" {
		t.Fatalf("trusted forwarded address = %s/%v", forwarded, err)
	}
	if _, err := ResolveClientIP(ForwardedHeaders{RemoteAddr: "198.51.100.9:443", XForwardedFor: "not-an-ip"}, trusted); !errors.Is(err, ErrUntrustedForwarded) {
		t.Fatalf("malformed trusted header error = %v", err)
	}
}
