package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/httpapi"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/organizations"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/session"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

func TestTask2ScopeSupersessionSelectsOnlyCurrentFact(t *testing.T) {
	pool := createTestDatabase(t, "task2_scope_supersession")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO organizations (id, legal_name, organization_type, status)
		VALUES ('scope-successor-org', 'Scope Successor', 'OPERATOR', 'ACTIVE');
		INSERT INTO regulated_targets (id, target_kind, organization_id)
		VALUES ('scope-successor-target', 'ORGANIZATION', 'scope-successor-org');
		INSERT INTO organization_service_provider_scopes (
			id, root_id, organization_id, service_provider_type_id, authorization_identifier,
			status, effective_from, primary_target_id
		) VALUES (
			'scope-root', 'scope-root', 'scope-successor-org', 'AIR_OPERATOR', 'AOC-SUPERSEDE',
			'ACTIVE', '2025-01-01', 'scope-successor-target'
		);
		INSERT INTO organization_service_provider_scopes (
			id, root_id, supersedes_id, organization_id, service_provider_type_id, authorization_identifier,
			status, effective_from, primary_target_id
		) VALUES (
			'scope-revoked', 'scope-root', 'scope-root', 'scope-successor-org', 'AIR_OPERATOR', 'AOC-SUPERSEDE',
			'REVOKED', '2026-01-01', 'scope-successor-target'
		);
		INSERT INTO organization_service_provider_scopes (
			id, root_id, supersedes_id, organization_id, service_provider_type_id, authorization_identifier,
			status, effective_from, primary_target_id
		) VALUES (
			'scope-renewed', 'scope-root', 'scope-revoked', 'scope-successor-org', 'AIR_OPERATOR', 'AOC-SUPERSEDE',
			'ACTIVE', '2027-01-01', 'scope-successor-target'
		);
	`); err != nil {
		t.Fatalf("seed supersession chain: %v", err)
	}
	for date, expectedID := range map[time.Time]string{
		time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC): "scope-root",
		time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC): "",
		time.Date(2027, time.June, 1, 0, 0, 0, 0, time.UTC): "scope-renewed",
	} {
		scopes, err := organizations.ListApplicableServiceProviderScopes(context.Background(), pool, "scope-successor-org", date)
		if err != nil {
			t.Fatalf("resolve scope at %s: %v", date.Format("2006-01-02"), err)
		}
		if expectedID == "" && len(scopes) != 0 {
			t.Fatalf("revoked scope remained applicable at %s: %+v", date.Format("2006-01-02"), scopes)
		}
		if expectedID != "" && (len(scopes) != 1 || scopes[0].ID != expectedID) {
			t.Fatalf("applicable scope at %s = %+v, want %s", date.Format("2006-01-02"), scopes, expectedID)
		}
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO organization_service_provider_scopes (
			id, root_id, supersedes_id, organization_id, service_provider_type_id, authorization_identifier,
			status, effective_from, primary_target_id
		) VALUES (
			'scope-conflict', 'scope-root', 'scope-root', 'scope-successor-org', 'AIR_OPERATOR', 'AOC-SUPERSEDE',
			'SUSPENDED', '2026-02-01', 'scope-successor-target'
		)
	`); err == nil {
		t.Fatal("a second successor for one scope fact was accepted")
	}
}

