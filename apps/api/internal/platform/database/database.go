package database

import (
	"context"
	"fmt"

	"github.com/aviason/aviaSurveil/internal/platform/telemetry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	*pgxpool.Pool
}

type TransactionFunc func(context.Context, pgx.Tx) error

func Open(ctx context.Context, databaseURL string) (*Pool, error) {
	return OpenWithTracer(ctx, databaseURL, nil)
}

func OpenWithTracer(
	ctx context.Context,
	databaseURL string,
	tracer pgx.QueryTracer,
) (*Pool, error) {
	configuration, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL configuration: %w", err)
	}
	configuration.ConnConfig.Tracer = tracer
	pool, err := pgxpool.NewWithConfig(ctx, configuration)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL pool: %w", err)
	}
	return &Pool{Pool: pool}, nil
}

func WithinTransaction(ctx context.Context, pool *Pool, function TransactionFunc) error {
	transaction, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin PostgreSQL transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if traceParent := telemetry.TraceParentFromContext(ctx); traceParent != "" {
		if _, err := transaction.Exec(
			ctx,
			"SELECT set_config('avia.traceparent', $1, true)",
			traceParent,
		); err != nil {
			return fmt.Errorf("bind PostgreSQL trace context: %w", err)
		}
	}
	if err := function(ctx, transaction); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit PostgreSQL transaction: %w", err)
	}
	return nil
}

type Readiness struct {
	Pool                     *Pool
	RequiredMigrationVersion int64
}

func (readiness Readiness) Ready(ctx context.Context) error {
	if readiness.Pool == nil {
		return fmt.Errorf("PostgreSQL pool is not configured")
	}
	if err := readiness.Pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping PostgreSQL: %w", err)
	}
	var version int64
	if err := readiness.Pool.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version); err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	if version != readiness.RequiredMigrationVersion {
		return fmt.Errorf("migration version %d does not match required version %d", version, readiness.RequiredMigrationVersion)
	}
	return nil
}
