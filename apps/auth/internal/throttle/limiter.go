package throttle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrRateLimited        = errors.New("authentication rate limit exceeded")
	ErrLimiterUnavailable = errors.New("authentication rate limiter unavailable")
)

type Clock func() time.Time

type Limiter interface {
	Allow(context.Context, ...Rule) error
}

// Rule is one operation-specific admission bucket. Key must already be a
// domain-separated, namespaced hash; raw attacker input must never be stored
// in the durable bucket table.
type Rule struct {
	Key    string
	Window time.Duration
	Limit  int
	Global bool
}

type bucket struct {
	started time.Time
	count   int
	window  time.Duration
}

// MemoryLimiter is a deterministic local admission limiter. The provider will
// replace it with a PostgreSQL/centralized backend before multi-instance
// deployment; a backend failure is deliberately fail-closed for auth calls.
type MemoryLimiter struct {
	mu          sync.Mutex
	clock       Clock
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
	return &MemoryLimiter{clock: clock, buckets: make(map[string]bucket)}, nil
}

func (limiter *MemoryLimiter) Allow(_ context.Context, rules ...Rule) error {
	ordered, err := normalizeRules(rules)
	if err != nil {
		return err
	}
	now := limiter.clock()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if limiter.unavailable {
		return ErrLimiterUnavailable
	}
	for _, rule := range ordered {
		state := limiter.buckets[rule.Key]
		if !state.started.IsZero() && now.Sub(state.started) < rule.Window && state.count >= rule.Limit {
			return ErrRateLimited
		}
	}
	for _, rule := range ordered {
		state := limiter.buckets[rule.Key]
		if state.started.IsZero() || now.Sub(state.started) >= rule.Window {
			state = bucket{started: now, window: rule.Window}
		} else {
			state.window = rule.Window
		}
		state.count++
		limiter.buckets[rule.Key] = state
	}
	return nil
}

// Cleanup removes no more than limit stale buckets. It is intentionally
// separate from Allow so admission never performs an unbounded scan.
func (limiter *MemoryLimiter) Cleanup(at time.Time, limit int) (int, error) {
	if limit < 1 {
		return 0, ErrLimiterUnavailable
	}
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	keys := make([]string, 0, len(limiter.buckets))
	for key, state := range limiter.buckets {
		if !state.started.IsZero() && !at.Before(state.started.Add(state.window)) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) > limit {
		keys = keys[:limit]
	}
	for _, key := range keys {
		delete(limiter.buckets, key)
	}
	return len(keys), nil
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

func normalizeRules(rules []Rule) ([]Rule, error) {
	if len(rules) == 0 || len(rules) > 8 {
		return nil, ErrLimiterUnavailable
	}
	ordered := make([]Rule, 0, len(rules))
	seen := make(map[string]Rule, len(rules))
	for _, rule := range rules {
		rule.Key = strings.TrimSpace(rule.Key)
		if rule.Key == "" || len(rule.Key) > 256 || rule.Window <= 0 || rule.Limit <= 0 {
			return nil, ErrLimiterUnavailable
		}
		if previous, exists := seen[rule.Key]; exists {
			if previous.Window != rule.Window || previous.Limit != rule.Limit || previous.Global != rule.Global {
				return nil, ErrLimiterUnavailable
			}
			continue
		}
		seen[rule.Key] = rule
		ordered = append(ordered, rule)
	}
	sort.SliceStable(ordered, func(left, right int) bool {
		if ordered[left].Global != ordered[right].Global {
			return ordered[left].Global
		}
		return ordered[left].Key < ordered[right].Key
	})
	return ordered, nil
}
