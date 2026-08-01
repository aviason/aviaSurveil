//go:build canonicaltest

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

func TestGovernedChecklistIntakeMigration(t *testing.T) {
	if migrations.LatestVersion != 28 {
		t.Fatalf("latest migration = %d, want version 28", migrations.LatestVersion)
	}
	path := filepath.Join("..", "..", "migrations", "000028_governed_checklist_intake_and_authoring.up.sql")
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read version-28 migration: %v", err)
	}
	sql := string(sqlBytes)
	for _, table := range []string{
		"governed_reviewed_source_sets",
		"governed_checklist_functional_assignments",
		"checklist_import_batches",
		"checklist_import_files",
		"checklist_import_extraction_review_packets",
		"checklist_import_extraction_decision_sets",
		"existing_checklist_candidates",
		"existing_checklist_candidate_questions",
		"checklist_import_identity_resolutions",
		"checklist_import_object_intents",
		"checklist_import_attempts",
		"checklist_import_attempt_events",
		"regulatory_source_authority_attestations",
		"governed_candidate_source_chain_links",
		"governed_required_owner_resolution_facts",
		"governed_checklist_review_comments",
		"governed_reviewer_recommendation_dispositions",
		"governed_source_mapping_attestations",
	} {
		if !strings.Contains(sql, "CREATE TABLE "+table) {
			t.Errorf("version-28 migration must create %s", table)
		}
	}
	if strings.Contains(strings.ToLower(sql), "drop table") || strings.Contains(strings.ToLower(sql), "down migration") {
		t.Fatal("version-28 migration must be forward-only")
	}
	for _, token := range []string{"append_only_guard", "supersedes", "semantic_payload_digest", "idempotency_key", "fencing_token", "terminal_manifest_digest"} {
		if !strings.Contains(strings.ToLower(sql), token) {
			t.Errorf("version-28 migration is missing invariant token %s", token)
		}
	}
}