func TestTask2MembershipRequiresCurrentConsistentActiveDepartmentUnit(t *testing.T) {
	pool := createTestDatabase(t, "task2_membership_consistency")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(context.Background(), "INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('task2-manager', 'test', 'Task 2 Manager')"); err != nil {
		t.Fatalf("seed manager identity: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO caa_department_memberships (
			id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from
		) VALUES (
			'task2-invalid-unit-pair', 'task2-manager', 'FLIGHT_OPERATIONS_INSPECTORATE', 'AIRWORTHINESS_INSPECTORATE',
			'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01'
		)
	`); err == nil {
		t.Fatal("cross-department unit assignment was accepted")
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO caa_department_memberships (
			id, root_id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from
		) VALUES (
			'task2-manager-root', 'task2-manager-root', 'task2-manager', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE',
			'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01'
		)
	`); err != nil {
		t.Fatalf("seed valid manager membership: %v", err)
	}
	assignments, err := identity.ResolveEffectiveDepartmentAssignments(context.Background(), pool, "task2-manager", time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("resolve active manager: %v", err)
	}
	if len(assignments) != 1 || assignments[0].OrganizationalUnitID != "FLIGHT_OPERATIONS_INSPECTORATE" {
		t.Fatalf("active consistent assignment = %+v", assignments)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO caa_organizational_unit_status_facts (
			id, root_id, supersedes_id, organizational_unit_id, status, effective_from
		) VALUES (
			'task2-inactive-unit', 'seed-unit-status-FLIGHT_OPERATIONS_INSPECTORATE',
			'seed-unit-status-FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'INACTIVE', '2026-01-01'
		)
	`); err != nil {
		t.Fatalf("append inactive-unit status fact: %v", err)
	}
	assignments, err = identity.ResolveEffectiveDepartmentAssignments(context.Background(), pool, "task2-manager", time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("resolve inactive-unit manager: %v", err)
	}
	if len(assignments) != 0 {
		t.Fatalf("inactive unit retained manager authority: %+v", assignments)
	}
}

func TestTask2MembershipRootIdentityHasOneSuccessorChainAndExactEffectiveResolution(t *testing.T) {
	pool := createTestDatabase(t, "task2_membership_root_identity")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (subject_id, issuer, display_name)
		VALUES ('task2-root-manager', 'test', 'Task 2 Root Manager');
		INSERT INTO caa_department_memberships (
			id, root_id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from
		) VALUES (
			'task2-membership-root', 'task2-membership-root', 'task2-root-manager',
			'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01'
		);
		INSERT INTO caa_department_memberships (
			id, root_id, supersedes_id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from
		) VALUES (
			'task2-membership-revoked', 'task2-membership-root', 'task2-membership-root', 'task2-root-manager',
			'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'REVOKED', '2026-01-01'
		);
		INSERT INTO caa_department_memberships (
			id, root_id, supersedes_id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from
		) VALUES (
			'task2-membership-renewed', 'task2-membership-root', 'task2-membership-revoked', 'task2-root-manager',
			'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'ACTIVE', '2027-01-01'
		)
	`); err != nil {
		t.Fatalf("seed membership successor chain: %v", err)
	}
	for at, want := range map[time.Time]bool{
		time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC): true,
		time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC): false,
		time.Date(2027, time.June, 1, 0, 0, 0, 0, time.UTC): true,
	} {
		assignments, err := identity.ResolveEffectiveDepartmentAssignments(context.Background(), pool, "task2-root-manager", at)
		if err != nil {
			t.Fatalf("resolve membership at %s: %v", at.Format("2006-01-02"), err)
		}
		if got := len(assignments) == 1 && assignments[0].DepartmentID == "FLIGHT_OPERATIONS_INSPECTORATE"; got != want {
			t.Fatalf("assignment at %s = %+v, want active=%t", at.Format("2006-01-02"), assignments, want)
		}
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO caa_department_memberships (
			id, root_id, supersedes_id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from
		) VALUES (
			'task2-membership-conflict', 'task2-membership-root', 'task2-membership-root', 'task2-root-manager',
			'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'REVOKED', '2026-02-01'
		)
	`); err == nil {
		t.Fatal("second successor was accepted for one membership fact")
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO caa_department_memberships (
			id, root_id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from
		) VALUES (
			'task2-membership-independent-root', 'task2-membership-independent-root', 'task2-root-manager',
			'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'ACTIVE', '2028-01-01'
		)
	`); err == nil {
		t.Fatal("independent membership root bypass was accepted")
	}
}

