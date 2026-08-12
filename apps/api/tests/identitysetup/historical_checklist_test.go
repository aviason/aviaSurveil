//go:build canonicaltest

package identitysetup_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/platform/database"
	"github.com/aviason/aviaSurveil/internal/testprofile"
	"github.com/aviason/aviaSurveil/migrations"
)

func TestGate0BootstrapHistoricalChecklistForFullProfile(t *testing.T) {
	// Production break: removing the one-shot internal fixture, changing its
	// exact template/question relationship, or recording approval history must
	// fail the full-profile preparation boundary.
	ctx := context.Background()
	pool, err := database.Open(ctx, requiredEnvironment(t, "AVIA_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("open full-profile preparation database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := migrations.Apply(ctx, pool); err != nil {
		t.Fatalf("apply full-profile preparation migrations: %v", err)
	}
	bootstrapTime := time.Date(2026, time.July, 25, 9, 0, 0, 0, time.UTC)
	if err := testprofile.BootstrapHistoricalFullProfileChecklist(ctx, pool, bootstrapTime); err != nil {
		t.Fatalf("bootstrap historical full-profile checklist: %v", err)
	}
	if err := testprofile.BootstrapHistoricalFullProfileChecklist(ctx, pool, bootstrapTime); err != nil {
		t.Fatalf("repeat identical historical full-profile checklist bootstrap: %v", err)
	}

	var templateID, title string
	var version int64
	var snapshot []byte
	var publishedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT template_id, version, title, snapshot, published_at
		FROM checklist_template_versions
		WHERE id = 'CTV-CABIN-1'
	`).Scan(&templateID, &version, &title, &snapshot, &publishedAt); err != nil {
		t.Fatalf("read historical checklist version: %v", err)
	}
	if templateID != "TPL-CABIN-2026" || version != 1 || title != "Cabin Inspection checklist" || !publishedAt.Equal(bootstrapTime) {
		t.Fatalf("historical checklist version = %q/%d/%q/%s", templateID, version, title, publishedAt)
	}
	var decodedSnapshot struct {
		CreatorSubjectID string `json:"creatorSubjectId"`
		ChangeReason     string `json:"changeReason"`
		Questions        []struct {
			ID string `json:"id"`
		} `json:"questions"`
	}
	if err := json.Unmarshal(snapshot, &decodedSnapshot); err != nil {
		t.Fatalf("decode historical checklist snapshot: %v", err)
	}
	if decodedSnapshot.CreatorSubjectID != "TEST-HISTORICAL-CHECKLIST-BOOTSTRAP" ||
		decodedSnapshot.ChangeReason != "Historical immutable full-profile checklist fixture." ||
		len(decodedSnapshot.Questions) != 6 {
		t.Fatalf("historical checklist snapshot = %#v", decodedSnapshot)
	}

	rows, err := pool.Query(ctx, `
		SELECT relation.question_version_id, version.question_id
		FROM template_version_questions relation
		JOIN question_versions version ON version.id = relation.question_version_id
		WHERE relation.template_version_id = 'CTV-CABIN-1'
		ORDER BY relation.position
	`)
	if err != nil {
		t.Fatalf("read historical checklist question relationship: %v", err)
	}
	defer rows.Close()
	var questionVersionIDs, questionIDs []string
	for rows.Next() {
		var questionVersionID, questionID string
		if err := rows.Scan(&questionVersionID, &questionID); err != nil {
			t.Fatalf("scan historical checklist question: %v", err)
		}
		questionVersionIDs = append(questionVersionIDs, questionVersionID)
		questionIDs = append(questionIDs, questionID)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate historical checklist questions: %v", err)
	}
	wantQuestionIDs := []string{
		"CAB-EMEQ-PBE-001", "CAB-LAV-001", "CAB-PAX-SEAT-001",
		"CAB-VID-CREW-SEAT-001", "CAB-GALLEY-001", "CAB-DOORS-001",
	}
	if len(questionIDs) != len(wantQuestionIDs) {
		t.Fatalf("historical checklist question count = %d, want %d", len(questionIDs), len(wantQuestionIDs))
	}
	for index, want := range wantQuestionIDs {
		if questionIDs[index] != want || questionVersionIDs[index] != "QV-"+want+"-V1" || decodedSnapshot.Questions[index].ID != want {
			t.Fatalf("historical checklist question %d = %q/%q/%q, want %q/%q/%q", index, questionVersionIDs[index], questionIDs[index], decodedSnapshot.Questions[index].ID, "QV-"+want+"-V1", want, want)
		}
	}

	var issuer, displayName, ownerRole, publishedTemplateVersionID string
	var revision int64
	if err := pool.QueryRow(ctx, `
		SELECT issuer, display_name FROM identity_references
		WHERE subject_id = 'TEST-HISTORICAL-CHECKLIST-BOOTSTRAP'
	`).Scan(&issuer, &displayName); err != nil {
		t.Fatalf("read historical bootstrap identity: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT owner_role, published_template_version_id, revision
		FROM template_masters WHERE id = 'TPL-CABIN-2026'
	`).Scan(&ownerRole, &publishedTemplateVersionID, &revision); err != nil {
		t.Fatalf("read historical checklist master: %v", err)
	}
	if issuer != "urn:avia:internal-testprofile" || displayName != "Historical checklist bootstrap" ||
		ownerRole != "Department Manager" || publishedTemplateVersionID != "CTV-CABIN-1" || revision != 1 {
		t.Fatalf("historical checklist identity/master = %q/%q/%q/%q/%d", issuer, displayName, ownerRole, publishedTemplateVersionID, revision)
	}

	var approvalEvents int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM audit_events
		WHERE entity_id = 'CTV-CABIN-1'
		  AND (action ILIKE '%approval%' OR action ILIKE '%technical%')
	`).Scan(&approvalEvents); err != nil {
		t.Fatalf("read historical checklist approval events: %v", err)
	}
	if approvalEvents != 0 {
		t.Fatalf("historical checklist synthesized %d approval events", approvalEvents)
	}
}
