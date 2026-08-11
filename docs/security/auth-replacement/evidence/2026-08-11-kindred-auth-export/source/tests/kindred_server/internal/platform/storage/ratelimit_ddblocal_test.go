//go:build dynamodblocal

package storage

import (
	"context"
	"testing"
	"time"
)

func TestRateLimitStore_AtomicIncr(t *testing.T) {
	fx := setupDDB(t)
	store := NewRateLimitStoreWithClient(fx.Client, fx.RateTable)
	ctx := context.Background()

	for i := int64(1); i <= 3; i++ {
		count, _, err := store.Incr(ctx, "client-A", 60)
		if err != nil {
			t.Fatalf("incr %d: %v", i, err)
		}
		if count != i {
			t.Fatalf("incr %d: got %d want %d", i, count, i)
		}
	}
}

func TestRateLimitStore_IndependentKeys(t *testing.T) {
	fx := setupDDB(t)
	store := NewRateLimitStoreWithClient(fx.Client, fx.RateTable)
	ctx := context.Background()

	a, _, err := store.Incr(ctx, "key-A", 60)
	if err != nil || a != 1 {
		t.Fatalf("key-A: count=%d err=%v", a, err)
	}
	b, _, err := store.Incr(ctx, "key-B", 60)
	if err != nil || b != 1 {
		t.Fatalf("key-B did not get its own counter: count=%d err=%v", b, err)
	}
	a2, _, err := store.Incr(ctx, "key-A", 60)
	if err != nil || a2 != 2 {
		t.Fatalf("key-A second incr: count=%d err=%v", a2, err)
	}
}

func TestRateLimitStore_WindowRollover(t *testing.T) {
	fx := setupDDB(t)
	store := NewRateLimitStoreWithClient(fx.Client, fx.RateTable)
	ctx := context.Background()

	// 1-second window: cross a boundary by sleeping just over 1s.
	first, reset1, err := store.Incr(ctx, "win-key", 1)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if first != 1 {
		t.Fatalf("first count: %d", first)
	}
	// Sleep until past the reset boundary.
	wait := time.Until(reset1) + 200*time.Millisecond
	if wait < 200*time.Millisecond {
		wait = 1200 * time.Millisecond
	}
	time.Sleep(wait)
	next, _, err := store.Incr(ctx, "win-key", 1)
	if err != nil {
		t.Fatalf("post-window: %v", err)
	}
	if next != 1 {
		t.Fatalf("expected counter to reset across window, got %d", next)
	}
}
