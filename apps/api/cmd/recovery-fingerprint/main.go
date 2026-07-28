// Command recovery-fingerprint emits a deterministic digest of authoritative
// application records for backup and isolated-restore comparison.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var authoritativeTables = []string{
	"schema_migrations",
	"organizations",
	"inspections",
	"findings",
	"cap_revisions",
	"evidence_versions",
	"object_metadata",
	"audit_events",
	"outbox_messages",
}

type tableFingerprint struct {
	Rows   int64  `json:"rows"`
	SHA256 string `json:"sha256"`
}

type applicationComponent struct {
	ApplicationDatabase struct {
		Status          string                      `json:"status"`
		ArtifactStatus  string                      `json:"artifactStatus"`
		RecoveryPointID string                      `json:"recoveryPointId"`
		GeneratedAt     string                      `json:"generatedAt"`
		SHA256          string                      `json:"sha256"`
		Tables          map[string]tableFingerprint `json:"tables"`
	} `json:"applicationDatabase"`
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "recovery fingerprint: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if os.Getenv("AVIA_ENVIRONMENT") != "local-candidate" ||
		os.Getenv("AVIA_ENABLE_RECOVERY_BACKUP") != "true" {
		return errors.New(
			"AVIA_ENVIRONMENT=local-candidate and AVIA_ENABLE_RECOVERY_BACKUP=true are required",
		)
	}
	recoveryPointID := strings.TrimSpace(os.Getenv("AVIA_RECOVERY_POINT_ID"))
	if recoveryPointID == "" {
		return errors.New("AVIA_RECOVERY_POINT_ID is required")
	}
	databaseURL := strings.TrimSpace(os.Getenv("AVIA_DATABASE_URL"))
	if databaseURL == "" {
		return errors.New("AVIA_DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("open application database: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping application database: %w", err)
	}

	tables := make(map[string]tableFingerprint, len(authoritativeTables))
	for _, table := range authoritativeTables {
		value, err := fingerprintTable(ctx, pool, table)
		if err != nil {
			return err
		}
		tables[table] = value
	}
	overall, err := fingerprintTables(tables)
	if err != nil {
		return err
	}

	var component applicationComponent
	component.ApplicationDatabase.Status = "verified locally"
	component.ApplicationDatabase.ArtifactStatus = "candidate-only"
	component.ApplicationDatabase.RecoveryPointID = recoveryPointID
	component.ApplicationDatabase.GeneratedAt = time.Now().UTC().Format(time.RFC3339Nano)
	component.ApplicationDatabase.SHA256 = overall
	component.ApplicationDatabase.Tables = tables
	return json.NewEncoder(os.Stdout).Encode(component)
}

func fingerprintTable(
	ctx context.Context,
	pool *pgxpool.Pool,
	table string,
) (tableFingerprint, error) {
	identifier := pgx.Identifier{table}.Sanitize()
	query := fmt.Sprintf(`
		SELECT
			COUNT(*)::bigint,
			COALESCE(
				jsonb_agg(to_jsonb(row_value) ORDER BY to_jsonb(row_value)::text),
				'[]'::jsonb
			)::text
		FROM %s AS row_value
	`, identifier)
	var rows int64
	var content []byte
	if err := pool.QueryRow(ctx, query).Scan(&rows, &content); err != nil {
		return tableFingerprint{}, fmt.Errorf("fingerprint table %s: %w", table, err)
	}
	digest, err := fingerprintJSON(content)
	if err != nil {
		return tableFingerprint{}, fmt.Errorf("canonicalize table %s: %w", table, err)
	}
	return tableFingerprint{Rows: rows, SHA256: digest}, nil
}

func fingerprintJSON(source []byte) (string, error) {
	var value any
	if err := json.Unmarshal(source, &value); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func fingerprintTables(
	tables map[string]tableFingerprint,
) (string, error) {
	names := make([]string, 0, len(tables))
	for name := range tables {
		names = append(names, name)
	}
	sort.Strings(names)
	ordered := make([]struct {
		Name string           `json:"name"`
		Data tableFingerprint `json:"data"`
	}, 0, len(names))
	for _, name := range names {
		ordered = append(ordered, struct {
			Name string           `json:"name"`
			Data tableFingerprint `json:"data"`
		}{Name: name, Data: tables[name]})
	}
	encoded, err := json.Marshal(ordered)
	if err != nil {
		return "", fmt.Errorf("encode application fingerprints: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
