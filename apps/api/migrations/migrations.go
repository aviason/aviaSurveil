package migrations

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
)

const LatestVersion int64 = 20
const advisoryLockID int64 = 36020260721

//go:embed *.up.sql
var migrationFiles embed.FS

type migration struct {
	version int64
	name    string
	sql     string
}

func Apply(ctx context.Context, pool *database.Pool) error {
	connection, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer connection.Release()
	if _, err := connection.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockID); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() { _, _ = connection.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockID) }()

	if _, err := connection.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version bigint PRIMARY KEY,
			name text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	var current int64
	if err := connection.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&current); err != nil {
		return fmt.Errorf("read migration ledger: %w", err)
	}
	available, err := load()
	if err != nil {
		return err
	}
	for _, candidate := range available {
		if candidate.version <= current {
			continue
		}
		transaction, err := connection.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", candidate.name, err)
		}
		if _, err := transaction.Exec(ctx, candidate.sql); err != nil {
			_ = transaction.Rollback(ctx)
			return fmt.Errorf("execute migration %s: %w", candidate.name, err)
		}
		if _, err := transaction.Exec(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", candidate.version, candidate.name); err != nil {
			_ = transaction.Rollback(ctx)
			return fmt.Errorf("record migration %s: %w", candidate.name, err)
		}
		if err := transaction.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration %s: %w", candidate.name, err)
		}
		current = candidate.version
	}
	if current != LatestVersion {
		return fmt.Errorf("migration version %d does not match embedded latest version %d", current, LatestVersion)
	}
	return nil
}

func CurrentVersion(ctx context.Context, pool *database.Pool) (int64, error) {
	var version int64
	if err := pool.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version); err != nil {
		return 0, fmt.Errorf("read migration version: %w", err)
	}
	return version, nil
}

func load() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, ".")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}
	loaded := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		prefix, _, ok := strings.Cut(filepath.Base(entry.Name()), "_")
		if !ok {
			return nil, fmt.Errorf("migration %s has no numeric prefix", entry.Name())
		}
		version, err := strconv.ParseInt(prefix, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse migration version %s: %w", entry.Name(), err)
		}
		contents, err := migrationFiles.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		loaded = append(loaded, migration{version: version, name: entry.Name(), sql: string(contents)})
	}
	sort.Slice(loaded, func(left, right int) bool { return loaded[left].version < loaded[right].version })
	if len(loaded) == 0 || loaded[len(loaded)-1].version != LatestVersion {
		return nil, fmt.Errorf("embedded migration set does not end at version %d", LatestVersion)
	}
	for index, candidate := range loaded {
		expected := int64(index + 1)
		if candidate.version != expected {
			return nil, fmt.Errorf("migration sequence has version %d, expected %d", candidate.version, expected)
		}
	}
	return loaded, nil
}
