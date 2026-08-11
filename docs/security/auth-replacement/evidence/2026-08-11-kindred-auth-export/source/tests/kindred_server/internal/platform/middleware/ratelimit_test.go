package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"kindred_server/internal/platform/middleware"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func doReq(h http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	h.ServeHTTP(rec, req)
	return rec
}

func TestRateLimiter_UnderLimitPasses(t *testing.T) {
	mw := middleware.RateLimiter(middleware.RateLimitConfig{Max: 3, Window: time.Minute})
	h := mw(okHandler())
	for i := 0; i < 3; i++ {
		rec := doReq(h, "1.2.3.4:1000")
		if rec.Code != http.StatusOK {
			t.Fatalf("req %d: status = %d, want 200", i, rec.Code)
		}
	}
}

func TestRateLimiter_OverLimitReturns429WithRetryAfter(t *testing.T) {
	mw := middleware.RateLimiter(middleware.RateLimitConfig{Max: 2, Window: time.Minute})
	h := mw(okHandler())
	for i := 0; i < 2; i++ {
		_ = doReq(h, "1.2.3.4:1000")
	}
	rec := doReq(h, "1.2.3.4:1000")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	ra := rec.Header().Get("Retry-After")
	if ra == "" {
		t.Fatal("missing Retry-After header")
	}
	n, err := strconv.Atoi(ra)
	if err != nil || n < 1 {
		t.Errorf("Retry-After = %q (want positive int)", ra)
	}
}

func TestRateLimiter_PerIPIsolation(t *testing.T) {
	mw := middleware.RateLimiter(middleware.RateLimitConfig{Max: 1, Window: time.Minute})
	h := mw(okHandler())
	if rec := doReq(h, "1.1.1.1:1000"); rec.Code != http.StatusOK {
		t.Fatalf("ip1 first: status = %d", rec.Code)
	}
	if rec := doReq(h, "1.1.1.1:1000"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("ip1 second: status = %d, want 429", rec.Code)
	}
	// Different IP must still pass.
	if rec := doReq(h, "2.2.2.2:1000"); rec.Code != http.StatusOK {
		t.Fatalf("ip2 first: status = %d, want 200", rec.Code)
	}
}

func TestRateLimiter_WindowResets(t *testing.T) {
	// Use a 1s window; it's the smallest the implementation supports.
	mw := middleware.RateLimiter(middleware.RateLimitConfig{Max: 1, Window: time.Second})
	h := mw(okHandler())
	if rec := doReq(h, "9.9.9.9:1000"); rec.Code != http.StatusOK {
		t.Fatalf("first: status = %d", rec.Code)
	}
	if rec := doReq(h, "9.9.9.9:1000"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second within window: status = %d, want 429", rec.Code)
	}
	time.Sleep(1100 * time.Millisecond)
	if rec := doReq(h, "9.9.9.9:1000"); rec.Code != http.StatusOK {
		t.Errorf("after window reset: status = %d, want 200", rec.Code)
	}
}

func TestRateLimiter_ZeroMaxDisablesLimiter(t *testing.T) {
	mw := middleware.RateLimiter(middleware.RateLimitConfig{Max: 0, Window: time.Minute})
	h := mw(okHandler())
	for i := 0; i < 50; i++ {
		if rec := doReq(h, "1.2.3.4:1"); rec.Code != http.StatusOK {
			t.Fatalf("req %d: status = %d", i, rec.Code)
		}
	}
}

func TestMemoryStore_IncrementsAndResets(t *testing.T) {
	s := middleware.NewMemoryStore()
	c1, r1, err := s.Incr(nil, "k", 60)
	if err != nil || c1 != 1 {
		t.Fatalf("first incr: c=%d err=%v", c1, err)
	}
	c2, r2, _ := s.Incr(nil, "k", 60)
	if c2 != 2 {
		t.Errorf("second incr: c=%d, want 2", c2)
	}
	if !r1.Equal(r2) {
		t.Errorf("resetAt drifted within window: %v vs %v", r1, r2)
	}
	c3, _, _ := s.Incr(nil, "other", 60)
	if c3 != 1 {
		t.Errorf("isolated key: c=%d, want 1", c3)
	}
}

func TestRateLimiter_CustomKeyFunc(t *testing.T) {
	keyFn := func(r *http.Request) string { return r.Header.Get("X-User") }
	mw := middleware.RateLimiter(middleware.RateLimitConfig{Max: 1, Window: time.Minute, KeyFunc: keyFn})
	h := mw(okHandler())
	req := func(user string) int {
		rec := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-User", user)
		h.ServeHTTP(rec, r)
		return rec.Code
	}
	if c := req("alice"); c != http.StatusOK {
		t.Fatalf("alice 1: %d", c)
	}
	if c := req("alice"); c != http.StatusTooManyRequests {
		t.Fatalf("alice 2: %d, want 429", c)
	}
	if c := req("bob"); c != http.StatusOK {
		t.Fatalf("bob 1: %d", c)
	}
}
