package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
	"github.com/jackc/pgx/v5"
)

func TestDataFeedMigrationCreatesImmutableEventAndDeliveryHistory(t *testing.T) {
	pool := createTestDatabase(t, "datafeed_task4")
	ctx := context.Background()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if migrations.LatestVersion != 28 {
		t.Fatalf("latest migration = %d, want 28", migrations.LatestVersion)
	}
	for _, table := range []string{"datafeed_event_type_catalog", "datafeed_events", "datafeed_delivery_state", "datafeed_delivery_attempts", "datafeed_replay_tombstones"} {
		var relation *string
		if err := pool.QueryRow(ctx, "SELECT to_regclass($1)::text", "public."+table).Scan(&relation); err != nil {
			t.Fatalf("find %s: %v", table, err)
		}
		if relation == nil {
			t.Errorf("missing Task 4 table %s", table)
		}
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO datafeed_events (
			event_id, contract_version, event_type, event_version, source_system,
			tenant_id, owning_organization_id, visibility_purpose_code, operation_id,
			correlation_id, aggregate_type, aggregate_id, aggregate_revision,
			effective_at, known_at, occurred_at, emitted_at, entity_refs, state_after,
			payload_ciphertext, payload_nonce, payload_key_ref, payload_sha256,
			canonical_event_sha256
		) VALUES (
			'10000000-0000-4000-8000-000000000001', '3.0.0', 'audit.planned', 1,
			'aviasurveil-production-api', 'tenant-a', 'organization-a', 'regulated_oversight',
			'operation-a', '20000000-0000-4000-8000-000000000001', 'audit', 'audit-a', 1,
			now(), now(), now(), now(), '{"audit_id":"audit-a"}', 'audit_planned',
			decode('00', 'hex'), decode(repeat('00', 12), 'hex'), 'test-key',
			repeat('a', 64), repeat('b', 64)
		)
	`)
	if err != nil {
		t.Fatalf("append datafeed event: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE datafeed_events SET aggregate_id = 'mutated' WHERE event_id = '10000000-0000-4000-8000-000000000001'`); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("event mutation error = %v, want immutable rejection", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE datafeed_delivery_state SET status = 'ACKNOWLEDGED' WHERE event_id = '10000000-0000-4000-8000-000000000001'`); err == nil || !strings.Contains(err.Error(), "lease") {
		t.Fatalf("unleased acknowledgement error = %v, want lease rejection", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE datafeed_delivery_state
		SET status = 'LEASED', lease_generation = 1, lease_expires_at = now() + interval '1 minute'
		WHERE event_id = '10000000-0000-4000-8000-000000000001'
	`); err != nil {
		t.Fatalf("lease event: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE datafeed_delivery_state
		SET status = 'ACKNOWLEDGED', lease_expires_at = NULL,
		    acknowledgement_receipt_digest = repeat('c', 64), acknowledged_at = now()
		WHERE event_id = '10000000-0000-4000-8000-000000000001' AND lease_generation = 1
	`); err != nil {
		t.Fatalf("acknowledge leased event: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO datafeed_delivery_attempts (
			attempt_id, event_id, lease_generation, outcome_code, occurred_at
		) VALUES ('30000000-0000-4000-8000-000000000001', '10000000-0000-4000-8000-000000000001', 1, 'accepted', now())
	`); err != nil {
		t.Fatalf("append delivery attempt: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM datafeed_delivery_attempts WHERE attempt_id = '30000000-0000-4000-8000-000000000001'`); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("attempt deletion error = %v, want immutable rejection", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO datafeed_events (
			event_id, contract_version, event_type, event_version, source_system,
			tenant_id, owning_organization_id, visibility_purpose_code, operation_id,
			correlation_id, aggregate_type, aggregate_id, aggregate_revision,
			effective_at, known_at, occurred_at, emitted_at, entity_refs, state_after,
			payload_ciphertext, payload_nonce, payload_key_ref, payload_sha256,
			canonical_event_sha256
		) VALUES (
			'10000000-0000-4000-8000-000000000003', '3.0.0', 'audit.planned', 1,
			'aviasurveil-production-api', 'tenant-a', 'organization-a', 'regulated_oversight',
			'operation-tombstone', '20000000-0000-4000-8000-000000000003', 'audit', 'audit-tombstone', 1,
			now(), now(), now(), now(), '{"audit_id":"audit-tombstone"}', 'audit_planned',
			decode('00', 'hex'), decode(repeat('00', 12), 'hex'), 'test-key',
			repeat('a', 64), repeat('b', 64)
		)
	`); err != nil {
		t.Fatalf("append tombstone candidate: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO datafeed_replay_tombstones (event_id, tombstone_reason_code, recorded_at)
		VALUES ('10000000-0000-4000-8000-000000000003', 'legal_hold_release', now())
	`); err != nil {
		t.Fatalf("append replay tombstone: %v", err)
	}
	var state string
	if err := pool.QueryRow(ctx, `SELECT status FROM datafeed_delivery_state WHERE event_id = '10000000-0000-4000-8000-000000000003'`).Scan(&state); err != nil || state != "TOMBSTONED" {
		t.Fatalf("tombstoned delivery state = %q, err=%v", state, err)
	}
}

func TestDataFeedPublisherMigrationFencesRetryAndQuarantineTransitions(t *testing.T) {
	pool := createTestDatabase(t, "datafeed_task5_delivery")
	ctx := context.Background()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if migrations.LatestVersion != 28 {
		t.Fatalf("latest migration = %d, want 28", migrations.LatestVersion)
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO datafeed_events (
			event_id, contract_version, event_type, event_version, source_system,
			tenant_id, owning_organization_id, visibility_purpose_code, operation_id,
			correlation_id, aggregate_type, aggregate_id, aggregate_revision,
			effective_at, known_at, occurred_at, emitted_at, entity_refs, state_after,
			payload_ciphertext, payload_nonce, payload_key_ref, payload_sha256, canonical_event_sha256
		) VALUES (
			'10000000-0000-4000-8000-000000000019', '3.0.0', 'audit.planned', 1, 'aviasurveil-production-api',
			'tenant-task5', 'organization-task5', 'regulated_oversight', 'operation-task5',
			'20000000-0000-4000-8000-000000000019', 'audit', 'audit-task5', 1,
			now(), now(), now(), now(), '{"audit_id":"audit-task5"}', 'audit_planned',
			decode('00', 'hex'), decode(repeat('00', 12), 'hex'), 'test-key', repeat('a', 64), repeat('b', 64)
		)
	`)
	if err != nil {
		t.Fatalf("insert task5 event: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE datafeed_delivery_state
		SET status = 'LEASED', lease_generation = 1, lease_expires_at = now() + interval '1 minute'
		WHERE event_id = '10000000-0000-4000-8000-000000000019'
	`); err != nil {
		t.Fatalf("lease task5 event: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE datafeed_delivery_state
		SET status = 'PENDING', lease_expires_at = NULL, next_attempt_at = now() + interval '1 second',
		    attempt_count = 1, terminal_outcome_code = 'retryable_failure'
		WHERE event_id = '10000000-0000-4000-8000-000000000019' AND lease_generation = 1
	`); err != nil {
		t.Fatalf("schedule fenced retry: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE datafeed_delivery_state
		SET status = 'QUARANTINED', quarantine_owner_role = 'data_feed_operator',
		    quarantine_sla_due_at = now() + interval '24 hours', terminal_outcome_code = 'conflict'
		WHERE event_id = '10000000-0000-4000-8000-000000000019' AND lease_generation = 1
	`); err == nil || !strings.Contains(err.Error(), "lease") {
		t.Fatalf("stale lease quarantine error = %v, want fenced rejection", err)
	}
}

func TestDataFeedReplayMigrationCreatesImmutableRunMembership(t *testing.T) {
	pool := createTestDatabase(t, "datafeed_task6_replay")
	ctx := context.Background()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if migrations.LatestVersion != 28 {
		t.Fatalf("latest migration = %d, want 28", migrations.LatestVersion)
	}
	for _, table := range []string{"datafeed_replay_runs", "datafeed_replay_run_events", "datafeed_replay_delivery_state"} {
		var relation *string
		if err := pool.QueryRow(ctx, "SELECT to_regclass($1)::text", "public."+table).Scan(&relation); err != nil || relation == nil {
			t.Fatalf("Task 6 table %s = %v, err=%v", table, relation, err)
		}
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO datafeed_events (
			event_id, contract_version, event_type, event_version, source_system,
			tenant_id, owning_organization_id, visibility_purpose_code, operation_id,
			correlation_id, aggregate_type, aggregate_id, aggregate_revision,
			effective_at, known_at, occurred_at, emitted_at, entity_refs, state_after,
			payload_ciphertext, payload_nonce, payload_key_ref, payload_sha256, canonical_event_sha256
		) VALUES (
			'10000000-0000-4000-8000-000000000061', '3.0.0', 'audit.planned', 1, 'aviasurveil-production-api',
			'tenant-task6', 'organization-task6', 'regulated_oversight', 'operation-task6-replay',
			'20000000-0000-4000-8000-000000000061', 'audit', 'audit-task6-replay', 1,
			'2026-07-29T10:00:00Z', '2026-07-29T10:01:00Z', '2026-07-29T10:02:00Z', '2026-07-29T10:03:00Z',
			'{"audit_id":"audit-task6-replay"}', 'audit_planned', decode('00', 'hex'),
			decode(repeat('00', 12), 'hex'), 'test-key', repeat('a', 64), repeat('b', 64)
		)
	`)
	if err != nil {
		t.Fatalf("insert replay source event: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO datafeed_replay_runs (
			run_id, run_kind, approval_id, request_sha256, tenant_id, owning_organization_id,
			source_system, contract_version, requested_at
		) VALUES (
			'10000000-0000-4000-8000-000000000062', 'REPLAY',
			'10000000-0000-4000-8000-000000000063', repeat('c', 64), 'tenant-task6',
			'organization-task6', 'aviasurveil-production-api', '3.0.0', '2026-07-30T10:00:00Z'
		)
	`)
	if err != nil {
		t.Fatalf("insert immutable replay run: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO datafeed_replay_run_events (
			run_id, event_id, canonical_event_sha256, effective_at, known_at, occurred_at
		)
		SELECT '10000000-0000-4000-8000-000000000062', event_id, canonical_event_sha256,
		       effective_at, known_at, occurred_at
		FROM datafeed_events WHERE event_id = '10000000-0000-4000-8000-000000000061'
	`)
	if err != nil {
		t.Fatalf("bind immutable replay event: %v", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO datafeed_replay_delivery_state (run_id, event_id)
		VALUES ('10000000-0000-4000-8000-000000000062', '10000000-0000-4000-8000-000000000061')
	`)
	if err != nil {
		t.Fatalf("create independent replay delivery state: %v", err)
	}
	var replayState string
	if err := pool.QueryRow(ctx, `SELECT status FROM datafeed_replay_delivery_state WHERE run_id = '10000000-0000-4000-8000-000000000062' AND event_id = '10000000-0000-4000-8000-000000000061'`).Scan(&replayState); err != nil || replayState != "PENDING" {
		t.Fatalf("replay delivery state=%q err=%v", replayState, err)
	}
	var occurredAt time.Time
	if err := pool.QueryRow(ctx, `SELECT occurred_at FROM datafeed_replay_run_events WHERE run_id = '10000000-0000-4000-8000-000000000062'`).Scan(&occurredAt); err != nil || !occurredAt.Equal(time.Date(2026, 7, 29, 10, 2, 0, 0, time.UTC)) {
		t.Fatalf("replay event preserved occurred_at=%s err=%v", occurredAt, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE datafeed_replay_run_events SET occurred_at = now() WHERE run_id = '10000000-0000-4000-8000-000000000062'`); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("replay membership mutation error=%v, want immutable rejection", err)
	}
}

func TestPostgresReplayStoreBindsExactScopedEventMembershipAndRejectsChangedReplay(t *testing.T) {
	pool := createTestDatabase(t, "datafeed_task6_store")
	ctx := context.Background()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO datafeed_events (
			event_id, contract_version, event_type, event_version, source_system,
			tenant_id, owning_organization_id, visibility_purpose_code, operation_id,
			correlation_id, aggregate_type, aggregate_id, aggregate_revision,
			effective_at, known_at, occurred_at, emitted_at, entity_refs, state_after,
			payload_ciphertext, payload_nonce, payload_key_ref, payload_sha256, canonical_event_sha256
		) VALUES (
			'10000000-0000-4000-8000-000000000071', '3.0.0', 'audit.planned', 1, 'aviasurveil-production-api',
			'tenant-task6', 'organization-task6', 'regulated_oversight', 'operation-task6-store',
			'20000000-0000-4000-8000-000000000071', 'audit', 'audit-task6-store', 1,
			'2026-07-29T10:00:00Z', '2026-07-29T10:01:00Z', '2026-07-29T10:02:00Z', '2026-07-29T10:03:00Z',
			'{"audit_id":"audit-task6-store"}', 'audit_planned', decode('00', 'hex'),
			decode(repeat('00', 12), 'hex'), 'test-key', repeat('a', 64), repeat('b', 64)
		)
	`)
	if err != nil {
		t.Fatalf("insert replay-store source event: %v", err)
	}
	request := datafeed.ReplayRequest{
		RunID:                 "10000000-0000-4000-8000-000000000072",
		ApprovalID:            "10000000-0000-4000-8000-000000000073",
		TenantID:              "tenant-task6",
		OwningOrganizationID:  "organization-task6",
		SourceSystem:          "aviasurveil-production-api",
		ContractVersion:       "3.0.0",
		EventIDs:              []string{"10000000-0000-4000-8000-000000000071"},
		AllowedTerminalStates: []string{"PENDING"},
		RequestedAt:           time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC),
	}
	store := datafeed.PostgresReplayStore{Pool: pool}
	result, err := store.CreateReplayRun(ctx, request)
	if err != nil || result.RunID != request.RunID || result.EventCount != 1 || result.RequestDigest == "" {
		t.Fatalf("create replay run=%+v err=%v", result, err)
	}
	if replay, err := store.CreateReplayRun(ctx, request); err != nil || replay != result {
		t.Fatalf("exact replay=%+v err=%v want=%+v", replay, err, result)
	}
	changed := request
	changed.AllowedTerminalStates = []string{"QUARANTINED"}
	if _, err := store.CreateReplayRun(ctx, changed); err == nil {
		t.Fatal("changed replay scope reused an immutable run")
	}
	var status string
	var memberships int
	if err := pool.QueryRow(ctx, `SELECT status FROM datafeed_delivery_state WHERE event_id = '10000000-0000-4000-8000-000000000071'`).Scan(&status); err != nil || status != "PENDING" {
		t.Fatalf("source delivery state=%q err=%v, replay creation must not mutate it", status, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM datafeed_replay_run_events WHERE run_id = '10000000-0000-4000-8000-000000000072'`).Scan(&memberships); err != nil || memberships != 1 {
		t.Fatalf("replay memberships=%d err=%v", memberships, err)
	}
	var replayDeliveries int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM datafeed_replay_delivery_state WHERE run_id = '10000000-0000-4000-8000-000000000072' AND status = 'PENDING'`).Scan(&replayDeliveries); err != nil || replayDeliveries != 1 {
		t.Fatalf("replay delivery lanes=%d err=%v", replayDeliveries, err)
	}
}

func TestPostgresReplayStoreCreatesSourceConsistentBackfillLane(t *testing.T) {
	pool := createTestDatabase(t, "datafeed_task6_backfill_store")
	ctx := context.Background()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO datafeed_events (
			event_id, contract_version, event_type, event_version, source_system,
			tenant_id, owning_organization_id, visibility_purpose_code, operation_id,
			correlation_id, aggregate_type, aggregate_id, aggregate_revision,
			effective_at, known_at, occurred_at, emitted_at, entity_refs, state_after,
			payload_ciphertext, payload_nonce, payload_key_ref, payload_sha256, canonical_event_sha256
		) VALUES (
			'10000000-0000-4000-8000-000000000074', '3.0.0', 'audit.planned', 1, 'aviasurveil-production-api',
			'tenant-task6', 'organization-task6', 'regulated_oversight', 'operation-task6-backfill',
			'20000000-0000-4000-8000-000000000074', 'audit', 'audit-task6-backfill', 1,
			'2026-07-29T10:00:00Z', '2026-07-29T10:01:00Z', '2026-07-29T10:02:00Z', '2026-07-29T10:03:00Z',
			'{"audit_id":"audit-task6-backfill"}', 'audit_planned', decode('00', 'hex'),
			decode(repeat('00', 12), 'hex'), 'test-key', repeat('a', 64), repeat('b', 64)
		)
	`)
	if err != nil {
		t.Fatalf("insert backfill source event: %v", err)
	}
	request := datafeed.BackfillRequest{
		RunID: "10000000-0000-4000-8000-000000000075", ApprovalID: "10000000-0000-4000-8000-000000000076",
		TenantID: "tenant-task6", OwningOrganizationID: "organization-task6", SourceSystem: "aviasurveil-production-api", ContractVersion: "3.0.0",
		SourceCutID: "2026-07-29-source-consistent-cut", SourceManifestDigest: strings.Repeat("a", 64),
		CutAt: time.Date(2026, 7, 29, 11, 0, 0, 0, time.UTC), RequestedAt: time.Date(2026, 7, 30, 11, 0, 0, 0, time.UTC),
		EventIDs: []string{"10000000-0000-4000-8000-000000000074"},
	}
	store := datafeed.PostgresReplayStore{Pool: pool}
	result, err := store.CreateBackfillRun(ctx, request)
	if err != nil || result.RunID != request.RunID || result.EventCount != 1 {
		t.Fatalf("create source-consistent backfill=%+v err=%v", result, err)
	}
	var kind, sourceCut, manifestDigest string
	if err := pool.QueryRow(ctx, `SELECT run_kind, source_cut_id, source_manifest_sha256 FROM datafeed_replay_runs WHERE run_id = '10000000-0000-4000-8000-000000000075'`).Scan(&kind, &sourceCut, &manifestDigest); err != nil || kind != "BACKFILL" || sourceCut != request.SourceCutID || manifestDigest != request.SourceManifestDigest {
		t.Fatalf("backfill run identity=%q/%q/%q err=%v", kind, sourceCut, manifestDigest, err)
	}
	var deliveries int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM datafeed_replay_delivery_state WHERE run_id = '10000000-0000-4000-8000-000000000075' AND status = 'PENDING'`).Scan(&deliveries); err != nil || deliveries != 1 {
		t.Fatalf("backfill delivery lanes=%d err=%v", deliveries, err)
	}
}

func TestPostgresReplayLeaseSourceClaimsOnlyTheIndependentReplayLane(t *testing.T) {
	pool := createTestDatabase(t, "datafeed_task6_replay_lease")
	ctx := context.Background()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	key := []byte("0123456789abcdef0123456789abcdef")
	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	writer, err := datafeed.NewWriter(datafeed.WriterConfig{TenantID: "tenant-task6", PayloadKey: key, PayloadKeyRef: "task6-key"})
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	err = database.WithinTransaction(ctx, pool, func(ctx context.Context, transaction pgx.Tx) error {
		_, appendErr := writer.Append(ctx, transaction, "task6-replay-lease-operation", datafeed.EventInput{
			EventID: "10000000-0000-4000-8000-000000000081", EventType: "audit.planned", OwningOrganizationID: "organization-task6", ActorOrganizationID: "caa-task6", CorrelationID: "20000000-0000-4000-8000-000000000081", AggregateType: "audit", AggregateID: "audit-task6-replay-lease", AggregateRevision: 1, EffectiveAt: now, KnownAt: now, OccurredAt: now, EmittedAt: now, VisibilityPurposeCode: "regulated_oversight", EntityRefs: map[string]any{"audit_id": "audit-task6-replay-lease"}, StateAfter: "audit_planned", Payload: map[string]any{"audit_program_ref": "program-task6", "audit_scope_code": "airport_operations", "planned_start_at": "2026-07-30T12:00:00Z"},
		})
		return appendErr
	})
	if err != nil {
		t.Fatalf("append replay source event: %v", err)
	}
	request := datafeed.ReplayRequest{
		RunID:                 "10000000-0000-4000-8000-000000000082",
		ApprovalID:            "10000000-0000-4000-8000-000000000083",
		TenantID:              "tenant-task6",
		OwningOrganizationID:  "organization-task6",
		SourceSystem:          "aviasurveil-production-api",
		ContractVersion:       "3.0.0",
		EventIDs:              []string{"10000000-0000-4000-8000-000000000081"},
		AllowedTerminalStates: []string{"PENDING"},
		RequestedAt:           now,
	}
	if _, err := (datafeed.PostgresReplayStore{Pool: pool}).CreateReplayRun(ctx, request); err != nil {
		t.Fatalf("create replay run: %v", err)
	}
	source := datafeed.PostgresReplayLeaseSource{Pool: pool, RunID: request.RunID, TenantID: request.TenantID, OwningOrganizationID: request.OwningOrganizationID, PayloadKey: key, Clock: func() time.Time { return now }}
	items, err := source.Claim(ctx, time.Minute, 10)
	if err != nil || len(items) != 1 || items[0].ReplayRunID != request.RunID || items[0].LeaseGeneration != 1 {
		t.Fatalf("claim replay lane=%+v err=%v", items, err)
	}
	var sourceStatus, replayStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM datafeed_delivery_state WHERE event_id = '10000000-0000-4000-8000-000000000081'`).Scan(&sourceStatus); err != nil || sourceStatus != "PENDING" {
		t.Fatalf("source lane status=%q err=%v", sourceStatus, err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM datafeed_replay_delivery_state WHERE run_id = '10000000-0000-4000-8000-000000000082' AND event_id = '10000000-0000-4000-8000-000000000081'`).Scan(&replayStatus); err != nil || replayStatus != "LEASED" {
		t.Fatalf("replay lane status=%q err=%v", replayStatus, err)
	}
	recorder := datafeed.PostgresDecisionRecorder{Pool: pool, NewID: func() (string, error) { return "30000000-0000-4000-8000-000000000081", nil }}
	err = recorder.Record(ctx, []datafeed.DeliveryDecision{{
		EventID: items[0].EventID, LeaseGeneration: items[0].LeaseGeneration, ReplayRunID: request.RunID,
		RecordedAt: now, Action: datafeed.DeliveryAcknowledge, OutcomeCode: "accepted", ReceiptDigest: strings.Repeat("c", 64),
	}})
	if err != nil {
		t.Fatalf("record replay acknowledgement: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM datafeed_delivery_state WHERE event_id = '10000000-0000-4000-8000-000000000081'`).Scan(&sourceStatus); err != nil || sourceStatus != "PENDING" {
		t.Fatalf("source lane changed after replay receipt status=%q err=%v", sourceStatus, err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM datafeed_replay_delivery_state WHERE run_id = '10000000-0000-4000-8000-000000000082' AND event_id = '10000000-0000-4000-8000-000000000081'`).Scan(&replayStatus); err != nil || replayStatus != "ACKNOWLEDGED" {
		t.Fatalf("replay acknowledgement state=%q err=%v", replayStatus, err)
	}
	var replayAttempts int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM datafeed_replay_delivery_attempts WHERE run_id = '10000000-0000-4000-8000-000000000082' AND event_id = '10000000-0000-4000-8000-000000000081' AND outcome_code = 'accepted'`).Scan(&replayAttempts); err != nil || replayAttempts != 1 {
		t.Fatalf("replay attempt count=%d err=%v", replayAttempts, err)
	}
	manifest, err := (datafeed.PostgresReconciliationManifestExporter{Pool: pool}).ExportReplayManifest(ctx, request.RunID)
	if err != nil || manifest.RunID != request.RunID || manifest.ContractVersion != "3.0.0" || len(manifest.Events) != 1 || manifest.Events[0].DeliveryOutcome != "ACKNOWLEDGED" || manifest.Events[0].AcknowledgementReceiptDigest != strings.Repeat("c", 64) {
		t.Fatalf("export producer replay reconciliation manifest=%+v err=%v", manifest, err)
	}
}

func TestDataFeedPublisherConcreteLeaseAndReceiptPersistence(t *testing.T) {
	pool := createTestDatabase(t, "datafeed_task5_concrete")
	ctx := context.Background()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	key := []byte("0123456789abcdef0123456789abcdef")
	writer, err := datafeed.NewWriter(datafeed.WriterConfig{TenantID: "tenant-task5", PayloadKey: key, PayloadKeyRef: "task5-key"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	err = database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := writer.Append(ctx, tx, "task5-concrete-operation", datafeed.EventInput{
			EventID: "10000000-0000-4000-8000-000000000055", EventType: "audit.planned", OwningOrganizationID: "organization-task5", ActorOrganizationID: "caa-task5", CorrelationID: "20000000-0000-4000-8000-000000000055", AggregateType: "audit", AggregateID: "audit-task5-concrete", AggregateRevision: 1, EffectiveAt: now, KnownAt: now, OccurredAt: now, EmittedAt: now, VisibilityPurposeCode: "regulated_oversight", EntityRefs: map[string]any{"audit_id": "audit-task5-concrete"}, StateAfter: "audit_planned", Payload: map[string]any{"audit_program_ref": "program-task5", "audit_scope_code": "airport_operations", "planned_start_at": "2026-07-30T10:00:00Z"},
		})
		return err
	})
	if err != nil {
		t.Fatalf("append concrete event: %v", err)
	}
	source := datafeed.PostgresLeaseSource{Pool: pool, TenantID: "tenant-task5", OwningOrganizationID: "organization-task5", PayloadKey: key, Clock: func() time.Time { return now }}
	items, err := source.Claim(ctx, time.Minute, 10)
	if err != nil || len(items) != 1 || items[0].LeaseGeneration != 1 || items[0].EventContentDigest == "" {
		t.Fatalf("claim concrete event = %+v, err = %v", items, err)
	}
	recorder := datafeed.PostgresDecisionRecorder{Pool: pool, NewID: func() (string, error) { return "30000000-0000-4000-8000-000000000055", nil }}
	err = recorder.Record(ctx, []datafeed.DeliveryDecision{{EventID: items[0].EventID, LeaseGeneration: items[0].LeaseGeneration, RecordedAt: now, Action: datafeed.DeliveryAcknowledge, OutcomeCode: "accepted", ReceiptDigest: strings.Repeat("c", 64)}})
	if err != nil {
		t.Fatalf("record concrete acknowledgement: %v", err)
	}
	var status string
	var attempts int
	if err := pool.QueryRow(ctx, `SELECT status FROM datafeed_delivery_state WHERE event_id = '10000000-0000-4000-8000-000000000055'`).Scan(&status); err != nil || status != "ACKNOWLEDGED" {
		t.Fatalf("delivery status = %q, err = %v", status, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM datafeed_delivery_attempts WHERE event_id = '10000000-0000-4000-8000-000000000055' AND outcome_code = 'accepted'`).Scan(&attempts); err != nil || attempts != 1 {
		t.Fatalf("attempts = %d, err = %v", attempts, err)
	}
}

func TestDataFeedWriterIsAtomicEncryptedAndOperationIdempotent(t *testing.T) {
	pool := createTestDatabase(t, "datafeed_atomic")
	ctx := context.Background()
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	key := []byte("0123456789abcdef0123456789abcdef")
	writer, err := datafeed.NewWriter(datafeed.WriterConfig{TenantID: "tenant-a", PayloadKey: key, PayloadKeyRef: "task4-test-key"})
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	now := time.Date(2026, time.August, 1, 8, 0, 1, 0, time.UTC)
	input := datafeed.EventInput{
		EventID:               "10000000-0000-4000-8000-000000000002",
		EventType:             "audit.planned",
		OwningOrganizationID:  "organization-a",
		ActorOrganizationID:   "caa-a",
		CorrelationID:         "20000000-0000-4000-8000-000000000002",
		AggregateType:         "audit",
		AggregateID:           "audit-b",
		AggregateRevision:     1,
		EffectiveAt:           now.Add(-time.Second),
		KnownAt:               now,
		OccurredAt:            now.Add(-time.Second),
		EmittedAt:             now,
		VisibilityPurposeCode: "regulated_oversight",
		EntityRefs:            map[string]any{"audit_id": "audit-b"},
		StateAfter:            "audit_planned",
		Payload: map[string]any{
			"audit_program_ref": "program-b",
			"audit_scope_code":  "airport_operations",
			"planned_start_at":  "2026-08-01T08:00:00Z",
		},
	}
	rollback := errors.New("deliberate rollback")
	err = database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO organizations (id, legal_name, organization_type, status, revision, created_at, updated_at) VALUES ('task4-rollback-org', 'Task 4 Rollback', 'OPERATOR', 'ACTIVE', 1, now(), now())`); err != nil {
			return err
		}
		if _, err := writer.Append(ctx, tx, "operation-rollback", input); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("rollback error = %v, want %v", err, rollback)
	}
	var rolledBackEvents, rolledBackOrganizations int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM datafeed_events WHERE operation_id = 'operation-rollback'`).Scan(&rolledBackEvents); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM organizations WHERE id = 'task4-rollback-org'`).Scan(&rolledBackOrganizations); err != nil {
		t.Fatal(err)
	}
	if rolledBackEvents != 0 || rolledBackOrganizations != 0 {
		t.Fatalf("rollback left events=%d organizations=%d", rolledBackEvents, rolledBackOrganizations)
	}
	if err := database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := writer.Append(ctx, tx, "operation-accepted", input)
		return err
	}); err != nil {
		t.Fatalf("append accepted event: %v", err)
	}
	var ciphertext, nonce []byte
	if err := pool.QueryRow(ctx, `SELECT payload_ciphertext, payload_nonce FROM datafeed_events WHERE operation_id = 'operation-accepted'`).Scan(&ciphertext, &nonce); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ciphertext), "airport_operations") {
		t.Fatal("payload plaintext was retained in the outbox")
	}
	plain, err := datafeed.DecryptPayload(key, datafeed.EncryptedPayload{Ciphertext: ciphertext, Nonce: nonce})
	if err != nil || !strings.Contains(string(plain), "airport_operations") {
		t.Fatalf("decrypt encrypted payload: %v, %q", err, plain)
	}
	if err := database.WithinTransaction(ctx, pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := writer.Append(ctx, tx, "operation-accepted", input)
		return err
	}); err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("same operation event error = %v, want unique rejection", err)
	}
}

