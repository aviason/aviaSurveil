package integration_test

import (
	"context"
	"testing"

	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
)

func TestCanonicalTestProfileResetSeedsExactServerOwnedScope(t *testing.T) {
	pool := createTestDatabase(t, "canonical_http_profile")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	for iteration := 0; iteration < 2; iteration++ {
		if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
			t.Fatalf("reset canonical profile iteration %d: %v", iteration, err)
		}
		var audits, questions, reports, preliminaryReports, otherFindings, offlineGrants int
		var invalidReportContentHashes int
		var preliminaryV1Ready bool
		var auditDueDate string
		var nextPlannedAuditDate string
		if err := pool.QueryRow(context.Background(), `
			SELECT
				(SELECT count(*) FROM inspections WHERE id = 'AUD-2026-001'),
				(SELECT count(*) FROM inspection_question_assignments WHERE inspection_id = 'AUD-2026-001'),
				(SELECT count(*) FROM report_versions WHERE id = 'RPT-CAB-2026-001-V1'),
				(SELECT count(*) FROM report_versions
				 WHERE (id = 'PR-2026-018-V0' AND version = 0)
				    OR (id = 'PR-2026-018-V1' AND version = 1)),
				(SELECT count(*) FROM findings WHERE organization_id = 'ORG-SKYCARGO'),
				(SELECT count(*) FROM offline_grants),
				(SELECT count(*) FROM report_versions
				 WHERE snapshot->>'contentHash' !~ '^sha256:[0-9a-f]{64}$'),
				(SELECT (snapshot->>'ready')::boolean
				 FROM report_versions WHERE id = 'PR-2026-018-V1'),
				(SELECT due_date::text FROM inspections WHERE id = 'AUD-2026-001'),
				(SELECT scheduled_date::text FROM surveillance_plan_items
				 WHERE id = 'PLAN-2026-CAB-001')
		`).Scan(
			&audits,
			&questions,
			&reports,
			&preliminaryReports,
			&otherFindings,
			&offlineGrants,
			&invalidReportContentHashes,
			&preliminaryV1Ready,
			&auditDueDate,
			&nextPlannedAuditDate,
		); err != nil {
			t.Fatalf("read canonical profile counts: %v", err)
		}
		if audits != 1 || questions != 6 || reports != 1 || preliminaryReports != 2 || otherFindings != 1 {
			t.Fatalf(
				"canonical counts = audits %d, questions %d, reports %d, preliminary Reports %d, other Findings %d",
				audits, questions, reports, preliminaryReports, otherFindings,
			)
		}
		if !preliminaryV1Ready {
			t.Fatal("canonical Preliminary Report V1 is not ready for its seeded Department review state")
		}
		if offlineGrants != 0 {
			t.Fatalf("canonical reset seeded %d OfflineGrant rows; checkout must create the first grant", offlineGrants)
		}
		if invalidReportContentHashes != 0 {
			t.Fatalf(
				"canonical reset seeded %d report versions without exact sha256 content hashes",
				invalidReportContentHashes,
			)
		}
		if auditDueDate != "2026-06-18" {
			t.Fatalf("canonical Audit due date = %q, want %q", auditDueDate, "2026-06-18")
		}
		if nextPlannedAuditDate != "2026-07-15" {
			t.Fatalf(
				"canonical next planned Audit date = %q, want %q",
				nextPlannedAuditDate,
				"2026-07-15",
			)
		}
	}
	principal, ok := testprofile.Principal("USR-AUDITEE-FLY")
	if !ok || principal.OrganizationID != "ORG-FLY-NAMIBIA" || len(principal.Roles) != 1 {
		t.Fatalf("server-owned Auditee principal = %+v, ok = %v", principal, ok)
	}
	if _, ok := testprofile.Principal("attacker-controlled-subject"); ok {
		t.Fatal("unknown test subject was accepted")
	}
}

func TestCanonicalGeneratorAllocatesUniqueFindingIdentityPairsAndResets(t *testing.T) {
	generator := testprofile.NewGenerator()
	assertPair := func(wantID, wantReference string) {
		t.Helper()
		if got := generator.Next("finding"); got != wantID {
			t.Fatalf("Finding ID = %q, want %q", got, wantID)
		}
		if got := generator.FindingReference(); got != wantReference {
			t.Fatalf("Finding reference = %q, want %q", got, wantReference)
		}
	}

	assertPair("FND-CAB-2026-001", "CAB-2026-001")
	assertPair("FND-CAB-2026-002", "CAB-2026-002")
	generator.Reset()
	assertPair("FND-CAB-2026-001", "CAB-2026-001")
}

func TestCanonicalGeneratorUsesContractEvidenceIdentityAndResets(t *testing.T) {
	generator := testprofile.NewGenerator()
	assertEvidence := func(wantEvidenceID, wantVersionID string) {
		t.Helper()
		if got := generator.Next("evidence"); got != wantEvidenceID {
			t.Fatalf("Evidence ID = %q, want %q", got, wantEvidenceID)
		}
		if got := generator.Next("evidence-version"); got != wantVersionID {
			t.Fatalf("Evidence version ID = %q, want %q", got, wantVersionID)
		}
	}

	assertEvidence("EV-CAB-2026-001", "EV-CAB-2026-001-V1")
	assertEvidence("EV-CAB-2026-001", "EV-CAB-2026-001-V2")
	generator.Reset()
	assertEvidence("EV-CAB-2026-001", "EV-CAB-2026-001-V1")
}

func TestCanonicalGeneratorUsesContractOfflineGrantIdentityAndResets(t *testing.T) {
	generator := testprofile.NewGenerator()
	if got := generator.Next("grant"); got != "GRANT-CANDIDATE-001" {
		t.Fatalf("OfflineGrant ID = %q, want %q", got, "GRANT-CANDIDATE-001")
	}
	if got := generator.Next("grant"); got != "GRANT-CANDIDATE-002" {
		t.Fatalf("second OfflineGrant ID = %q, want %q", got, "GRANT-CANDIDATE-002")
	}
	generator.Reset()
	if got := generator.Next("grant"); got != "GRANT-CANDIDATE-001" {
		t.Fatalf("OfflineGrant ID after reset = %q, want %q", got, "GRANT-CANDIDATE-001")
	}
}
