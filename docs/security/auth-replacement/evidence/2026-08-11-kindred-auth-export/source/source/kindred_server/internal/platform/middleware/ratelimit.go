package middleware

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	apperrors "kindred_server/internal/platform/errors"
	"kindred_server/pkg/response"
)

// Store is the backing counter for the rate limiter. Implementations must
// return the bucket's new count *after* incrementing, along with the time
// at which the current window expires.
type Store interface {
	Incr(ctx context.Context, key string, windowSeconds int64) (count int64, resetAt time.Time, err error)
}

// RateLimitConfig configures a fixed-window rate limiter.
//   - Max: maximum requests allowed per window
//   - Window: the rolling window duration
//   - KeyFunc: derives the bucket key (defaults to client IP)
//   - Store: counter backend. Defaults to an in-memory store (per-instance).
type RateLimitConfig struct {
	Max     int
	Window  time.Duration
	KeyFunc func(*http.Request) string
	Store   Store
}

// RateLimiter returns a middleware enforcing the given RateLimitConfig.
func RateLimiter(cfg RateLimitConfig) func(http.Handler) http.Handler {
	if cfg.Max <= 0 || cfg.Window <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = clientIP
	}
	if cfg.Store == nil {
		cfg.Store = NewMemoryStore()
	}
	windowSeconds := int64(cfg.Window / time.Second)
	if windowSeconds <= 0 {
		windowSeconds = 1
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := cfg.KeyFunc(r)
			count, resetAt, err := cfg.Store.Incr(r.Context(), key, windowSeconds)
			if err != nil {
				// Fail open: if the counter backend is unavailable we serve
				// the request rather than locking every user out.
				next.ServeHTTP(w, r)
				return
			}
			if count > int64(cfg.Max) {
				w.Header().Set("Retry-After", formatSeconds(time.Until(resetAt)))
				response.Error(w, &apperrors.AppError{
					Status:  http.StatusTooManyRequests,
					Code:    "rate_limited",
					Message: "too many requests, please try again later",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MemoryStore is an in-process Store suitable for local dev or single-instance
// deployments. It resets each bucket when its window elapses.
type MemoryStore struct {
	mu      sync.Mutex
	buckets map[string]*memoryBucket
}

type memoryBucket struct {
	count   int64
	resetAt time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{buckets: make(map[string]*memoryBucket)}
}

func (m *MemoryStore) Incr(_ context.Context, key string, windowSeconds int64) (int64, time.Time, error) {
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.buckets[key]
	if !ok || now.After(b.resetAt) {
		b = &memoryBucket{count: 0, resetAt: now.Add(time.Duration(windowSeconds) * time.Second)}
		m.buckets[key] = b
	}
	b.count++
	return b.count, b.resetAt, nil
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx >= 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func formatSeconds(d time.Duration) string {
	secs := int(d.Seconds())
	if secs < 1 {
		secs = 1
	}
	return strconv.Itoa(secs)
}