func TestTask2DepartmentAndUnitStatusFactsRemainAppendOnlyAndRemoveAuthority(t *testing.T) {
	pool := createTestDatabase(t, "task2_department_status_facts")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('task2-status-manager', 'test', 'Status Manager');
		INSERT INTO caa_department_memberships (id, root_id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from)
		VALUES ('task2-status-membership', 'task2-status-membership', 'task2-status-manager', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01');
		INSERT INTO caa_organizational_unit_status_facts (id, root_id, supersedes_id, organizational_unit_id, status, effective_from)
		VALUES ('task2-unit-inactive', 'seed-unit-status-FLIGHT_OPERATIONS_INSPECTORATE', 'seed-unit-status-FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'INACTIVE', '2026-01-01');
	`); err != nil {
		t.Fatalf("append public status facts: %v", err)
	}
	assignments, err := identity.ResolveEffectiveDepartmentAssignments(context.Background(), pool, "task2-status-manager", time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("resolve inactive unit: %v", err)
	}
	if len(assignments) != 0 {
		t.Fatalf("inactive unit retained authority: %+v", assignments)
	}
}

func TestTask2RejectsSameKindCrossOrganizationTargetsAndRepairsDerivedIndex(t *testing.T) {
	pool := createTestDatabase(t, "task2_target_identity_repair")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO organizations (id,legal_name,organization_type,status) VALUES ('target-org-a','A','OPERATOR','ACTIVE'),('target-org-b','B','OPERATOR','ACTIVE');
		INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('target-history-owner', 'test', 'Target History Owner');
		INSERT INTO inspections (id, organization_id, assigned_inspector_subject_id, title, inspection_type, status, revision)
		VALUES ('target-history-audit', 'target-org-a', 'target-history-owner', 'Target history audit', 'RAMP', 'IN_PROGRESS', 1);
		INSERT INTO regulatory_reference_versions (id, reference_id, version, title, status, effective_date, snapshot)
		VALUES ('target-history-source', 'target-history-source', 1, 'Target history source', 'ACTIVE', '2025-01-01', '{"digest":"sha256:target-history-source"}');
		INSERT INTO review_decisions (id, entity_type, entity_id, expected_revision, decision, reason, decided_by_subject_id, decided_at)
		VALUES ('target-history-review', 'regulatory_reference_version', 'target-history-source', 1, 'ACKNOWLEDGED', 'Representative review history.', 'target-history-owner', now());
		INSERT INTO checklist_template_versions (id, template_id, version, title, snapshot, published_at)
		VALUES ('target-history-template-v1', 'target-history-template', 1, 'Target history template', '{"digest":"sha256:target-history-template"}', now());
		INSERT INTO inspection_packages (id, inspection_id, checklist_template_version_id, package_version, snapshot, expires_at, package_digest)
		VALUES ('target-history-package-v1', 'target-history-audit', 'target-history-template-v1', 1, '{"digest":"sha256:target-history-package"}', now() + interval '1 day', 'sha256:target-history-package');
		INSERT INTO caa_department_memberships (id, root_id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from)
		VALUES ('target-history-membership', 'target-history-membership', 'target-history-owner', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01');
		INSERT INTO regulated_targets (id,target_kind,organization_id) VALUES ('target-org-a','ORGANIZATION','target-org-a');
		INSERT INTO organization_service_provider_scopes (id,root_id,organization_id,service_provider_type_id,authorization_identifier,status,effective_from,primary_target_id) VALUES ('target-scope-valid','target-scope-valid','target-org-a','AIR_OPERATOR','AOC-VALID','ACTIVE','2025-01-01','target-org-a');
		INSERT INTO regulated_targets (id,target_kind,organization_id) VALUES ('target-org-b','ORGANIZATION','target-org-b');
	`); err != nil {
		t.Fatalf("seed valid scope and foreign target: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO organization_service_provider_scopes (id,organization_id,service_provider_type_id,authorization_identifier,status,effective_from,primary_target_id) VALUES ('target-scope-a','target-org-a','AIR_OPERATOR','AOC-A','ACTIVE','2025-01-01','target-org-b')
	`); err == nil {
		t.Fatal("same-kind cross-organization primary target was accepted")
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO organization_service_provider_scope_targets (organization_service_provider_scope_id,regulated_target_id) VALUES ('target-scope-valid','target-org-b');
	`); err == nil {
		t.Fatal("same-kind cross-organization linked target was accepted")
	}
	var beforeSourceSnapshot, beforeTemplateSnapshot, beforePackageSnapshot, beforePackageDigest string
	var beforeReviewRow, beforeMembershipRow, beforeScopeRow string
	if err := pool.QueryRow(context.Background(), `
		SELECT source.snapshot::text, template.snapshot::text, package.snapshot::text, package.package_digest,
		       to_jsonb(review)::text, to_jsonb(membership)::text, to_jsonb(scope)::text
		FROM regulatory_reference_versions source
		JOIN checklist_template_versions template ON template.id = 'target-history-template-v1'
		JOIN inspection_packages package ON package.id = 'target-history-package-v1'
		JOIN review_decisions review ON review.id = 'target-history-review'
		JOIN caa_department_memberships membership ON membership.id = 'target-history-membership'
		JOIN organization_service_provider_scopes scope ON scope.id = 'target-scope-valid'
		WHERE source.id = 'target-history-source'
	`).Scan(
		&beforeSourceSnapshot,
		&beforeTemplateSnapshot,
		&beforePackageSnapshot,
		&beforePackageDigest,
		&beforeReviewRow,
		&beforeMembershipRow,
		&beforeScopeRow,
	); err != nil {
		t.Fatalf("capture governed history before repair: %v", err)
	}
	if _, err := pool.Exec(context.Background(), "DROP INDEX organization_service_provider_scope_applicability_idx; CREATE UNIQUE INDEX organization_service_provider_scope_applicability_idx ON organization_service_provider_scopes (organization_id, root_id, effective_from DESC, id DESC) WHERE status = 'ACTIVE'"); err != nil {
		t.Fatalf("replace derived index with wrong definition: %v", err)
	}
	if err := migrations.RepairRegulatoryChecklistGovernance(context.Background(), pool); err != nil {
		t.Fatalf("repair derived index: %v", err)
	}
	var indexDefinition string
	if err := pool.QueryRow(context.Background(), "SELECT pg_get_indexdef('organization_service_provider_scope_applicability_idx'::regclass)").Scan(&indexDefinition); err != nil || strings.Join(strings.Fields(indexDefinition), " ") != "CREATE INDEX organization_service_provider_scope_applicability_idx ON public.organization_service_provider_scopes USING btree (organization_id, root_id, effective_from DESC, id DESC)" {
		t.Fatalf("repair index definition = %q, err = %v", indexDefinition, err)
	}
	if err := migrations.RepairRegulatoryChecklistGovernance(context.Background(), pool); err != nil {
		t.Fatalf("idempotent repair: %v", err)
	}
	var afterSourceSnapshot, afterTemplateSnapshot, afterPackageSnapshot, afterPackageDigest string
	var afterReviewRow, afterMembershipRow, afterScopeRow string
	if err := pool.QueryRow(context.Background(), `
		SELECT source.snapshot::text, template.snapshot::text, package.snapshot::text, package.package_digest,
		       to_jsonb(review)::text, to_jsonb(membership)::text, to_jsonb(scope)::text
		FROM regulatory_reference_versions source
		JOIN checklist_template_versions template ON template.id = 'target-history-template-v1'
		JOIN inspection_packages package ON package.id = 'target-history-package-v1'
		JOIN review_decisions review ON review.id = 'target-history-review'
		JOIN caa_department_memberships membership ON membership.id = 'target-history-membership'
		JOIN organization_service_provider_scopes scope ON scope.id = 'target-scope-valid'
		WHERE source.id = 'target-history-source'
	`).Scan(
		&afterSourceSnapshot,
		&afterTemplateSnapshot,
		&afterPackageSnapshot,
		&afterPackageDigest,
		&afterReviewRow,
		&afterMembershipRow,
		&afterScopeRow,
	); err != nil {
		t.Fatalf("read governed history after repair: %v", err)
	}
	if beforeSourceSnapshot != afterSourceSnapshot || beforeTemplateSnapshot != afterTemplateSnapshot || beforePackageSnapshot != afterPackageSnapshot || beforePackageDigest != afterPackageDigest {
		t.Fatalf("repair changed governed history: before=%q/%q/%q/%q after=%q/%q/%q/%q", beforeSourceSnapshot, beforeTemplateSnapshot, beforePackageSnapshot, beforePackageDigest, afterSourceSnapshot, afterTemplateSnapshot, afterPackageSnapshot, afterPackageDigest)
	}
	if beforeReviewRow != afterReviewRow || beforeMembershipRow != afterMembershipRow || beforeScopeRow != afterScopeRow {
		t.Fatalf(
			"repair changed governed review, membership, or scope history: before=%q/%q/%q after=%q/%q/%q",
			beforeReviewRow,
			beforeMembershipRow,
			beforeScopeRow,
			afterReviewRow,
			afterMembershipRow,
			afterScopeRow,
		)
	}
	if version, err := migrations.CurrentVersion(context.Background(), pool); err != nil || version != migrations.LatestVersion {
		t.Fatalf("repair changed migration version: %d %v, want %d", version, err, migrations.LatestVersion)
	}
}

func TestTask2GuardedRollbackAllowsBaselineButRefusesAdoptedCatalogFacts(t *testing.T) {
	pool := createTestDatabase(t, "task2_adopted_rollback")
	// This guarded down-migration proof is intentionally a pre-v22 state. A
	// v22 database must not run a v21-only down migration out of order.
	applyMigrationFilesThrough(t, pool, 21)
	down, err := os.ReadFile(filepath.Join(apiModuleRoot(t), "migrations", "000021_regulatory_checklist_governance.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(down)); err != nil {
		t.Fatalf("pristine baseline rollback: %v", err)
	}
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("reapply: %v", err)
	}
	if _, err := pool.Exec(context.Background(), "INSERT INTO caa_departments (id,name,status,baseline_seed) VALUES ('FORGED_BASELINE','Forged Baseline','ACTIVE',true)"); err == nil {
		t.Fatal("caller forged a migration-owned baseline marker")
	}
	if _, err := pool.Exec(context.Background(), "INSERT INTO caa_departments (id,name,status) VALUES ('ADOPTED_DEPARTMENT','Adopted Department','ACTIVE')"); err != nil {
		t.Fatalf("adopt department: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(down)); err == nil {
		t.Fatal("adopted department did not refuse rollback")
	}
}

func TestTask2AuthenticatedAuditeeHTTPProjectionExcludesGovernedInternalFacts(t *testing.T) {
	pool := canonicalDatabase(t, "task2_auditee_http_projection")
	now := canonicalNow
	seedTask4Membership(t, pool, "task2-http-auditee", "task2-http-auditee-membership", 1, "ACTIVE", "airline-xyz", []string{"auditee"}, now)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO regulated_targets (id, target_kind, organization_id) VALUES
			('task2-http-target-other', 'ORGANIZATION', 'airline-other');
		INSERT INTO organization_service_provider_scopes (id, root_id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id) VALUES
			('task2-http-own-scope', 'task2-http-own-scope', 'airline-xyz', 'AIR_OPERATOR', 'AOC-HTTP-OWN', 'ACTIVE', '2025-01-01', 'target-airline-xyz'),
			('task2-http-other-scope', 'task2-http-other-scope', 'airline-other', 'CAMO', 'CAMO-HTTP-OTHER', 'ACTIVE', '2025-01-01', 'task2-http-target-other');
		INSERT INTO caa_department_memberships (id, root_id, subject_id, department_id, organizational_unit_id, membership_role, status, effective_from) VALUES
			('task2-http-internal-membership', 'task2-http-internal-membership', 'manager-001', 'AIRWORTHINESS_INSPECTORATE', 'AIRWORTHINESS_INSPECTORATE', 'DEPARTMENT_MANAGER', 'ACTIVE', '2025-01-01')
	`); err != nil {
		t.Fatalf("seed governed facts outside auditee projection: %v", err)
	}
	manager, err := session.NewManager(pool, []byte("0123456789abcdef0123456789abcdef"), session.ManagerDependencies{
		Clock: func() time.Time { return now },
		AuthorityObserver: &task4AuthorityObserver{observation: identity.AuthorityObservation{
			SubjectID: "task2-http-auditee", Enabled: true, OrganizationID: "airline-xyz", Roles: []identity.Role{identity.RoleAuditee}, ObservedAt: now,
		}},
	})
	if err != nil {
		t.Fatalf("new authenticated session manager: %v", err)
	}
	browserSession, err := manager.Create(context.Background(), session.CreateInput{
		SubjectID: "task2-http-auditee", Issuer: "https://identity.example/realms/avia", DisplayName: "Task 2 HTTP Auditee", OrganizationID: "airline-xyz", Roles: []identity.Role{identity.RoleAuditee},
	})
	if err != nil {
		t.Fatalf("create authenticated auditee session: %v", err)
	}
	handler := httpapi.NewAuthBoundary(nil, manager).Protect(httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{Pool: pool, Application: testService(pool), Clock: func() time.Time { return now }}).Handler())
	request := httptest.NewRequest(http.MethodGet, "/v1/auditee/coordination", nil)
	request.AddCookie(&http.Cookie{Name: httpapi.SessionCookieName, Value: browserSession.Token})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("authenticated auditee projection status=%d body=%s", response.Code, response.Body.String())
	}
	for _, forbidden := range []string{"task2-http-own-scope", "task2-http-other-scope", "CAMO-HTTP-OTHER", "task2-http-internal-membership", "departmentAssignments", "AIRWORTHINESS_INSPECTORATE", "airline-other"} {
		if strings.Contains(response.Body.String(), forbidden) {
			t.Fatalf("authenticated Auditee HTTP projection leaked %q: %s", forbidden, response.Body.String())
		}
	}
}
