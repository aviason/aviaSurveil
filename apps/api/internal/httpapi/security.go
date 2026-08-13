package httpapi

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const apiContentSecurityPolicy = "default-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'"

type applicationSecurityOptions struct {
	clock                     func() time.Time
	window                    time.Duration
	mutationRequestsPerWindow int
}

type rateBucket struct {
	startedAt time.Time
	count     int
}

type applicationRateLimiter struct {
	mutex    sync.Mutex
	clock    func() time.Time
	window   time.Duration
	mutation int
	buckets  map[string]rateBucket
}

func newApplicationRateLimiter(options applicationSecurityOptions) *applicationRateLimiter {
	clock := options.clock
	if clock == nil {
		clock = time.Now
	}
	window := options.window
	if window <= 0 {
		window = time.Minute
	}
	mutation := options.mutationRequestsPerWindow
	if mutation <= 0 {
		mutation = 300
	}
	return &applicationRateLimiter{
		clock: clock, window: window, mutation: mutation,
		buckets: make(map[string]rateBucket),
	}
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "no-store")
		writer.Header().Set("Content-Security-Policy", apiContentSecurityPolicy)
		writer.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(writer, request)
	})
}

func (limiter *applicationRateLimiter) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		class, maximum, limited := limiter.classify(request)
		if !limited {
			next.ServeHTTP(writer, request)
			return
		}
		allowed, retryAfter := limiter.allow(class+":"+socketPeer(request.RemoteAddr), maximum)
		if !allowed {
			writer.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			writeProblem(writer, http.StatusTooManyRequests, "Too many requests", "request rate exceeded the local candidate policy", "RATE_LIMITED")
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (limiter *applicationRateLimiter) classify(request *http.Request) (string, int, bool) {
	if strings.HasPrefix(request.URL.Path, "/v1/") && isMutation(request.Method) {
		return "mutation", limiter.mutation, true
	}
	return "", 0, false
}

func (limiter *applicationRateLimiter) allow(key string, maximum int) (bool, int) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()
	now := limiter.clock().UTC()
	for candidate, bucket := range limiter.buckets {
		if !now.Before(bucket.startedAt.Add(limiter.window)) {
			delete(limiter.buckets, candidate)
		}
	}
	bucket, exists := limiter.buckets[key]
	if !exists {
		bucket = rateBucket{startedAt: now}
	}
	if bucket.count >= maximum {
		remaining := bucket.startedAt.Add(limiter.window).Sub(now).Seconds()
		return false, max(1, int(math.Ceil(remaining)))
	}
	bucket.count++
	limiter.buckets[key] = bucket
	return true, 0
}

func socketPeer(remoteAddress string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddress))
	if err == nil && host != "" {
		return host
	}
	if trimmed := strings.TrimSpace(remoteAddress); trimmed != "" {
		return trimmed
	}
	return "unknown"
}