func TestDataFeedMigrationUpgradesVersionTwentyOneToCurrentWithoutRewritingHistory(t *testing.T) {
	pool := createTestDatabase(t, "datafeed_upgrade")
	ctx := context.Background()
	applyMigrationFilesThrough(t, pool, 21)
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_events (
			event_id, occurred_at, action, entity_type, entity_id, details
		) VALUES ('task4-history-audit', now(), 'history.retained', 'history', 'record-1', '{"retained":true}')
	`); err != nil {
		t.Fatalf("seed immutable predecessor history: %v", err)
	}
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("upgrade v21 to current: %v", err)
	}
	if version, err := migrations.CurrentVersion(ctx, pool); err != nil || version != migrations.LatestVersion {
		t.Fatalf("upgraded version = %d, err=%v, want %d", version, err, migrations.LatestVersion)
	}
	var details string
	if err := pool.QueryRow(ctx, `SELECT details::text FROM audit_events WHERE event_id = 'task4-history-audit'`).Scan(&details); err != nil || details != `{"retained": true}` {
		t.Fatalf("predecessor history after current migration = %q, err=%v", details, err)
	}
	var events int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM datafeed_events`).Scan(&events); err != nil || events != 0 {
		t.Fatalf("migration backfilled producer events=%d err=%v; source-consistent event-api backfill remains a later controlled run", events, err)
	}
}

