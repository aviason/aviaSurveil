package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/potentialfindings"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

const fullPlatformSchemaVersion int64 = migrations.LatestVersion

var fullPlatformTables = []string{
	"planning_intake_drafts",
	"audit_assignments",
	"audit_team_members",
	"audit_question_assignments",
	"communication_threads",
	"communication_messages",
	"communication_attachments",
	"document_records",
	"document_versions",
	"document_render_jobs",
	"notification_records",
	"notification_delivery_jobs",
	"reminder_dispatches",
	"risk_projection_versions",
	"regulatory_reference_versions",
	"question_versions",
	"template_masters",
	"template_draft_versions",
	"template_version_questions",
	"report_definition_versions",
	"user_lifecycle_requests",
	"user_profiles",
	"user_settings",
	"command_transaction_links",
}

var newPersistentModuleTables = map[string]map[string]bool{
	"assignments": {
		"audit_assignments": true, "audit_team_members": true, "audit_question_assignments": true,
	},
	"communications": {
		"communication_threads": true, "communication_messages": true, "communication_attachments": true,
	},
	"documents": {
		"document_records": true, "document_versions": true, "document_render_jobs": true,
	},
	"notifications": {
		"notification_records": true, "notification_delivery_jobs": true, "reminder_dispatches": true,
	},
	"risk": {
		"risk_projection_versions": true,
	},
	"administration": {
		"regulatory_reference_versions": true,
		"question_versions":             true,
		"template_masters":              true,
		"template_draft_versions":       true,
		"template_version_questions":    true,
		"report_definition_versions":    true,
		"user_lifecycle_requests":       true,
	},
}

func TestFullPlatformMigrationAndStoreInventory(t *testing.T) {
	moduleRoot := apiModuleRoot(t)
	for _, relativePath := range []string{
		"migrations/000007_full_workflow_projection.up.sql",
		"migrations/000008_communications_documents.up.sql",
		"migrations/000009_notifications_risk_admin.up.sql",
		"migrations/000010_identity_settings.up.sql",
		"migrations/000011_preliminary_report_versions.up.sql",
	} {
		if _, err := os.Stat(filepath.Join(moduleRoot, relativePath)); err != nil {
			t.Errorf("required full-platform migration %s: %v", relativePath, err)
		}
	}
	for module := range newPersistentModuleTables {
		for _, file := range []string{"queries.sql", "db.go", "models.go", "querier.go", "queries.sql.go"} {
			relativePath := filepath.Join("internal", module, "store", "postgres", file)
			if _, err := os.Stat(filepath.Join(moduleRoot, relativePath)); err != nil {
				t.Errorf("required module-owned SQLC file %s: %v", relativePath, err)
			}
		}
	}
}

func TestFullPlatformFreshMigration(t *testing.T) {
	pool := createTestDatabase(t, "full_platform_fresh")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply full-platform migrations: %v", err)
	}
	version, err := migrations.CurrentVersion(context.Background(), pool)
	if err != nil {
		t.Fatalf("read migration version: %v", err)
	}
	if version != fullPlatformSchemaVersion {
		t.Fatalf("migration version = %d, want %d", version, fullPlatformSchemaVersion)
	}
	for _, table := range fullPlatformTables {
		var relation *string
		if err := pool.QueryRow(context.Background(), "SELECT to_regclass($1)::text", "public."+table).Scan(&relation); err != nil {
			t.Fatalf("look up full-platform table %s: %v", table, err)
		}
		if relation == nil {
			t.Errorf("required full-platform table %s does not exist", table)
		}
	}
}

func TestFullPlatformNMinusOneUpgradeFromVersionSix(t *testing.T) {
	pool := createTestDatabase(t, "full_platform_n_minus_one")
	applyMigrationFilesThrough(t, pool, 6)
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("upgrade v6 schema to full platform: %v", err)
	}
	version, err := migrations.CurrentVersion(context.Background(), pool)
	if err != nil || version != fullPlatformSchemaVersion {
		t.Fatalf("upgraded migration version = %d, err = %v, want %d", version, err, fullPlatformSchemaVersion)
	}
}

