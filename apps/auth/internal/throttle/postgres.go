package throttle

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresLimiter provides durable, fail-closed admission control for the
// isolated candidate. Every key in one authentication attempt is updated in
// one transaction, so separate provider processes cannot bypass a bucket.
type PostgresLimiter struct {
	pool   *pgxpool.Pool
	clock  Clock
	window time.Duration
	limit  int
}

func NewPostgresLimiter(pool *pgxpool.Pool, window time.Duration, limit int, clock Clock) (*PostgresLimiter, error) {
	if pool == nil || window <= 0 || limit <= 0 {
		return nil, errors.New("PostgreSQL limiter requires pool, positive window, and positive limit")
	}
	if clock == nil {
		clock = time.Now
	}
	return &PostgresLimiter{pool: pool, clock: clock, window: window, limit: limit}, nil
}

func (limiter *PostgresLimiter) Allow(ctx context.Context, keys ...string) error {
	keys = uniqueKeys(keys)
	if len(keys) == 0 || len(keys) > 8 {
		return ErrLimiterUnavailable
	}
	sort.Strings(keys)
	now := limiter.clock().UTC()
	tx, err := limiter.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ErrLimiterUnavailable
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, key := range keys {
		if _, err := tx.Exec(ctx, `INSERT INTO auth_identity.throttle_buckets(bucket_key, window_started, request_count, updated_at) VALUES ($1, $2, 0, $2) ON CONFLICT (bucket_key) DO NOTHING`, key, now); err != nil {
			return ErrLimiterUnavailable
		}
	}
	rows, err := tx.Query(ctx, `SELECT bucket_key, window_started, request_count FROM auth_identity.throttle_buckets WHERE bucket_key = ANY($1) ORDER BY bucket_key FOR UPDATE`, keys)
	if err != nil {
		return ErrLimiterUnavailable
	}
	defer rows.Close()
	type state struct {
		key     string
		started time.Time
		count   int
	}
	states := make([]state, 0, len(keys))
	for rows.Next() {
		var entry state
		if err := rows.Scan(&entry.key, &entry.started, &entry.count); err != nil {
			return ErrLimiterUnavailable
		}
		if !now.Before(entry.started.Add(limiter.window)) {
			entry.started, entry.count = now, 0
		}
		if entry.count >= limiter.limit {
			return ErrRateLimited
		}
		states = append(states, entry)
	}
	if err := rows.Err(); err != nil || len(states) != len(keys) {
		return ErrLimiterUnavailable
	}
	for _, entry := range states {
		if _, err := tx.Exec(ctx, `UPDATE auth_identity.throttle_buckets SET window_started=$2, request_count=$3, updated_at=$4 WHERE bucket_key=$1`, entry.key, entry.started, entry.count+1, now); err != nil {
			return fmt.Errorf("%w: update durable throttle bucket", ErrLimiterUnavailable)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrLimiterUnavailable
	}
	return nil
}
