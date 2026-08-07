package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/datafeed"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

var canonicalNow = time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)

func canonicalDatabase(t *testing.T, label string) *database.Pool {
	t.Helper()
	pool := createTestDatabase(t, label)
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	type seedStatement struct {
		sql  string
		args []any
	}
	statements := []seedStatement{
		{sql: `INSERT INTO organizations (id, legal_name, organization_type, status) VALUES
			('caa', 'Civil Aviation Authority', 'AUTHORITY', 'ACTIVE'),
			('airline-xyz', 'Airline XYZ', 'OPERATOR', 'ACTIVE'),
			('airline-other', 'Other Airline', 'OPERATOR', 'ACTIVE')`},
		{sql: `INSERT INTO identity_references (subject_id, issuer, display_name, email) VALUES
			('inspector-cabin-001', 'test', 'Cabin Inspector', 'inspector.cabin@example.test'),
			('inspector-other', 'test', 'Other Inspector', 'inspector.other@example.test'),
			('lead-001', 'test', 'Lead Inspector', 'lead@example.test'),
			('auditee-xyz', 'test', 'Airline XYZ Auditee', 'auditee.xyz@example.test'),
			('auditee-other', 'test', 'Other Airline Auditee', 'auditee.other@example.test'),
			('manager-001', 'test', 'Department Manager', 'manager@example.test'),
			('gm-001', 'test', 'General Manager', 'gm@example.test'),
			('executive-001', 'test', 'Executive Director', 'executive@example.test')`},
		{sql: `INSERT INTO session_references (id, subject_id, organization_id, expires_at, last_seen_at, absolute_expires_at, roles) VALUES
			('session-inspector', 'inspector-cabin-001', 'caa', $1, $2, $1, ARRAY['inspector']),
			('session-lead', 'lead-001', 'caa', $1, $2, $1, ARRAY['leadInspector']),
			('session-auditee', 'auditee-xyz', 'airline-xyz', $1, $2, $1, ARRAY['auditee']),
			('session-manager', 'manager-001', 'caa', $1, $2, $1, ARRAY['manager']),
			('session-gm', 'gm-001', 'caa', $1, $2, $1, ARRAY['gm']),
			('session-executive', 'executive-001', 'caa', $1, $2, $1, ARRAY['executiveDirector'])`, args: []any{canonicalNow.Add(24 * time.Hour), canonicalNow}},
		{sql: `INSERT INTO caa_department_memberships (id, subject_id, department_id, organizational_unit_id, membership_role, effective_from, status)
			VALUES ('membership-manager-001', 'manager-001', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', '2026-01-01', 'ACTIVE')`},
		{sql: `INSERT INTO regulated_targets (id, target_kind, organization_id)
			VALUES ('target-airline-xyz', 'ORGANIZATION', 'airline-xyz')`},
		{sql: `INSERT INTO organization_service_provider_scopes (
				id, organization_id, service_provider_type_id, authorization_identifier, status,
				effective_from, primary_target_id
			) VALUES (
				'scope-airline-xyz-air-operator', 'airline-xyz', 'AIR_OPERATOR', 'AOC-XYZ-001', 'ACTIVE',
				'2026-01-01', 'target-airline-xyz'
			)`},
		{sql: `INSERT INTO organization_service_provider_scope_targets (organization_service_provider_scope_id, regulated_target_id)
			VALUES ('scope-airline-xyz-air-operator', 'target-airline-xyz')`},
		{sql: `INSERT INTO question_versions (id, question_id, version, prompt, configured_reference, expected_evidence, created_by_subject_id)
			VALUES ('q-cabin-crew-training', 'q-cabin-crew-training', 1,
				'Are crew training records current and complete?', 'ICAO Annex 6', 'Current crew training records.', 'manager-001')`},
		{sql: `INSERT INTO canonical_question_catalogs (
				id, catalog_version, usage_class, profile_name, profile_version, status,
				source_package_version, source_package_json_sha256, source_package_zip_sha256,
				root_digest, question_count, form_count, created_by_subject_id
			) VALUES (
				'catalog-cabin-fixture', 'aga-fixture@1.0.0', 'PREPROD_EXERCISE', 'aga-preprod', '1.0.0', 'SEALED',
				'fixture-1.0.0', 'sha256:fixture-package-json', 'sha256:fixture-package-zip',
				'sha256:fixture-catalog-root', 1, 1, 'manager-001'
			)`},
		{sql: `INSERT INTO canonical_question_catalog_forms (catalog_id, form_code, form_digest, archive_digest, question_count, source_gap_state)
			VALUES ('catalog-cabin-fixture', 'CABIN', 'sha256:fixture-form', 'sha256:fixture-archive', 1, 'SOURCE_MAPPING_REQUIRED')`},
		{sql: `INSERT INTO canonical_question_version_provenance (question_version_id, usage_class, catalog_id)
			VALUES ('q-cabin-crew-training', 'PREPROD_EXERCISE', 'catalog-cabin-fixture')`},
		{sql: `INSERT INTO canonical_question_catalog_memberships (
				catalog_id, question_version_id, usage_class, form_code, proposal_id, ordinal,
				question_digest, source_locator, source_gap_state, proposed_domain, proposed_topic, proposed_risk_band
			) VALUES (
				'catalog-cabin-fixture', 'q-cabin-crew-training', 'PREPROD_EXERCISE', 'CABIN', 'proposal-cabin-001', 1,
				'sha256:fixture-question', 'fixture://cabin/1', 'SOURCE_MAPPING_REQUIRED', 'Cabin Safety', 'Crew Training', 'MEDIUM'
			)`},
		{sql: `INSERT INTO canonical_question_catalog_applicabilities (
				catalog_id, question_version_id, provider_scope_id, regulated_target_id,
				status, reason, actor_subject_id
			) VALUES (
				'catalog-cabin-fixture', 'q-cabin-crew-training', 'scope-airline-xyz-air-operator',
				'target-airline-xyz', 'ELIGIBLE', 'fixture active provider/target eligibility', 'manager-001'
			)`},
		{sql: `INSERT INTO surveillance_plan_items (
				id, title, plan_year, organization_id, inspection_type, scheduled_date, estimated_budget,
				status, current_owner_role, next_action, revision, created_at, updated_at
			) VALUES (
				'planning-item-cabin-canonical', 'Cabin Safety Canonical Fixture', 2026, 'airline-xyz', 'CABIN',
				'2026-07-25', 1000, 'RELEASED', 'Department Manager', 'Assign Lead Inspector', 1,
				'2026-07-21T12:00:00Z', '2026-07-21T12:00:00Z'
			)`},
		{sql: `INSERT INTO planning_intake_drafts (
				id, organization_id, values, submitted_planning_item_id, revision, created_by_subject_id, created_at, updated_at
			) VALUES (
				'draft-cabin-canonical', 'airline-xyz',
				'{"organizationId":"airline-xyz","organizationName":"Airline XYZ","applicationType":"Routine Cabin Safety","domain":"Cabin Safety","inspectionCategory":"Routine / Announced","noticePolicy":"ADVANCE","purpose":"Fixture canonical audit","triggerType":"Periodic","riskCategory":"Operational","plannedDate":"2026-07-25","mode":"ANNOUNCED","location":"Windhoek","catalogVersion":"aga-fixture@1.0.0","providerScopeId":"scope-airline-xyz-air-operator","regulatedTargetId":"target-airline-xyz","selectedQuestionVersionIds":["q-cabin-crew-training"],"selectionDigest":"sha256:fixture-selection","estimatedResourceRequirement":1,"requestedBudget":1000,"currency":"NAD"}',
				'planning-item-cabin-canonical', 1, 'manager-001', '2026-07-21T12:00:00Z', '2026-07-21T12:00:00Z'
			)`},
		{sql: `INSERT INTO canonical_audit_scope_drafts (
				id, planning_intake_draft_id, organization_id, provider_scope_id, regulated_target_id,
				audit_type, catalog_id, usage_class, revision, status, selected_question_count,
				selection_digest, requested_budget, notice_policy, created_by_subject_id
			) VALUES (
				'scope-draft-cabin-fixture', 'draft-cabin-canonical', 'airline-xyz',
				'scope-airline-xyz-air-operator', 'target-airline-xyz', 'CABIN', 'catalog-cabin-fixture',
				'PREPROD_EXERCISE', 1, 'RELEASED', 1, 'sha256:fixture-selection', 1000, 'ADVANCE', 'manager-001'
			)`},
		{sql: `INSERT INTO canonical_audit_scope_draft_questions (
				scope_draft_id, revision, catalog_id, question_version_id, position, selection_digest
			) VALUES ('scope-draft-cabin-fixture', 1, 'catalog-cabin-fixture', 'q-cabin-crew-training', 0, 'sha256:fixture-selection')`},
		{sql: `INSERT INTO canonical_audit_scope_snapshots (
				id, scope_draft_id, revision, stage, catalog_id, usage_class, selection_digest,
				selected_question_count, snapshot, planning_snapshot_digest, created_by_subject_id
			) VALUES (
				'scope-snapshot-cabin-submitted', 'scope-draft-cabin-fixture', 1, 'SUBMITTED', 'catalog-cabin-fixture',
				'PREPROD_EXERCISE', 'sha256:fixture-selection', 1,
				'{"planningItemId":"planning-item-cabin-canonical","catalogVersion":"aga-fixture@1.0.0","selectedQuestionVersionIds":["q-cabin-crew-training"]}',
				governed_jsonb_sha256('{"planningItemId":"planning-item-cabin-canonical","catalogVersion":"aga-fixture@1.0.0","selectedQuestionVersionIds":["q-cabin-crew-training"]}'::jsonb), 'manager-001'
			)`},
		{sql: `INSERT INTO canonical_audit_scope_snapshots (
				id, scope_draft_id, revision, stage, catalog_id, usage_class, selection_digest,
				selected_question_count, snapshot, planning_snapshot_digest, created_by_subject_id
			) VALUES (
				'scope-snapshot-cabin-001', 'scope-draft-cabin-fixture', 1, 'RELEASED', 'catalog-cabin-fixture',
				'PREPROD_EXERCISE', 'sha256:fixture-selection', 1,
				'{"planningItemId":"planning-item-cabin-canonical","catalogVersion":"aga-fixture@1.0.0","selectedQuestionVersionIds":["q-cabin-crew-training"]}',
				governed_jsonb_sha256('{"planningItemId":"planning-item-cabin-canonical","catalogVersion":"aga-fixture@1.0.0","selectedQuestionVersionIds":["q-cabin-crew-training"]}'::jsonb), 'manager-001'
			)`},
		{sql: `INSERT INTO canonical_audit_scope_snapshot_questions (snapshot_id, catalog_id, question_version_id, position)
			VALUES
				('scope-snapshot-cabin-submitted', 'catalog-cabin-fixture', 'q-cabin-crew-training', 0),
				('scope-snapshot-cabin-001', 'catalog-cabin-fixture', 'q-cabin-crew-training', 0)`},
		{sql: `INSERT INTO inspections (id, organization_id, assigned_inspector_subject_id, title, inspection_type, status, due_date, revision)
		 VALUES ('audit-cabin-001', 'airline-xyz', 'inspector-cabin-001', 'Cabin Safety Inspection', 'CABIN', 'IN_PROGRESS', '2026-08-01', 1)`},
		{sql: `INSERT INTO audit_assignments (id, inspection_id, planning_item_id, released_scope_snapshot_id, organization_id, lead_subject_id, status, scheduled_start_date, scheduled_end_date, revision)
		 VALUES ('assignment-cabin-001', 'audit-cabin-001', 'planning-item-cabin-canonical', 'scope-snapshot-cabin-001', 'airline-xyz', 'lead-001', 'IN_PROGRESS', '2026-07-25', '2026-08-01', 1)`},
		{sql: `INSERT INTO audit_team_members (assignment_id, subject_id, member_role, revision)
		 VALUES ('assignment-cabin-001', 'lead-001', 'LEAD_INSPECTOR', 1),
		        ('assignment-cabin-001', 'inspector-cabin-001', 'INSPECTOR', 1)`},
		{sql: `INSERT INTO audit_question_assignments (assignment_id, question_id, subject_id, revision)
		 VALUES ('assignment-cabin-001', 'q-cabin-crew-training', 'inspector-cabin-001', 1)`},
		{sql: `INSERT INTO checklist_template_versions (id, template_id, version, title, snapshot, published_at)
		 VALUES ('template-cabin-v1', 'template-cabin', 1, 'Cabin Checklist', '{"questionIds":["q-cabin-crew-training"]}', $1)`, args: []any{canonicalNow}},
		{sql: `INSERT INTO inspection_packages (id, inspection_id, checklist_template_version_id, canonical_scope_snapshot_id, package_version, snapshot, expires_at, package_digest)
		 VALUES ('package-cabin-001', 'audit-cabin-001', NULL, 'scope-snapshot-cabin-001', 1, '{"questionIds":["q-cabin-crew-training"],"questionVersionIds":["q-cabin-crew-training"]}', $1, 'sha256:package-cabin-001')`, args: []any{canonicalNow.Add(72 * time.Hour)}},
		{sql: `INSERT INTO inspection_question_assignments (inspection_id, question_id, subject_id, assignment_revision)
		 VALUES ('audit-cabin-001', 'q-cabin-crew-training', 'inspector-cabin-001', 1)`},
		{sql: `INSERT INTO inspection_checklists (inspection_id, status, revision) VALUES ('audit-cabin-001', 'IN_PROGRESS', 1)`},
		{sql: `INSERT INTO checklist_responses (id, inspection_id, package_id, question_id, assigned_inspector_subject_id, response_value, comment_to_auditee, internal_caa_note, revision)
		 VALUES ('response-cabin-001', 'audit-cabin-001', 'package-cabin-001', 'q-cabin-crew-training', 'inspector-cabin-001', 'NON_COMPLIANT', 'Training record gap.', 'Internal workload note.', 1)`},
		{sql: `INSERT INTO potential_findings (
			id, inspection_id, checklist_response_id, organization_id, status, finding_basis, expected_evidence,
			comment_to_auditee, internal_caa_note, revision, question_id, title, description, created_by_subject_id
		) VALUES (
			'potential-cabin-001', 'audit-cabin-001', 'response-cabin-001', 'airline-xyz', 'PENDING_LEAD_REVIEW',
			'Crew training records incomplete.', 'Updated training records.', 'Provide corrective records.',
			'Internal CAA Note: monitor repeat pattern.', 1, 'q-cabin-crew-training', 'Crew training record gap',
			'Crew training records incomplete.', 'inspector-cabin-001'
		)`},
		{sql: `INSERT INTO report_versions (id, report_id, inspection_id, version, status, snapshot)
		 VALUES ('report-preliminary-canonical', 'report-preliminary-canonical', 'audit-cabin-001', 1, 'ISSUED',
		         '{"kind":"PRELIMINARY","ready":true,"potentialFindingIds":["potential-cabin-001"],"findingIds":[],"contentHash":"sha256:preliminary-canonical"}')`},
		{sql: `INSERT INTO report_approval_states (report_version_id, status, revision, issued_at)
		 VALUES ('report-preliminary-canonical', 'ISSUED', 1, '2026-07-21T12:00:00Z')`},
	}
	for _, statement := range statements {
		if _, err := pool.Exec(context.Background(), statement.sql, statement.args...); err != nil {
			t.Fatalf("seed canonical database: %v\n%s", err, statement.sql)
		}
	}
	return pool
}