func TestFullPlatformVersionRowsAreAppendOnly(t *testing.T) {
	pool := createTestDatabase(t, "full_platform_immutable")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	seedFullPlatformReferences(t, pool)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO communication_threads (
			id, organization_id, visibility, subject, revision, created_at
		) VALUES ('thread-immutable', 'operator-immutable', 'AUDITEE_VISIBLE', 'Coordination', 1, now());
		INSERT INTO communication_messages (
			id, thread_id, organization_id, visibility, sender_subject_id, audience, direction,
			subject, body, idempotency_key, revision, created_at
		) VALUES (
			'message-immutable', 'thread-immutable', 'operator-immutable', 'AUDITEE_VISIBLE',
			'subject-immutable', 'AUDITEE', 'CAA_TO_AUDITEE', 'Coordination', 'Body',
			'message-operation-immutable', 1, now()
		);
		INSERT INTO document_records (
			id, organization_id, kind, title, revision, created_at
		) VALUES ('document-immutable', 'operator-immutable', 'REPORT', 'Final Report', 1, now());
		INSERT INTO document_versions (
			id, document_id, organization_id, version, visibility, status, file_name,
			media_type, sha256, size_bytes, created_by_subject_id, created_at
		) VALUES (
			'document-version-immutable', 'document-immutable', 'operator-immutable', 1,
			'AUDITEE_VISIBLE', 'LOCKED', 'report.pdf', 'application/pdf', 'sha256:immutable',
			100, 'subject-immutable', now()
		);
		INSERT INTO risk_projection_versions (
			id, projection_kind, organization_id, version, source, snapshot,
			advisory_only, calculated_at
		) VALUES (
			'risk-version-immutable', 'OVERSIGHT_HEALTH', 'operator-immutable', 1,
			'configured-local', '{"score":72}', true, now()
		);
		INSERT INTO regulatory_reference_versions (
			id, reference_id, version, title, status, effective_date, snapshot, created_at
		) VALUES (
			'reference-version-immutable', 'reference-immutable', 1, 'Configured rule',
			'ACTIVE', CURRENT_DATE, '{"rules":["configured"]}', now()
		)
	`); err != nil {
		t.Fatalf("seed full-platform immutable versions: %v", err)
	}

	for name, statement := range map[string]string{
		"communication message update": "UPDATE communication_messages SET body = 'changed' WHERE id = 'message-immutable'",
		"communication message delete": "DELETE FROM communication_messages WHERE id = 'message-immutable'",
		"document version update":      "UPDATE document_versions SET file_name = 'changed.pdf' WHERE id = 'document-version-immutable'",
		"document version delete":      "DELETE FROM document_versions WHERE id = 'document-version-immutable'",
		"risk projection update":       "UPDATE risk_projection_versions SET snapshot = '{\"score\":0}' WHERE id = 'risk-version-immutable'",
		"regulatory version update":    "UPDATE regulatory_reference_versions SET title = 'changed' WHERE id = 'reference-version-immutable'",
	} {
		if _, err := pool.Exec(context.Background(), statement); err == nil {
			t.Errorf("%s succeeded; version rows must be append-only", name)
		}
	}
}

func TestFullPlatformScopedUniquenessIndexesAndQueryPlans(t *testing.T) {
	pool := createTestDatabase(t, "full_platform_indexes")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	for _, index := range []string{
		"audit_assignments_org_status_idx",
		"communication_messages_org_visibility_created_idx",
		"document_versions_org_visibility_created_idx",
		"notification_records_recipient_unread_idx",
		"notification_records_recipient_created_idx",
		"risk_projection_kind_org_calculated_idx",
		"user_lifecycle_requests_status_created_idx",
	} {
		var relation *string
		if err := pool.QueryRow(context.Background(), "SELECT to_regclass($1)::text", "public."+index).Scan(&relation); err != nil {
			t.Fatalf("look up index %s: %v", index, err)
		}
		if relation == nil {
			t.Errorf("required organization-scope index %s does not exist", index)
		}
	}

	for name, assertion := range map[string]struct {
		query string
		index string
	}{
		"assignments": {
			query: "SELECT id FROM audit_assignments WHERE organization_id = 'operator-plan' AND status = 'PLANNED' AND tombstoned_at IS NULL ORDER BY scheduled_start_date, id",
			index: "audit_assignments_org_status_idx",
		},
		"communications": {
			query: "SELECT id FROM communication_messages WHERE organization_id = 'operator-plan' AND visibility = 'AUDITEE_VISIBLE' ORDER BY created_at DESC, id DESC",
			index: "communication_messages_org_visibility_created_idx",
		},
		"documents": {
			query: "SELECT id FROM document_versions WHERE organization_id = 'operator-plan' AND visibility = 'AUDITEE_VISIBLE' ORDER BY created_at DESC, id DESC",
			index: "document_versions_org_visibility_created_idx",
		},
		"notifications": {
			query: "SELECT id FROM notification_records WHERE recipient_subject_id = 'subject-plan' AND tombstoned_at IS NULL ORDER BY created_at DESC, id DESC",
			index: "notification_records_recipient_created_idx",
		},
		"risk": {
			query: "SELECT id FROM risk_projection_versions WHERE projection_kind = 'OVERSIGHT_HEALTH' AND organization_id IS NOT DISTINCT FROM 'operator-plan'::text ORDER BY calculated_at DESC, id DESC",
			index: "risk_projection_kind_org_calculated_idx",
		},
		"administration": {
			query: "SELECT id FROM user_lifecycle_requests WHERE status = 'PENDING' ORDER BY created_at, id",
			index: "user_lifecycle_requests_status_created_idx",
		},
	} {
		if _, err := pool.Exec(context.Background(), "SET enable_seqscan = off"); err != nil {
			t.Fatalf("disable sequential scan: %v", err)
		}
		rows, err := pool.Query(context.Background(), "EXPLAIN "+assertion.query)
		if err != nil {
			t.Fatalf("explain %s query: %v", name, err)
		}
		var plan strings.Builder
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				rows.Close()
				t.Fatalf("scan %s plan: %v", name, err)
			}
			plan.WriteString(line)
			plan.WriteByte('\n')
		}
		rows.Close()
		if !strings.Contains(plan.String(), assertion.index) {
			t.Errorf(
				"%s list query did not use %s:\n%s",
				name,
				assertion.index,
				plan.String(),
			)
		}
	}

	for name, assertion := range map[string]struct {
		query string
		index string
	}{
		"assignment detail": {
			query: "SELECT id FROM audit_assignments WHERE id = 'assignment-plan'",
			index: "audit_assignments_pkey",
		},
		"communication detail": {
			query: "SELECT id FROM communication_messages WHERE id = 'message-plan'",
			index: "communication_messages_pkey",
		},
		"document detail": {
			query: "SELECT id FROM document_versions WHERE id = 'document-plan'",
			index: "document_versions_pkey",
		},
		"notification detail": {
			query: "SELECT id FROM notification_records WHERE id = 'notification-plan'",
			index: "notification_records_pkey",
		},
		"risk detail": {
			query: "SELECT id FROM risk_projection_versions WHERE id = 'risk-plan'",
			index: "risk_projection_versions_pkey",
		},
		"administration detail": {
			query: "SELECT id FROM regulatory_reference_versions WHERE id = 'reference-plan'",
			index: "regulatory_reference_versions_pkey",
		},
	} {
		rows, err := pool.Query(context.Background(), "EXPLAIN "+assertion.query)
		if err != nil {
			t.Fatalf("explain %s query: %v", name, err)
		}
		var plan strings.Builder
		for rows.Next() {
			var line string
			if err := rows.Scan(&line); err != nil {
				rows.Close()
				t.Fatalf("scan %s plan: %v", name, err)
			}
			plan.WriteString(line)
			plan.WriteByte('\n')
		}
		rows.Close()
		if !strings.Contains(plan.String(), assertion.index) {
			t.Errorf(
				"%s query did not use %s:\n%s",
				name,
				assertion.index,
				plan.String(),
			)
		}
	}
}

func TestFullPlatformScopedDeduplicationConstraints(t *testing.T) {
	pool := createTestDatabase(t, "full_platform_deduplication")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	seedFullPlatformReferences(t, pool)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO communication_threads (
			id, organization_id, visibility, subject
		) VALUES (
			'thread-deduplication', 'operator-immutable', 'AUDITEE_VISIBLE', 'Subject'
		);
		INSERT INTO communication_messages (
			id, thread_id, organization_id, visibility, sender_subject_id,
			audience, direction, subject, body, idempotency_key
		) VALUES (
			'message-deduplication-1', 'thread-deduplication', 'operator-immutable',
			'AUDITEE_VISIBLE', 'subject-immutable', 'AUDITEE', 'CAA_TO_AUDITEE',
			'Subject', 'Body', 'message-deduplication-key'
		);
		INSERT INTO document_records (
			id, organization_id, kind, title
		) VALUES (
			'document-deduplication', 'operator-immutable', 'REPORT', 'Report'
		);
		INSERT INTO document_versions (
			id, document_id, organization_id, version, visibility, status,
			file_name, media_type, sha256, size_bytes, created_by_subject_id
		) VALUES (
			'document-version-deduplication-1', 'document-deduplication',
			'operator-immutable', 1, 'AUDITEE_VISIBLE', 'LOCKED', 'report.pdf',
			'application/pdf', 'sha256:deduplication', 10, 'subject-immutable'
		);
		INSERT INTO notification_records (
			id, recipient_subject_id, organization_id, title, body, deduplication_key
		) VALUES (
			'notification-deduplication-1', 'subject-immutable', 'operator-immutable',
			'Title', 'Body', 'notification-deduplication-key'
		);
		INSERT INTO risk_projection_versions (
			id, projection_kind, organization_id, version, source, snapshot,
			advisory_only, calculated_at
		) VALUES (
			'risk-deduplication-1', 'SYSTEM', NULL, 1, 'configured-local', '{}', true, now()
		);
		INSERT INTO idempotency_responses (
			scope, operation_id, semantic_hash, response_status, response_body
		) VALUES (
			'deduplication', 'operation-deduplication', 'sha256:deduplication', 200, '{}'
		)
	`); err != nil {
		t.Fatalf("seed scoped deduplication records: %v", err)
	}

	for name, statement := range map[string]string{
		"communication sender/idempotency": `
			INSERT INTO communication_messages (
				id, thread_id, organization_id, visibility, sender_subject_id,
				audience, direction, subject, body, idempotency_key
			) VALUES (
				'message-deduplication-2', 'thread-deduplication', 'operator-immutable',
				'AUDITEE_VISIBLE', 'subject-immutable', 'AUDITEE', 'CAA_TO_AUDITEE',
				'Subject', 'Changed body', 'message-deduplication-key'
			)`,
		"document version": `
			INSERT INTO document_versions (
				id, document_id, organization_id, version, visibility, status,
				file_name, media_type, sha256, size_bytes, created_by_subject_id
			) VALUES (
				'document-version-deduplication-2', 'document-deduplication',
				'operator-immutable', 1, 'AUDITEE_VISIBLE', 'LOCKED', 'other.pdf',
				'application/pdf', 'sha256:other', 11, 'subject-immutable'
			)`,
		"notification recipient/key": `
			INSERT INTO notification_records (
				id, recipient_subject_id, organization_id, title, body, deduplication_key
			) VALUES (
				'notification-deduplication-2', 'subject-immutable', 'operator-immutable',
				'Changed', 'Changed', 'notification-deduplication-key'
			)`,
		"global risk kind/version": `
			INSERT INTO risk_projection_versions (
				id, projection_kind, organization_id, version, source, snapshot,
				advisory_only, calculated_at
			) VALUES (
				'risk-deduplication-2', 'SYSTEM', NULL, 1, 'configured-local', '{}', true, now()
			)`,
		"idempotency scope/operation": `
			INSERT INTO idempotency_responses (
				scope, operation_id, semantic_hash, response_status, response_body
			) VALUES (
				'deduplication', 'operation-deduplication', 'sha256:changed', 200, '{}'
			)`,
	} {
		if _, err := pool.Exec(context.Background(), statement); err == nil {
			t.Errorf("%s duplicate insert succeeded", name)
		}
	}
}