func TestDataFeedWorkspaceTransitionReconstructsOnlySourceConsistentAuditLifecycle(t *testing.T) {
	pool := canonicalDatabase(t, "datafeed_workspace_transition")
	ctx := context.Background()
	key := []byte("0123456789abcdef0123456789abcdef")
	writer, err := datafeed.NewWriter(datafeed.WriterConfig{
		TenantID: "tenant-task4", PayloadKey: key, PayloadKeyRef: "task4-workspace-key",
	})
	if err != nil {
		t.Fatalf("new writer: %v", err)
	}
	service := application.NewService(pool, application.Dependencies{
		Clock:          func() time.Time { return canonicalNow },
		DataFeedWriter: writer,
		IDGenerator: func(prefix string) string {
			return "task4-workspace-" + prefix
		},
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO surveillance_plan_items (
			id, title, plan_year, organization_id, inspection_type, scheduled_date,
			estimated_budget, status, current_owner_role, next_action, revision
		) VALUES (
			'task4-plan-ramp', 'Task 4 Ramp plan', 2026, 'airline-xyz', 'RAMP', '2026-08-01',
			0, 'RELEASED', 'manager', 'Create audit workspace', 1
		);
		INSERT INTO checklist_template_versions (id, template_id, version, title, snapshot, published_at)
		VALUES (
			'task4-template-v1', 'task4-template', 1, 'Task 4 template',
			'{"schemaVersion":1,"protocolVersion":1,"questions":[{"id":"task4-question-1","prompt":"Verify ramp control.","expectedEvidence":"Ramp record","regulatoryReference":"REF-TASK4"}]}'::jsonb,
			'2026-07-01T00:00:00Z'
		)
	`); err != nil {
		t.Fatalf("seed source-consistent workspace prerequisites: %v", err)
	}
	command := application.CreateAuditWorkspaceCommand{
		OperationID: "task4-workspace-operation", IdempotencyKey: "task4-workspace-idempotency",
		PlanningItemID: "task4-plan-ramp", ExpectedPlanningRevision: 1,
		AuditID: "task4-audit-ramp", AssignmentID: "task4-assignment-ramp",
		PackageID: "task4-package-ramp", PackageDraftID: "task4-package-draft-ramp",
		TemplateID: "task4-template", TemplateVersionID: "task4-template-v1",
		LeadInspectorSubjectID: "lead-001", MemberSubjectIDs: []string{"lead-001", "inspector-cabin-001"},
		Questions: []application.AuditWorkspaceQuestion{{
			QuestionID: "task4-question-1", AssignedInspectorSubjectIDs: []string{"inspector-cabin-001"},
		}},
		ScheduledStartDate: "2026-08-01", ScheduledEndDate: "2026-08-02",
		ExpiresAt: canonicalNow.Add(72 * time.Hour),
	}
	actor := identity.Principal{SubjectID: "manager-001", OrganizationID: "caa", Roles: []identity.Role{identity.RoleDepartmentManager}}
	withoutWriter := application.NewService(pool, application.Dependencies{
		Clock: func() time.Time { return canonicalNow },
		IDGenerator: func(prefix string) string {
			return "task4-unconfigured-" + prefix
		},
	})
	unconfigured := command
	unconfigured.OperationID, unconfigured.IdempotencyKey = "task4-unconfigured-operation", "task4-unconfigured-idempotency"
	unconfigured.AuditID, unconfigured.AssignmentID = "task4-audit-unconfigured", "task4-assignment-unconfigured"
	unconfigured.PackageID, unconfigured.PackageDraftID = "task4-package-unconfigured", "task4-package-draft-unconfigured"
	if _, err := withoutWriter.CreateAuditWorkspace(ctx, actor, unconfigured); !errors.Is(err, application.ErrDataFeedNotConfigured) {
		t.Fatalf("unconfigured writer error = %v, want closed datafeed configuration rejection", err)
	}
	var unconfiguredAudits, unconfiguredEvents int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inspections WHERE id = $1`, unconfigured.AuditID).Scan(&unconfiguredAudits); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM datafeed_events WHERE operation_id = $1`, unconfigured.OperationID).Scan(&unconfiguredEvents); err != nil {
		t.Fatal(err)
	}
	if unconfiguredAudits != 0 || unconfiguredEvents != 0 {
		t.Fatalf("unconfigured writer left audit=%d events=%d, want closed rollback", unconfiguredAudits, unconfiguredEvents)
	}
	if _, err := service.CreateAuditWorkspace(ctx, actor, command); err != nil {
		t.Fatalf("create source-consistent workspace: %v", err)
	}
	type storedEvent struct {
		EventID, EventType, StateAfter, CorrelationID string
		StateBefore, CausationID                      *string
		EffectiveAt, OccurredAt                       time.Time
		Revision                                      int64
		Ciphertext, Nonce                             []byte
	}
	rows, err := pool.Query(ctx, `
		SELECT event_id, event_type, state_before, state_after, correlation_id, causation_id,
		       effective_at, occurred_at, aggregate_revision, payload_ciphertext, payload_nonce
		FROM datafeed_events WHERE operation_id = $1 ORDER BY event_type
	`, command.OperationID)
	if err != nil {
		t.Fatalf("read emitted workspace events: %v", err)
	}
	defer rows.Close()
	events := []storedEvent{}
	for rows.Next() {
		var event storedEvent
		if err := rows.Scan(&event.EventID, &event.EventType, &event.StateBefore, &event.StateAfter,
			&event.CorrelationID, &event.CausationID, &event.EffectiveAt, &event.OccurredAt,
			&event.Revision, &event.Ciphertext, &event.Nonce); err != nil {
			t.Fatalf("scan emitted workspace event: %v", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate emitted workspace events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("emitted event count = %d, want the planned and started pair", len(events))
	}
	planned, started := events[0], events[1]
	if planned.EventType != "audit.planned" || planned.StateBefore != nil || planned.StateAfter != "audit_planned" || planned.Revision != 1 {
		t.Fatalf("planned event lifecycle = %#v", planned)
	}
	if started.EventType != "audit.started" || started.StateBefore == nil || *started.StateBefore != "audit_planned" || started.StateAfter != "audit_in_progress" || started.Revision != 1 {
		t.Fatalf("started event lifecycle = %#v", started)
	}
	if planned.CorrelationID == "" || planned.CorrelationID != started.CorrelationID || started.CausationID == nil || *started.CausationID != planned.EventID {
		t.Fatalf("planned/started correlation and causation linkage is not exact: planned=%#v started=%#v", planned, started)
	}
	if !planned.EffectiveAt.Equal(canonicalNow) || !started.OccurredAt.Equal(canonicalNow) {
		t.Fatalf("effective/occurred source times = planned=%s started=%s", planned.EffectiveAt, started.OccurredAt)
	}
	for _, assertion := range []struct {
		event storedEvent
		want  map[string]any
	}{
		{planned, map[string]any{"audit_program_ref": "task4-plan-ramp", "audit_scope_code": "ramp", "planned_start_at": "2026-08-01T00:00:00Z"}},
		{started, map[string]any{"started_at": canonicalNow.Format(time.RFC3339Nano)}},
	} {
		plain, err := datafeed.DecryptPayload(key, datafeed.EncryptedPayload{Ciphertext: assertion.event.Ciphertext, Nonce: assertion.event.Nonce})
		if err != nil {
			t.Fatalf("decrypt %s payload: %v", assertion.event.EventType, err)
		}
		var payload map[string]any
		if err := json.Unmarshal(plain, &payload); err != nil {
			t.Fatalf("decode %s payload: %v", assertion.event.EventType, err)
		}
		if !mapsEqual(payload, assertion.want) {
			t.Fatalf("%s payload = %#v, want %#v", assertion.event.EventType, payload, assertion.want)
		}
	}
	if _, err := service.CreateAuditWorkspace(ctx, actor, command); err != nil {
		t.Fatalf("replay accepted workspace operation: %v", err)
	}
	var eventCount, auditCount, outboxCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM datafeed_events WHERE operation_id = $1`, command.OperationID).Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE entity_id = $1`, command.AuditID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM outbox_messages WHERE aggregate_id = $1`, command.AuditID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if eventCount != 2 || auditCount != 1 || outboxCount != 1 {
		t.Fatalf("idempotent source transition rows = events:%d audit:%d outbox:%d", eventCount, auditCount, outboxCount)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO surveillance_plan_items (
			id, title, plan_year, organization_id, inspection_type, scheduled_date,
			estimated_budget, status, current_owner_role, next_action, revision
		) VALUES ('task4-plan-unknown', 'Task 4 unknown plan', 2026, 'airline-xyz', 'RAMP/AIS', '2026-08-01', 0, 'RELEASED', 'manager', 'Create audit workspace', 1)
	`); err != nil {
		t.Fatalf("seed unsupported source scope: %v", err)
	}
	unsupported := command
	unsupported.OperationID, unsupported.IdempotencyKey = "task4-unknown-operation", "task4-unknown-idempotency"
	unsupported.PlanningItemID, unsupported.AuditID, unsupported.AssignmentID = "task4-plan-unknown", "task4-audit-unknown", "task4-assignment-unknown"
	unsupported.PackageID, unsupported.PackageDraftID = "task4-package-unknown", "task4-package-draft-unknown"
	if _, err := withoutWriter.CreateAuditWorkspace(ctx, actor, unsupported); !errors.Is(err, application.ErrInvalid) {
		t.Fatalf("unsupported inspection type error = %v, want closed invalid rejection", err)
	}
	var unsupportedAudits, unsupportedEvents int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inspections WHERE id = 'task4-audit-unknown'`).Scan(&unsupportedAudits); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM datafeed_events WHERE operation_id = 'task4-unknown-operation'`).Scan(&unsupportedEvents); err != nil {
		t.Fatal(err)
	}
	if unsupportedAudits != 0 || unsupportedEvents != 0 {
		t.Fatalf("unsupported scope left audit=%d events=%d, want atomic rollback", unsupportedAudits, unsupportedEvents)
	}
}

func mapsEqual(actual, want map[string]any) bool {
	actualBytes, actualErr := json.Marshal(actual)
	wantBytes, wantErr := json.Marshal(want)
	return actualErr == nil && wantErr == nil && string(actualBytes) == string(wantBytes)
}
