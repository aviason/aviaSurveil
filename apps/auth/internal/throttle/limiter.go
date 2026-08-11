package throttle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"
)

var (
	ErrRateLimited        = errors.New("authentication rate limit exceeded")
	ErrLimiterUnavailable = errors.New("authentication rate limiter unavailable")
	ErrUntrustedForwarded = errors.New("forwarded client identity is not trusted")
)

type Clock func() time.Time

type bucket struct {
	started time.Time
	count   int
}

// MemoryLimiter is a deterministic local admission limiter. The provider will
// replace it with a PostgreSQL/centralized backend before multi-instance
// deployment; a backend failure is deliberately fail-closed for auth calls.
type MemoryLimiter struct {
	mu          sync.Mutex
	clock       Clock
	window      time.Duration
	limit       int
	buckets     map[string]bucket
	unavailable bool
}

func NewMemoryLimiter(window time.Duration, limit int, clock Clock) (*MemoryLimiter, error) {
	if window <= 0 || limit <= 0 {
		return nil, errors.New("limiter window and limit must be positive")
	}
	if clock == nil {
		clock = time.Now
	}
	return &MemoryLimiter{clock: clock, window: window, limit: limit, buckets: make(map[string]bucket)}, nil
}

func (limiter *MemoryLimiter) Allow(_ context.Context, keys ...string) error {
	cleanKeys := uniqueKeys(keys)
	if len(cleanKeys) == 0 {
		return ErrLimiterUnavailable
	}
	now := limiter.clock()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if limiter.unavailable {
		return ErrLimiterUnavailable
	}
	for _, key := range cleanKeys {
		state := limiter.buckets[key]
		if state.started.IsZero() || now.Sub(state.started) >= limiter.window {
			continue
		}
		if state.count >= limiter.limit {
			return ErrRateLimited
		}
	}
	for _, key := range cleanKeys {
		state := limiter.buckets[key]
		if state.started.IsZero() || now.Sub(state.started) >= limiter.window {
			state = bucket{started: now}
		}
		state.count++
		limiter.buckets[key] = state
	}
	return nil
}

func (limiter *MemoryLimiter) SetUnavailable(value bool) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	limiter.unavailable = value
}

func (limiter *MemoryLimiter) Reset() {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	limiter.buckets = make(map[string]bucket)
}

func Key(namespace, value string) string {
	digest := sha256.Sum256([]byte(namespace + "\x00" + strings.TrimSpace(value)))
	return namespace + ":" + hex.EncodeToString(digest[:])
}

type ForwardedHeaders struct {
	RemoteAddr    string
	ForwardedFor  string
	XForwardedFor string
	XRealIP       string
}

// ResolveClientIP trusts forwarded identity only when the immediate peer is
// inside a configured gateway prefix. Untrusted peers cannot spoof a client
// address by sending X-Forwarded-For; malformed trusted headers fail closed.
func ResolveClientIP(headers ForwardedHeaders, trusted []netip.Prefix) (netip.Addr, error) {
	remote, err := parseRemoteAddr(headers.RemoteAddr)
	if err != nil {
		return netip.Addr{}, ErrUntrustedForwarded
	}
	if !containsPrefix(trusted, remote) {
		return remote, nil
	}
	forwarded := strings.TrimSpace(headers.XForwardedFor)
	if forwarded == "" {
		forwarded = strings.TrimSpace(headers.ForwardedFor)
	}
	if forwarded == "" {
		forwarded = strings.TrimSpace(headers.XRealIP)
	}
	if forwarded == "" {
		return netip.Addr{}, ErrUntrustedForwarded
	}
	parts := strings.Split(forwarded, ",")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
		if parts[index] == "" {
			return netip.Addr{}, ErrUntrustedForwarded
		}
		if _, err := netip.ParseAddr(parts[index]); err != nil {
			return netip.Addr{}, ErrUntrustedForwarded
		}
	}
	return netip.ParseAddr(parts[0])
}

func parseRemoteAddr(raw string) (netip.Addr, error) {
	if host, _, err := net.SplitHostPort(strings.TrimSpace(raw)); err == nil {
		return netip.ParseAddr(host)
	}
	return netip.ParseAddr(strings.TrimSpace(raw))
}

func containsPrefix(prefixes []netip.Prefix, address netip.Addr) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func uniqueKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, key)
	}
	return result
}