func TestFullPlatformCommandTransactionLinkage(t *testing.T) {
	pool := createTestDatabase(t, "full_platform_linkage")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	seedFullPlatformReferences(t, pool)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO idempotency_responses (
			scope, operation_id, semantic_hash, response_status, response_body
		) VALUES ('full-platform', 'operation-linked', 'sha256:linked', 200, '{}');
		INSERT INTO audit_events (
			event_id, occurred_at, actor_subject_id, actor_role, organization_id, action,
			entity_type, entity_id, operation_id, details
		) VALUES (
			'audit-linked', now(), 'subject-immutable', 'manager', 'operator-immutable',
			'full_platform.changed', 'full_platform', 'entity-linked', 'operation-linked', '{}'
		);
		INSERT INTO authorized_sync_changes (
			subject_id, organization_id, kind, entity_id, entity_revision, payload, operation_id
		) VALUES (
			'subject-immutable', 'operator-immutable', 'full_platform', 'entity-linked', 1,
			'{}', 'operation-linked'
		);
		INSERT INTO outbox_messages (
			id, topic, aggregate_type, aggregate_id, payload, idempotency_key, operation_id
		) VALUES (
			'outbox-linked', 'full-platform.changed', 'full_platform', 'entity-linked',
			'{}', 'outbox-operation-linked', 'operation-linked'
		);
		INSERT INTO command_transaction_links (
			operation_id, idempotency_scope, audit_event_id, change_sequence_id,
			outbox_message_id, created_at
		)
		SELECT
			'operation-linked', 'full-platform', 'audit-linked', sequence_id,
			'outbox-linked', now()
		FROM authorized_sync_changes
		WHERE operation_id = 'operation-linked'
	`); err != nil {
		t.Fatalf("insert linked command envelope: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO idempotency_responses (
			scope, operation_id, semantic_hash, response_status, response_body
		) VALUES ('full-platform-other', 'operation-linked', 'sha256:linked-other', 200, '{}');
		INSERT INTO audit_events (
			event_id, occurred_at, actor_subject_id, actor_role, organization_id, action,
			entity_type, entity_id, operation_id, details
		) VALUES (
			'audit-linked-other', now(), 'subject-immutable', 'manager', 'operator-immutable',
			'full_platform.other', 'full_platform', 'entity-linked-other', 'operation-linked', '{}'
		);
		INSERT INTO authorized_sync_changes (
			subject_id, organization_id, kind, entity_id, entity_revision, payload, operation_id
		) VALUES (
			'subject-immutable', 'operator-immutable', 'full_platform', 'entity-linked-other', 1,
			'{}', 'operation-linked'
		);
		INSERT INTO outbox_messages (
			id, topic, aggregate_type, aggregate_id, payload, idempotency_key, operation_id
		) VALUES (
			'outbox-linked-other', 'full-platform.other', 'full_platform', 'entity-linked-other',
			'{}', 'outbox-operation-linked-other', 'operation-linked'
		);
		INSERT INTO command_transaction_links (
			operation_id, idempotency_scope, audit_event_id, change_sequence_id,
			outbox_message_id, created_at
		)
		SELECT
			'operation-linked', 'full-platform-other', 'audit-linked-other', sequence_id,
			'outbox-linked-other', now()
		FROM authorized_sync_changes
		WHERE entity_id = 'entity-linked-other'
	`); err != nil {
		t.Fatalf("insert same operation ID in a different idempotency scope: %v", err)
	}
	var linked int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM command_transaction_links link
		JOIN idempotency_responses response
		  ON response.scope = link.idempotency_scope
		 AND response.operation_id = link.operation_id
		JOIN audit_events event ON event.event_id = link.audit_event_id
		JOIN authorized_sync_changes change ON change.sequence_id = link.change_sequence_id
		JOIN outbox_messages message ON message.id = link.outbox_message_id
		WHERE event.operation_id = link.operation_id
		  AND change.operation_id = link.operation_id
		  AND message.operation_id = link.operation_id
	`).Scan(&linked); err != nil || linked != 2 {
		t.Fatalf("linked command envelope count = %d, err = %v", linked, err)
	}
}

func TestFullPlatformApplicationTransitionWritesLinkedEnvelope(t *testing.T) {
	pool := canonicalDatabase(t, "full_platform_application_linkage")
	service := testService(pool)
	_, err := service.ConvertPotentialFinding(
		context.Background(),
		principal("lead-001", "caa", "session-lead", identity.RoleLeadInspector),
		application.ConvertPotentialFindingCommand{
			OperationID:        "operation-application-linked",
			CorrelationID:      "correlation-application-linked",
			PotentialFindingID: "potential-cabin-001",
			ExpectedRevision:   1,
			Severity:           potentialfindings.SeverityLevel2Major,
		},
	)
	if err != nil {
		t.Fatalf("execute linked application transition: %v", err)
	}

	var linked int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM command_transaction_links link
		JOIN audit_events event ON event.event_id = link.audit_event_id
		JOIN authorized_sync_changes change ON change.sequence_id = link.change_sequence_id
		JOIN outbox_messages message ON message.id = link.outbox_message_id
		WHERE link.operation_id = 'operation-application-linked'
		  AND event.operation_id = link.operation_id
		  AND change.operation_id = link.operation_id
		  AND message.operation_id = link.operation_id
	`).Scan(&linked); err != nil || linked != 1 {
		t.Fatalf("application command linkage count = %d, err = %v", linked, err)
	}
}

