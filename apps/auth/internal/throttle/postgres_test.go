package throttle

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/auth/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgreSQLLimiterPersistsAtomicBuckets(t *testing.T) {
	databaseURL := strings.TrimSpace(os.Getenv("AVIA_AUTH_TEST_DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("not run: AVIA_AUTH_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE auth_identity.throttle_buckets`); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_725_000_000, 0).UTC()
	limiter, err := NewPostgresLimiter(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	rules := []Rule{{Key: Key("global", "login"), Window: time.Minute, Limit: 2, Global: true}, {Key: Key("identifier", "operator@example.invalid"), Window: time.Minute, Limit: 2}}
	if err := limiter.Allow(ctx, rules...); err != nil {
		t.Fatal(err)
	}
	if err := limiter.Allow(ctx, rules...); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(limiter.Allow(ctx, rules...), ErrRateLimited) {
		t.Fatal("third durable attempt was not rate limited")
	}
	now = now.Add(time.Minute)
	if err := limiter.Allow(ctx, rules...); err != nil {
		t.Fatalf("expired durable bucket was not reset: %v", err)
	}
}
