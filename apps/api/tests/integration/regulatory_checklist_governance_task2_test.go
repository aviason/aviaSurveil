package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/organizations"
	"github.com/aviason/aviaSurveil/migrations"
)

func TestRegulatoryChecklistGovernanceSchemaInventory(t *testing.T) {
	pool := createTestDatabase(t, "regulatory_governance_schema")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	for _, table := range []string{
		"caa_departments",
		"caa_organizational_units",
		"caa_department_status_facts",
		"caa_organizational_unit_status_facts",
		"caa_department_memberships",
		"service_provider_types",
		"service_provider_topics",
		"service_provider_topic_links",
		"service_provider_unit_responsibilities",
		"regulated_targets",
		"organization_service_provider_scopes",
		"organization_service_provider_scope_targets",
	} {
		var relation *string
		if err := pool.QueryRow(context.Background(), "SELECT to_regclass($1)::text", "public."+table).Scan(&relation); err != nil {
			t.Fatalf("look up governed table %s: %v", table, err)
		}
		if relation == nil {
			t.Errorf("governed table %s is absent", table)
		}
	}

	var providerCount int
	if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM service_provider_types").Scan(&providerCount); err != nil {
		t.Fatalf("count seeded provider catalog: %v", err)
	}
	if providerCount != 20 {
		t.Fatalf("seeded provider catalog count = %d, want 20", providerCount)
	}
	var providerCodes []string
	if err := pool.QueryRow(context.Background(), "SELECT array_agg(id ORDER BY id) FROM service_provider_types").Scan(&providerCodes); err != nil {
		t.Fatalf("read exact provider codes: %v", err)
	}
	expectedProviderCodes := "AEMC,AERODROME_OPERATOR,AIR_OPERATOR,AIS_AIM_PROVIDER,AME,AMO,ANSP,ATO,AVSEC_PROVIDER,CAMO,CARGO_REGULATED_AGENT,CNS_PROVIDER,DOA,FSTD,FUEL_PROVIDER,GROUND_HANDLING,MET_PROVIDER,POA,RPAS_UAS_OPERATOR,SAR_ORGANIZATION"
	if strings.Join(providerCodes, ",") != expectedProviderCodes {
		t.Fatalf("provider catalog identities = %v", providerCodes)
	}
	catalogBytes, err := os.ReadFile(filepath.Join(
		apiModuleRoot(t), "..", "..", "docs", "regulatory-sources",
		"catalogs", "service-provider-catalog.v1.json",
	))
	if err != nil {
		t.Fatalf("read tracked provider catalog: %v", err)
	}
	var catalog struct {
		Providers []struct {
			Code                  string   `json:"code"`
			Label                 string   `json:"label"`
			RawOversightTopics    string   `json:"rawOversightTopics"`
			RawResponsibleCAAUnit string   `json:"rawResponsibleCaaUnit"`
			TargetKinds           []string `json:"targetKinds"`
			Responsibility        struct {
				NormalizationStatus string   `json:"normalizationStatus"`
				ApprovalRequired    []string `json:"approvalRequired"`
			} `json:"responsibility"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(catalogBytes, &catalog); err != nil {
		t.Fatalf("decode tracked provider catalog: %v", err)
	}
	for _, expected := range catalog.Providers {
		var label, rawTopics, rawUnit, normalizationStatus string
		var targetKinds, topics []string
		if err := pool.QueryRow(context.Background(), `
			SELECT provider.label, provider.raw_oversight_topics,
			       provider.raw_responsible_caa_unit, provider.target_kinds,
			       provider.normalization_status,
			       array_agg(topic.topic ORDER BY link.ordinal)
			FROM service_provider_types provider
			JOIN service_provider_topic_links link
			  ON link.service_provider_type_id = provider.id
			JOIN service_provider_topics topic
			  ON topic.id = link.service_provider_topic_id
			WHERE provider.id = $1
			GROUP BY provider.id
		`, expected.Code).Scan(
			&label, &rawTopics, &rawUnit, &targetKinds,
			&normalizationStatus, &topics,
		); err != nil {
			t.Fatalf("read seeded provider %s: %v", expected.Code, err)
		}
		expectedTopics := strings.Split(expected.RawOversightTopics, ";")
		for index := range expectedTopics {
			expectedTopics[index] = strings.TrimSpace(expectedTopics[index])
		}
		if label != expected.Label ||
			rawTopics != expected.RawOversightTopics ||
			rawUnit != expected.RawResponsibleCAAUnit ||
			normalizationStatus != expected.Responsibility.NormalizationStatus ||
			strings.Join(targetKinds, "\x00") != strings.Join(expected.TargetKinds, "\x00") ||
			strings.Join(topics, "\x00") != strings.Join(expectedTopics, "\x00") {
			t.Fatalf(
				"seeded provider %s diverges from tracked catalog: label=%q rawTopics=%q rawUnit=%q targetKinds=%v normalization=%q topics=%v",
				expected.Code, label, rawTopics, rawUnit, targetKinds,
				normalizationStatus, topics,
			)
		}
		var ownerCount int
		var ownerUnit, relationship string
		var approvalRequired bool
		err := pool.QueryRow(context.Background(), `
			SELECT COUNT(*), COALESCE(min(organizational_unit_id), ''),
			       COALESCE(min(relationship), ''),
			       COALESCE(bool_and(approval_required), false)
			FROM service_provider_unit_responsibilities
			WHERE service_provider_type_id = $1
		`, expected.Code).Scan(
			&ownerCount, &ownerUnit, &relationship, &approvalRequired,
		)
		if err != nil {
			t.Fatalf("read seeded responsibility %s: %v", expected.Code, err)
		}
		if expected.Responsibility.NormalizationStatus == "REVIEW_REQUIRED" {
			if ownerCount != 0 {
				t.Fatalf("%s acquired %d inferred responsibility owners", expected.Code, ownerCount)
			}
			continue
		}
		if ownerCount != 1 || len(expected.Responsibility.ApprovalRequired) != 1 ||
			ownerUnit != expected.Responsibility.ApprovalRequired[0] ||
			relationship != "PRIMARY" || !approvalRequired {
			t.Fatalf(
				"seeded responsibility %s = count:%d unit:%q relationship:%q approval:%t; catalog=%v",
				expected.Code, ownerCount, ownerUnit, relationship,
				approvalRequired, expected.Responsibility.ApprovalRequired,
			)
		}
	}
	for table, expectedForeignKeys := range map[string]int{
		"caa_organizational_units":                    1,
		"caa_department_status_facts":                 3,
		"caa_organizational_unit_status_facts":        3,
		"caa_department_memberships":                  6,
		"service_provider_topic_links":                2,
		"service_provider_unit_responsibilities":      2,
		"regulated_targets":                           3,
		"organization_service_provider_scopes":        5,
		"organization_service_provider_scope_targets": 2,
	} {
		var foreignKeys int
		if err := pool.QueryRow(context.Background(), `
			SELECT COUNT(*) FROM pg_constraint
			WHERE conrelid = $1::regclass AND contype = 'f'
		`, table).Scan(&foreignKeys); err != nil {
			t.Fatalf("inspect foreign keys for %s: %v", table, err)
		}
		if foreignKeys != expectedForeignKeys {
			t.Errorf("%s foreign key count = %d, want %d", table, foreignKeys, expectedForeignKeys)
		}
	}
	for index, requiredDefinition := range map[string]string{
		"caa_department_membership_root_identity_idx":           "UNIQUE INDEX caa_department_membership_root_identity_idx",
		"caa_department_memberships_effective_authority_idx":    "(subject_id, root_id, effective_from DESC, id DESC)",
		"organization_service_provider_scope_root_identity_idx": "UNIQUE INDEX organization_service_provider_scope_root_identity_idx",
		"organization_service_provider_scope_applicability_idx": "(organization_id, root_id, effective_from DESC, id DESC)",
	} {
		var definition string
		if err := pool.QueryRow(context.Background(), `
			SELECT pg_get_indexdef(indexrelid)
			FROM pg_index
			WHERE indexrelid = $1::regclass
		`, index).Scan(&definition); err != nil {
			t.Fatalf("look up governed index %s: %v", index, err)
		}
		if !strings.Contains(definition, requiredDefinition) {
			t.Errorf("governed index %s definition = %q, missing %q", index, definition, requiredDefinition)
		}
	}
	for index, expectedDefinition := range map[string]string{
		"caa_department_status_fact_root_identity_idx":          "CREATE UNIQUE INDEX caa_department_status_fact_root_identity_idx ON public.caa_department_status_facts USING btree (department_id) WHERE (supersedes_id IS NULL)",
		"caa_organizational_unit_status_fact_root_identity_idx": "CREATE UNIQUE INDEX caa_organizational_unit_status_fact_root_identity_idx ON public.caa_organizational_unit_status_facts USING btree (organizational_unit_id) WHERE (supersedes_id IS NULL)",
		"caa_department_status_facts_effective_idx":             "CREATE INDEX caa_department_status_facts_effective_idx ON public.caa_department_status_facts USING btree (department_id, root_id, effective_from DESC, id DESC)",
		"caa_organizational_unit_status_facts_effective_idx":    "CREATE INDEX caa_organizational_unit_status_facts_effective_idx ON public.caa_organizational_unit_status_facts USING btree (organizational_unit_id, root_id, effective_from DESC, id DESC)",
	} {
		var definition string
		if err := pool.QueryRow(context.Background(), `
			SELECT pg_get_indexdef(indexrelid)
			FROM pg_index
			WHERE indexrelid = $1::regclass
		`, index).Scan(&definition); err != nil {
			t.Fatalf("look up governed status index %s: %v", index, err)
		}
		if normalized := strings.Join(strings.Fields(definition), " "); normalized != expectedDefinition {
			t.Errorf("governed status index %s definition = %q, want %q", index, normalized, expectedDefinition)
		}
	}
	type triggerExpectation struct {
		relation, function string
		eventMask          int
	}
	for trigger, expected := range map[string]triggerExpectation{
		"caa_departments_append_only": {"caa_departments", "governed_append_only_guard", 24}, "caa_departments_baseline_seed_guard": {"caa_departments", "governed_baseline_seed_guard", 4},
		"caa_organizational_units_append_only": {"caa_organizational_units", "governed_append_only_guard", 24}, "caa_organizational_units_baseline_seed_guard": {"caa_organizational_units", "governed_baseline_seed_guard", 4},
		"caa_department_status_successor_guard": {"caa_department_status_facts", "governed_status_successor_guard", 4}, "caa_department_status_facts_append_only": {"caa_department_status_facts", "governed_append_only_guard", 24}, "caa_department_status_baseline_seed_guard": {"caa_department_status_facts", "governed_baseline_seed_guard", 4},
		"caa_organizational_unit_status_successor_guard": {"caa_organizational_unit_status_facts", "governed_status_successor_guard", 4}, "caa_organizational_unit_status_facts_append_only": {"caa_organizational_unit_status_facts", "governed_append_only_guard", 24}, "caa_organizational_unit_status_baseline_seed_guard": {"caa_organizational_unit_status_facts", "governed_baseline_seed_guard", 4},
		"caa_department_memberships_append_only": {"caa_department_memberships", "governed_append_only_guard", 24}, "department_membership_successor_guard": {"caa_department_memberships", "governed_successor_guard", 4},
		"service_provider_types_append_only": {"service_provider_types", "governed_append_only_guard", 24}, "service_provider_types_baseline_seed_guard": {"service_provider_types", "governed_baseline_seed_guard", 4},
		"service_provider_topics_append_only": {"service_provider_topics", "governed_append_only_guard", 24}, "service_provider_topics_baseline_seed_guard": {"service_provider_topics", "governed_baseline_seed_guard", 4},
		"service_provider_topic_links_append_only": {"service_provider_topic_links", "governed_append_only_guard", 24}, "service_provider_topic_links_baseline_seed_guard": {"service_provider_topic_links", "governed_baseline_seed_guard", 4},
		"service_provider_unit_responsibilities_append_only": {"service_provider_unit_responsibilities", "governed_append_only_guard", 24}, "service_provider_unit_responsibilities_baseline_seed_guard": {"service_provider_unit_responsibilities", "governed_baseline_seed_guard", 4},
		"organization_service_provider_scopes_append_only": {"organization_service_provider_scopes", "governed_append_only_guard", 24}, "organization_scope_successor_guard": {"organization_service_provider_scopes", "governed_successor_guard", 4}, "organization_service_provider_scope_primary_target_kind_guard": {"organization_service_provider_scopes", "validate_governed_scope_primary_target", 20},
		"organization_service_provider_scope_targets_append_only": {"organization_service_provider_scope_targets", "governed_append_only_guard", 24}, "organization_service_provider_scope_target_kind_guard": {"organization_service_provider_scope_targets", "validate_governed_scope_target", 4}, "regulated_targets_append_only": {"regulated_targets", "governed_append_only_guard", 24},
	} {
		var relation, function string
		var eventMask int
		var before bool
		if err := pool.QueryRow(context.Background(), `SELECT relation.relname, function.proname, trigger.tgtype::int & 60, (trigger.tgtype::int & 2) = 2 FROM pg_trigger trigger JOIN pg_class relation ON relation.oid = trigger.tgrelid JOIN pg_proc function ON function.oid = trigger.tgfoid WHERE trigger.tgname = $1 AND NOT trigger.tgisinternal`, trigger).Scan(&relation, &function, &eventMask, &before); err != nil {
			t.Fatalf("inspect governed trigger %s: %v", trigger, err)
		}
		if relation != expected.relation || function != expected.function || eventMask != expected.eventMask || !before {
			t.Errorf("governed trigger %s = relation=%q function=%q before=%t eventMask=%d; want relation=%q function=%q before=true eventMask=%d", trigger, relation, function, before, eventMask, expected.relation, expected.function, expected.eventMask)
		}
	}
	for table, requiredDefinitions := range map[string][]string{
		"caa_department_memberships": {
			"FOREIGN KEY (subject_id) REFERENCES identity_references(subject_id)",
			"FOREIGN KEY (department_id) REFERENCES caa_departments(id)",
			"FOREIGN KEY (organizational_unit_id) REFERENCES caa_organizational_units(id)",
			"FOREIGN KEY (department_id, organizational_unit_id) REFERENCES caa_organizational_units(department_id, id)",
			"FOREIGN KEY (root_id) REFERENCES caa_department_memberships(id)",
			"FOREIGN KEY (supersedes_id) REFERENCES caa_department_memberships(id)",
			"CHECK (((effective_to IS NULL) OR (effective_to > effective_from)))",
			"UNIQUE (subject_id, department_id, organizational_unit_id, effective_from)",
			"UNIQUE (supersedes_id)",
		},
		"caa_department_status_facts": {
			"FOREIGN KEY (department_id) REFERENCES caa_departments(id)", "FOREIGN KEY (root_id) REFERENCES caa_department_status_facts(id)", "FOREIGN KEY (supersedes_id) REFERENCES caa_department_status_facts(id)", "CHECK ((status = ANY (ARRAY['ACTIVE'::text, 'INACTIVE'::text]))", "UNIQUE (supersedes_id)",
		},
		"caa_organizational_unit_status_facts": {
			"FOREIGN KEY (organizational_unit_id) REFERENCES caa_organizational_units(id)", "FOREIGN KEY (root_id) REFERENCES caa_organizational_unit_status_facts(id)", "FOREIGN KEY (supersedes_id) REFERENCES caa_organizational_unit_status_facts(id)", "CHECK ((status = ANY (ARRAY['ACTIVE'::text, 'INACTIVE'::text]))", "UNIQUE (supersedes_id)",
		},
		"regulated_targets": {
			"FOREIGN KEY (organization_id) REFERENCES organizations(id)",
			"FOREIGN KEY (person_subject_id) REFERENCES identity_references(subject_id)",
			"FOREIGN KEY (owner_organization_id) REFERENCES organizations(id)",
			"CHECK ((((target_kind = 'ORGANIZATION'::text)",
			"UNIQUE (target_kind, external_identifier)",
		},
		"organization_service_provider_scopes": {
			"FOREIGN KEY (organization_id) REFERENCES organizations(id)",
			"FOREIGN KEY (service_provider_type_id) REFERENCES service_provider_types(id)",
			"FOREIGN KEY (primary_target_id) REFERENCES regulated_targets(id)",
			"FOREIGN KEY (root_id) REFERENCES organization_service_provider_scopes(id)",
			"FOREIGN KEY (supersedes_id) REFERENCES organization_service_provider_scopes(id)",
			"CHECK (((effective_to IS NULL) OR (effective_to > effective_from)))",
			"UNIQUE (organization_id, service_provider_type_id, authorization_identifier, effective_from)",
			"UNIQUE (supersedes_id)",
		},
	} {
		var definitions string
		if err := pool.QueryRow(context.Background(), `
			SELECT COALESCE(string_agg(pg_get_constraintdef(oid), E'\n' ORDER BY conname), '')
			FROM pg_constraint WHERE conrelid = $1::regclass
		`, table).Scan(&definitions); err != nil {
			t.Fatalf("read semantic constraints for %s: %v", table, err)
		}
		for _, required := range requiredDefinitions {
			if !strings.Contains(definitions, required) {
				t.Errorf("%s lacks required constraint %q; definitions:\n%s", table, required, definitions)
			}
		}
	}
}

func TestOrganizationServiceProviderScopesAllowMultipleActiveScopesAndExcludeExpiredApplicability(t *testing.T) {
	pool := createTestDatabase(t, "regulatory_governance_scopes")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO organizations (id, legal_name, organization_type, status)
		VALUES ('operator-regulatory', 'Regulatory Operator', 'OPERATOR', 'ACTIVE');
		INSERT INTO regulated_targets (id, target_kind, organization_id)
		VALUES ('target-operator-regulatory', 'ORGANIZATION', 'operator-regulatory');
		INSERT INTO organization_service_provider_scopes (
			id, organization_id, service_provider_type_id, authorization_identifier,
			status, effective_from, primary_target_id, operation_qualifiers, activity_qualifiers
		) VALUES
			('scope-air', 'operator-regulatory', 'AIR_OPERATOR', 'AOC-001', 'ACTIVE', '2025-01-01', 'target-operator-regulatory', '{"operation":"commercial"}', '{}'),
			('scope-camo', 'operator-regulatory', 'CAMO', 'CAMO-001', 'ACTIVE', '2025-01-01', 'target-operator-regulatory', '{}', '{}'),
			('scope-ato', 'operator-regulatory', 'ATO', 'ATO-001', 'ACTIVE', '2025-01-01', 'target-operator-regulatory', '{}', '{}');
	`); err != nil {
		t.Fatalf("insert concurrent active scopes: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE organization_service_provider_scopes SET effective_to = '2027-01-01' WHERE id = 'scope-air'
	`); err == nil {
		t.Fatal("governed scopes were mutable; effective-period corrections must be represented by a new fact")
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO organization_service_provider_scopes (
			id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, effective_to, primary_target_id
		) VALUES ('scope-expired', 'operator-regulatory', 'AMO', 'AMO-EXPIRED', 'ACTIVE', '2020-01-01', '2025-01-01', 'target-operator-regulatory')
	`); err != nil {
		t.Fatalf("insert expired scope: %v", err)
	}
	applicable, err := organizations.ListApplicableServiceProviderScopes(context.Background(), pool, "operator-regulatory", time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("resolve applicable scopes: %v", err)
	}
	if len(applicable) != 3 {
		t.Fatalf("applicable scope count = %d, want Air Operator + CAMO + ATO only", len(applicable))
	}
}

func TestRegulatedTargetsRetainTypedIdentityAndCompatibility(t *testing.T) {
	pool := createTestDatabase(t, "regulatory_governance_targets")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO organizations (id, legal_name, organization_type, status)
		VALUES ('operator-targets', 'Target Operator', 'OPERATOR', 'ACTIVE');
		INSERT INTO identity_references (subject_id, issuer, display_name)
		VALUES ('ame-person', 'test', 'AME Person');
		INSERT INTO regulated_targets (id, target_kind, owner_organization_id, external_identifier) VALUES
			('target-fstd-device', 'DEVICE', 'operator-targets', 'FSTD-SIM-001'),
			('target-facility', 'FACILITY', 'operator-targets', 'FAC-001'),
			('target-system', 'SYSTEM', 'operator-targets', 'SYS-001'),
			('target-asset', 'ASSET', 'operator-targets', 'AST-001'),
			('target-location', 'LOCATION', 'operator-targets', 'LOC-001');
		INSERT INTO regulated_targets (id, target_kind, person_subject_id, owner_organization_id)
		VALUES ('target-ame-person', 'PERSON', 'ame-person', 'operator-targets');
		INSERT INTO organization_service_provider_scopes (id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id) VALUES
			('scope-fstd', 'operator-targets', 'FSTD', 'FSTD-001', 'ACTIVE', '2025-01-01', 'target-fstd-device'),
			('scope-ame', 'operator-targets', 'AME', 'AME-001', 'ACTIVE', '2025-01-01', 'target-ame-person');
	`); err != nil {
		t.Fatalf("seed typed targets: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO organization_service_provider_scope_targets (organization_service_provider_scope_id, regulated_target_id)
		VALUES ('scope-fstd', 'target-facility')
	`); err != nil {
		t.Fatalf("FSTD rejected its facility target: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO organization_service_provider_scope_targets (organization_service_provider_scope_id, regulated_target_id)
		VALUES ('scope-fstd', 'target-system')
	`); err == nil {
		t.Fatal("FSTD accepted incompatible system scope target")
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO regulated_targets (id, target_kind, organization_id) VALUES ('target-organization', 'ORGANIZATION', 'operator-targets');
		INSERT INTO organization_service_provider_scopes (id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id)
		VALUES ('scope-fstd-invalid-primary', 'operator-targets', 'FSTD', 'FSTD-INVALID', 'ACTIVE', '2025-01-01', 'target-organization');
	`); err == nil {
		t.Fatal("FSTD scope accepted an organization primary target at insert")
	}
	var organizationID *string
	var personID *string
	if err := pool.QueryRow(context.Background(), `
		SELECT organization_id, person_subject_id FROM regulated_targets WHERE id = 'target-fstd-device'
	`).Scan(&organizationID, &personID); err != nil {
		t.Fatalf("read FSTD target identity: %v", err)
	}
	if organizationID != nil || personID != nil {
		t.Fatalf("FSTD device was coerced into organization/person identity: organization=%v person=%v", organizationID, personID)
	}
}

func TestGovernedCatalogFactsRemainAppendOnlyAndAmbiguousOwnersHaveNoApprovalOwner(t *testing.T) {
	pool := createTestDatabase(t, "regulatory_governance_catalog")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	var ambiguousOwners int
	if err := pool.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM service_provider_unit_responsibilities
		WHERE service_provider_type_id IN ('AIS_AIM_PROVIDER', 'GROUND_HANDLING', 'CARGO_REGULATED_AGENT', 'RPAS_UAS_OPERATOR')
	`).Scan(&ambiguousOwners); err != nil {
		t.Fatalf("count ambiguous owner links: %v", err)
	}
	if ambiguousOwners != 0 {
		t.Fatalf("ambiguous provider rows acquired %d inferred approval owner links", ambiguousOwners)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE service_provider_types SET label = 'changed' WHERE id = 'AIR_OPERATOR'"); err == nil {
		t.Fatal("catalog mutation succeeded; governed catalog facts must be append-only")
	}
}

func TestDepartmentMembershipsResolveEffectiveManagerAuthorityAndFailClosed(t *testing.T) {
	pool := createTestDatabase(t, "regulatory_governance_department_authority")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (subject_id, issuer, display_name) VALUES
			('manager-ops', 'test', 'OPS Manager'), ('manager-air', 'test', 'AIR Manager'), ('manager-unassigned', 'test', 'Unassigned Manager');
		INSERT INTO caa_department_memberships (id, subject_id, department_id, organizational_unit_id, membership_role, effective_from, effective_to, status) VALUES
			('membership-ops-current', 'manager-ops', 'FLIGHT_OPERATIONS_INSPECTORATE', 'FLIGHT_OPERATIONS_INSPECTORATE', 'DEPARTMENT_MANAGER', '2025-01-01', NULL, 'ACTIVE'),
			('membership-air-expired', 'manager-air', 'AIRWORTHINESS_INSPECTORATE', 'AIRWORTHINESS_INSPECTORATE', 'DEPARTMENT_MANAGER', '2024-01-01', '2025-01-01', 'ACTIVE');
	`); err != nil {
		t.Fatalf("seed department memberships: %v", err)
	}
	at := time.Date(2026, time.July, 29, 0, 0, 0, 0, time.UTC)
	assignments, err := identity.ResolveEffectiveDepartmentAssignments(context.Background(), pool, "manager-ops", at)
	if err != nil {
		t.Fatalf("resolve current department assignment: %v", err)
	}
	opsManager := identity.Principal{SubjectID: "manager-ops", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleDepartmentManager}, DepartmentAssignments: assignments}
	if !identity.CanTechnicallyReview(opsManager, "FLIGHT_OPERATIONS_INSPECTORATE") {
		t.Fatal("current responsible OPS Department Manager was denied")
	}
	if !identity.CanTechnicallyReviewUnit(opsManager, "FLIGHT_OPERATIONS_INSPECTORATE", "FLIGHT_OPERATIONS_INSPECTORATE") {
		t.Fatal("current manager assignment did not retain its organizational unit")
	}
	if identity.CanTechnicallyReviewUnit(opsManager, "FLIGHT_OPERATIONS_INSPECTORATE", "AIRWORTHINESS_INSPECTORATE") {
		t.Fatal("department assignment leaked into another organizational unit")
	}
	if identity.CanTechnicallyReview(opsManager, "AIRWORTHINESS_INSPECTORATE") {
		t.Fatal("OPS Department Manager acquired AIR authority")
	}
	if identity.CanTechnicallyReview(identity.Principal{SubjectID: "manager-unassigned", OrganizationID: "CAA", Roles: []identity.Role{identity.RoleDepartmentManager}}, "FLIGHT_OPERATIONS_INSPECTORATE") {
		t.Fatal("generic manager role acquired technical authority without a current assignment")
	}
	expired, err := identity.ResolveEffectiveDepartmentAssignments(context.Background(), pool, "manager-air", at)
	if err != nil {
		t.Fatalf("resolve expired department assignment: %v", err)
	}
	if len(expired) != 0 {
		t.Fatalf("expired department membership resolved as current: %+v", expired)
	}
}