func TestNewModuleQueriesDoNotWriteAnotherModulesTables(t *testing.T) {
	moduleRoot := apiModuleRoot(t)
	writePattern := regexp.MustCompile(`(?im)\b(?:INSERT\s+INTO|UPDATE|DELETE\s+FROM)\s+([a-z_][a-z0-9_]*)`)
	for module, allowedTables := range newPersistentModuleTables {
		queryPath := filepath.Join(moduleRoot, "internal", module, "store", "postgres", "queries.sql")
		contents, err := os.ReadFile(queryPath)
		if err != nil {
			t.Errorf("read %s queries: %v", module, err)
			continue
		}
		for _, match := range writePattern.FindAllStringSubmatch(string(contents), -1) {
			table := strings.ToLower(match[1])
			if !allowedTables[table] {
				t.Errorf("%s query writes table %s owned by another module", module, table)
			}
		}
	}
}

func applyMigrationFilesThrough(t *testing.T, pool *database.Pool, version int) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		CREATE TABLE schema_migrations (
			version bigint PRIMARY KEY,
			name text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`); err != nil {
		t.Fatalf("create N-1 migration ledger: %v", err)
	}
	for candidate := 1; candidate <= version; candidate++ {
		matches, err := filepath.Glob(filepath.Join(
			apiModuleRoot(t),
			"migrations",
			migrationFileName(candidate)+"*.up.sql",
		))
		if err != nil || len(matches) != 1 {
			t.Fatalf("resolve migration %d: matches=%v err=%v", candidate, matches, err)
		}
		contents, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatalf("read migration %d: %v", candidate, err)
		}
		if _, err := pool.Exec(ctx, string(contents)); err != nil {
			t.Fatalf("apply migration %d: %v", candidate, err)
		}
		if _, err := pool.Exec(
			ctx,
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
			candidate,
			filepath.Base(matches[0]),
		); err != nil {
			t.Fatalf("record migration %d: %v", candidate, err)
		}
	}
}

func seedFullPlatformReferences(t *testing.T, pool *database.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO organizations (
			id, legal_name, organization_type, status
		) VALUES (
			'operator-immutable', 'Immutable Operator', 'OPERATOR', 'ACTIVE'
		);
		INSERT INTO identity_references (
			subject_id, issuer, display_name
		) VALUES (
			'subject-immutable', 'test', 'Immutable Subject'
		)
	`); err != nil {
		t.Fatalf("seed full-platform references: %v", err)
	}
}

func migrationFileName(version int) string {
	return fmt.Sprintf("%06d_", version)
}
