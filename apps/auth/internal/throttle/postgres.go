package throttle

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresLimiter provides durable, fail-closed admission control for the
// isolated candidate. Every key in one authentication attempt is updated in
// one transaction, so separate provider processes cannot bypass a bucket.
type PostgresLimiter struct {
	pool  *pgxpool.Pool
	clock Clock
}

func NewPostgresLimiter(pool *pgxpool.Pool, clock Clock) (*PostgresLimiter, error) {
	if pool == nil {
		return nil, errors.New("PostgreSQL limiter requires a pool")
	}
	if clock == nil {
		clock = time.Now
	}
	return &PostgresLimiter{pool: pool, clock: clock}, nil
}

func (limiter *PostgresLimiter) Allow(ctx context.Context, rules ...Rule) error {
	ordered, err := normalizeRules(rules)
	if err != nil {
		return err
	}
	now := limiter.clock().UTC()
	tx, err := limiter.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ErrLimiterUnavailable
	}
	defer func() { _ = tx.Rollback(ctx) }()
	type state struct {
		rule    Rule
		started time.Time
		ends    time.Time
		count   int
	}
	states := make([]state, 0, len(ordered))
	// Global rules are inserted and checked first. If one is denied the
	// transaction rolls back before any attacker-variable bucket is created.
	for pass := 0; pass < 2; pass++ {
		for _, rule := range ordered {
			if (pass == 0) != rule.Global {
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO auth_identity.throttle_buckets(bucket_key, window_started, window_ends_at, request_count, updated_at) VALUES ($1, $2, $3, 0, $2) ON CONFLICT (bucket_key) DO NOTHING`, rule.Key, now, now.Add(rule.Window)); err != nil {
				return ErrLimiterUnavailable
			}
			var entry state
			entry.rule = rule
			if err := tx.QueryRow(ctx, `SELECT window_started, window_ends_at, request_count FROM auth_identity.throttle_buckets WHERE bucket_key = $1 FOR UPDATE`, rule.Key).Scan(&entry.started, &entry.ends, &entry.count); err != nil {
				return ErrLimiterUnavailable
			}
			if !now.Before(entry.ends) {
				entry.started, entry.ends, entry.count = now, now.Add(rule.Window), 0
			}
			if entry.count >= rule.Limit {
				return ErrRateLimited
			}
			states = append(states, entry)
		}
	}
	for _, entry := range states {
		if _, err := tx.Exec(ctx, `UPDATE auth_identity.throttle_buckets SET window_started=$2, window_ends_at=$3, request_count=$4, updated_at=$5 WHERE bucket_key=$1`, entry.rule.Key, entry.started, entry.ends, entry.count+1, now); err != nil {
			return fmt.Errorf("%w: update durable throttle bucket", ErrLimiterUnavailable)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrLimiterUnavailable
	}
	return nil
}

// Cleanup removes no more than limit buckets whose last update is at or
// before before. The bounded statement is safe to repeat after a restart.
func (limiter *PostgresLimiter) Cleanup(ctx context.Context, before time.Time, limit int) (int64, error) {
	if limit < 1 || limit > 10000 {
		return 0, ErrLimiterUnavailable
	}
	command, err := limiter.pool.Exec(ctx, `
		WITH stale AS (
			SELECT bucket_key FROM auth_identity.throttle_buckets
			WHERE window_ends_at <= $1
			ORDER BY window_ends_at, bucket_key
			LIMIT $2
		)
		DELETE FROM auth_identity.throttle_buckets bucket
		USING stale
		WHERE bucket.bucket_key = stale.bucket_key
		  AND bucket.window_ends_at <= $1
	`, before.UTC(), limit)
	if err != nil {
		return 0, fmt.Errorf("cleanup durable throttle buckets: %w", err)
	}
	return command.RowsAffected(), nil
}