func TestGovernedScopesAndMembershipsDoNotLeakIntoAuditeeWorkspace(t *testing.T) {
	pool := canonicalDatabase(t, "regulatory_governance_auditee")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO caa_department_memberships (id, subject_id, department_id, organizational_unit_id, membership_role, effective_from, status)
		VALUES ('membership-internal', 'manager-001', 'AIRWORTHINESS_INSPECTORATE', 'AIRWORTHINESS_INSPECTORATE', 'DEPARTMENT_MANAGER', '2025-01-01', 'ACTIVE');
		INSERT INTO regulated_targets (id, target_kind, organization_id) VALUES
			('target-other-scope', 'ORGANIZATION', 'airline-other');
		INSERT INTO organization_service_provider_scopes (id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id) VALUES
			('scope-auditee', 'airline-xyz', 'AIR_OPERATOR', 'AOC-AUDITEE', 'ACTIVE', '2025-01-01', 'target-airline-xyz'),
			('scope-other', 'airline-other', 'CAMO', 'CAMO-OTHER', 'ACTIVE', '2025-01-01', 'target-other-scope');
	`); err != nil {
		t.Fatalf("seed governed internal records: %v", err)
	}
	workspace, err := testService(pool).GetAuditeeWorkspace(context.Background(), identity.Principal{
		SubjectID: "auditee-xyz", OrganizationID: "airline-xyz", Roles: []identity.Role{identity.RoleAuditee},
	})
	if err != nil {
		t.Fatalf("get auditee workspace: %v", err)
	}
	encoded, err := json.Marshal(workspace)
	if err != nil {
		t.Fatalf("marshal auditee workspace: %v", err)
	}
	for _, forbidden := range []string{"scope-auditee", "scope-other", "membership-internal", "AIRWORTHINESS_INSPECTORATE"} {
		if string(encoded) != "" && contains(string(encoded), forbidden) {
			t.Fatalf("Auditee workspace leaked internal governed fact %q: %s", forbidden, encoded)
		}
	}
}

func TestGovernedChecklistMigrationRecovery(t *testing.T) {
	pool := createTestDatabase(t, "regulatory_governance_recovery")
	// Exercise the guarded v21 rollback from its actual historical endpoint.
	// Task 4 separately proves the normal v21-to-v22 forward upgrade without a
	// history rewrite; applying a v21 down migration after v22 would be invalid.
	applyMigrationFilesThrough(t, pool, 21)
	if version, err := migrations.CurrentVersion(context.Background(), pool); err != nil || version != 21 {
		t.Fatalf("historical v21 version = %d, err = %v, want 21", version, err)
	}
	down, err := os.ReadFile(filepath.Join(apiModuleRoot(t), "migrations", "000021_regulatory_checklist_governance.down.sql"))
	if err != nil {
		t.Fatalf("read guarded down migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(down)); err != nil {
		t.Fatalf("rollback before governed records: %v", err)
	}
	if version, err := migrations.CurrentVersion(context.Background(), pool); err != nil || version != 20 {
		t.Fatalf("rollback version = %d, err = %v, want 20", version, err)
	}
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("forward repair from guarded rollback: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (subject_id, issuer, display_name) VALUES ('source-owner-recovery', 'test', 'Recovery Source Owner');
		INSERT INTO organizations (id, legal_name, organization_type, status) VALUES ('operator-recovery', 'Recovery Operator', 'OPERATOR', 'ACTIVE');
		INSERT INTO inspections (id, organization_id, assigned_inspector_subject_id, title, inspection_type, status, revision)
		VALUES ('inspection-recovery', 'operator-recovery', 'source-owner-recovery', 'Recovery audit', 'RAMP', 'IN_PROGRESS', 1);
		INSERT INTO regulatory_reference_versions (id, reference_id, version, title, status, effective_date, snapshot)
		VALUES ('source-recovery', 'source-recovery', 1, 'Recovery source', 'ACTIVE', '2025-01-01', '{"digest":"sha256:source-recovery"}');
		INSERT INTO review_decisions (id, entity_type, entity_id, expected_revision, decision, reason, decided_by_subject_id, decided_at)
		VALUES ('review-recovery', 'regulatory_reference_version', 'source-recovery', 1, 'ACKNOWLEDGED', 'Representative existing review history.', 'source-owner-recovery', now());
		INSERT INTO checklist_template_versions (id, template_id, version, title, snapshot, published_at)
		VALUES ('template-recovery-v1', 'template-recovery', 1, 'Existing published template', '{"digest":"sha256:template-recovery"}', now());
		INSERT INTO inspection_packages (id, inspection_id, checklist_template_version_id, package_version, snapshot, expires_at, package_digest)
		VALUES ('package-recovery-v1', 'inspection-recovery', 'template-recovery-v1', 1, '{"digest":"sha256:package-recovery"}', now() + interval '1 day', 'sha256:package-recovery');
		INSERT INTO regulated_targets (id, target_kind, organization_id) VALUES ('target-recovery', 'ORGANIZATION', 'operator-recovery');
		INSERT INTO organization_service_provider_scopes (id, organization_id, service_provider_type_id, authorization_identifier, status, effective_from, primary_target_id)
		VALUES ('scope-recovery', 'operator-recovery', 'AIR_OPERATOR', 'AOC-RECOVERY', 'ACTIVE', '2025-01-01', 'target-recovery');
	`); err != nil {
		t.Fatalf("seed accepted governed scope: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(down)); err == nil {
		t.Fatal("rollback after governed records succeeded")
	}
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("forward repair after refused rollback: %v", err)
	}
	for table, id := range map[string]string{
		"organization_service_provider_scopes": "scope-recovery",
		"regulatory_reference_versions":        "source-recovery",
		"review_decisions":                     "review-recovery",
		"checklist_template_versions":          "template-recovery-v1",
		"inspection_packages":                  "package-recovery-v1",
	} {
		var preserved int
		if err := pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM "+table+" WHERE id = $1", id).Scan(&preserved); err != nil {
			t.Fatalf("read repaired %s history: %v", table, err)
		}
		if preserved != 1 {
			t.Fatalf("forward repair lost %s history count = %d", table, preserved)
		}
	}
}

func contains(value, part string) bool {
	return len(part) == 0 || (len(value) >= len(part) && containsAt(value, part))
}

func containsAt(value, part string) bool {
	for index := 0; index+len(part) <= len(value); index++ {
		if value[index:index+len(part)] == part {
			return true
		}
	}
	return false
}