func testService(pool *database.Pool) *application.Service {
	counters := map[string]int{}
	writer, err := datafeed.NewWriter(datafeed.WriterConfig{
		TenantID:      "tenant-canonical-fixture",
		PayloadKey:    []byte("0123456789abcdef0123456789abcdef"),
		PayloadKeyRef: "canonical-fixture-key",
	})
	if err != nil {
		panic(err)
	}
	return application.NewService(pool, application.Dependencies{
		Clock:          func() time.Time { return canonicalNow },
		DataFeedWriter: writer,
		IDGenerator: func(prefix string) string {
			counters[prefix]++
			return fmt.Sprintf("%s-test-%03d", prefix, counters[prefix])
		},
	})
}

func principal(subject, organization, sessionID string, roles ...identity.Role) identity.Principal {
	return identity.Principal{SubjectID: subject, OrganizationID: organization, SessionID: sessionID, Roles: roles}
}

func seedFinding(t *testing.T, pool *database.Pool, id, reference, organization string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO findings (id, reference, inspection_id, organization_id, severity, status, owner_subject_id, next_action, due_date, revision)
		VALUES ($1, $2, 'audit-cabin-001', $3, 'LEVEL_2_MAJOR', 'WAITING_FOR_CAP', 'auditee-xyz', 'Submit CAP', '2026-08-15', 1)
	`, id, reference, organization); err != nil {
		t.Fatalf("seed Finding: %v", err)
	}
}
