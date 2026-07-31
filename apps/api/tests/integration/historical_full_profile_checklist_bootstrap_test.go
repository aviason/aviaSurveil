package integration_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
	"github.com/MarlonJD/aviaSurveil360/apps/api/migrations"
)

var historicalFullProfileBootstrapTime = time.Date(2026, time.July, 25, 9, 0, 0, 0, time.UTC)

func TestHistoricalFullProfileChecklistBootstrapPreservesExactIdentityAndFailsClosed(t *testing.T) {
	t.Run("identical bootstrap is idempotent and preserves question-version identities", func(t *testing.T) {
		pool := historicalFullProfileBootstrapDatabase(t)
		bootstrapHistoricalFullProfileChecklist(t, pool)
		bootstrapHistoricalFullProfileChecklist(t, pool)

		rows, err := pool.Query(context.Background(), `
			SELECT relation.question_version_id, version.question_id
			FROM template_version_questions relation
			JOIN question_versions version ON version.id = relation.question_version_id
			WHERE relation.template_version_id = 'CTV-CABIN-1'
			ORDER BY relation.position
		`)
		if err != nil {
			t.Fatalf("read historical checklist identities: %v", err)
		}
		defer rows.Close()
		for index, questionID := range historicalFullProfileQuestionIDs {
			if !rows.Next() {
				t.Fatalf("historical question identity %d is missing", index)
			}
			var questionVersionID, actualQuestionID string
			if err := rows.Scan(&questionVersionID, &actualQuestionID); err != nil {
				t.Fatalf("scan historical question identity: %v", err)
			}
			if actualQuestionID != questionID || questionVersionID != "QV-"+questionID+"-V1" {
				t.Fatalf("historical question identity = %q/%q, want %q/%q", questionVersionID, actualQuestionID, "QV-"+questionID+"-V1", questionID)
			}
		}
		if rows.Next() {
			t.Fatal("historical checklist has an unexpected extra question identity")
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("iterate historical question identities: %v", err)
		}
	})

	for name, testCase := range map[string]struct {
		seed              func(*testing.T, *database.Pool)
		expected          string
		expectedRelations int
	}{
		"bootstrap identity": {
			seed: seedConflictingHistoricalBootstrapIdentity, expected: "historical bootstrap identity conflict",
		},
		"template snapshot title and publication time": {
			seed: seedConflictingHistoricalTemplate, expected: "historical checklist conflict",
		},
		"question content and creator": {
			seed: seedConflictingHistoricalQuestion, expected: "historical question conflict",
		},
		"template master owner and revision": {
			seed: seedConflictingHistoricalTemplateMasterOwnerAndRevision, expected: "historical template master conflict", expectedRelations: 6,
		},
		"template master published-version pointer": {
			seed: seedConflictingHistoricalTemplateMasterPointer, expected: "historical template master conflict", expectedRelations: 6,
		},
	} {
		t.Run("rejects conflicting "+name, func(t *testing.T) {
			pool := historicalFullProfileBootstrapDatabase(t)
			testCase.seed(t, pool)
			err := testprofile.BootstrapHistoricalFullProfileChecklist(
				context.Background(), pool, historicalFullProfileBootstrapTime,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.expected) {
				t.Fatalf("conflicting historical bootstrap error = %v, want %q", err, testCase.expected)
			}
			assertHistoricalBootstrapFailureLeavesNoAdditionalFixture(t, pool, testCase.expectedRelations)
		})
	}
}

var historicalFullProfileQuestionIDs = []string{
	"CAB-EMEQ-PBE-001", "CAB-LAV-001", "CAB-PAX-SEAT-001",
	"CAB-VID-CREW-SEAT-001", "CAB-GALLEY-001", "CAB-DOORS-001",
}

func historicalFullProfileBootstrapDatabase(t *testing.T) *database.Pool {
	t.Helper()
	pool := createTestDatabase(t, "gate0_historical_checklist")
	if err := migrations.Apply(context.Background(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return pool
}

func bootstrapHistoricalFullProfileChecklist(t *testing.T, pool *database.Pool) {
	t.Helper()
	if err := testprofile.BootstrapHistoricalFullProfileChecklist(
		context.Background(), pool, historicalFullProfileBootstrapTime,
	); err != nil {
		t.Fatalf("bootstrap historical full-profile checklist: %v", err)
	}
}

func seedConflictingHistoricalTemplate(t *testing.T, pool *database.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO checklist_template_versions (
			id, template_id, version, title, snapshot, published_at
		) VALUES ('CTV-CABIN-1', 'TPL-CABIN-2026', 1, 'Conflicting title', '{"conflict":true}', $1)
	`, historicalFullProfileBootstrapTime.Add(time.Minute)); err != nil {
		t.Fatalf("seed conflicting historical template: %v", err)
	}
}

func seedConflictingHistoricalBootstrapIdentity(t *testing.T, pool *database.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (subject_id, issuer, display_name, created_at)
		VALUES ('TEST-HISTORICAL-CHECKLIST-BOOTSTRAP', 'urn:conflicting', 'Conflicting bootstrap', $1)
	`, historicalFullProfileBootstrapTime); err != nil {
		t.Fatalf("seed conflicting historical bootstrap identity: %v", err)
	}
}

func seedConflictingHistoricalQuestion(t *testing.T, pool *database.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO identity_references (subject_id, issuer, display_name, created_at)
		VALUES ('TEST-HISTORICAL-CHECKLIST-BOOTSTRAP', 'urn:avia:internal-testprofile', 'Historical checklist bootstrap', $1)
	`, historicalFullProfileBootstrapTime); err != nil {
		t.Fatalf("seed historical bootstrap identity: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO question_versions (
			id, question_id, version, prompt, configured_reference,
			expected_evidence, created_by_subject_id, created_at
		) VALUES ('QV-CAB-EMEQ-PBE-001-V1', 'CAB-EMEQ-PBE-001', 1, 'Conflicting prompt',
			'Conflicting reference', 'Conflicting evidence', 'TEST-HISTORICAL-CHECKLIST-BOOTSTRAP', $1)
	`, historicalFullProfileBootstrapTime); err != nil {
		t.Fatalf("seed conflicting historical question: %v", err)
	}
}

func seedConflictingHistoricalTemplateMasterOwnerAndRevision(t *testing.T, pool *database.Pool) {
	t.Helper()
	bootstrapHistoricalFullProfileChecklist(t, pool)
	if _, err := pool.Exec(context.Background(), `
		UPDATE template_masters
		SET owner_role = 'Admin Preview', revision = 2
		WHERE id = 'TPL-CABIN-2026'
	`); err != nil {
		t.Fatalf("seed conflicting historical template master owner/revision: %v", err)
	}
}

func seedConflictingHistoricalTemplateMasterPointer(t *testing.T, pool *database.Pool) {
	t.Helper()
	bootstrapHistoricalFullProfileChecklist(t, pool)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO checklist_template_versions (
			id, template_id, version, title, snapshot, published_at
		) VALUES ('CTV-OTHER-1', 'TPL-CABIN-2026', 2, 'Other', '{"questions":[]}', $1)
	`, historicalFullProfileBootstrapTime); err != nil {
		t.Fatalf("seed conflicting historical template pointer target: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE template_masters
		SET published_template_version_id = 'CTV-OTHER-1'
		WHERE id = 'TPL-CABIN-2026'
	`); err != nil {
		t.Fatalf("seed conflicting historical template master pointer: %v", err)
	}
}

func assertHistoricalBootstrapFailureLeavesNoAdditionalFixture(t *testing.T, pool *database.Pool, want int) {
	t.Helper()
	var relationCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM template_version_questions WHERE template_version_id = 'CTV-CABIN-1'
	`).Scan(&relationCount); err != nil {
		t.Fatalf("read failed-bootstrap relationship count: %v", err)
	}
	if relationCount != want {
		t.Fatalf("failed bootstrap relationship count = %d, want %d", relationCount, want)
	}
}
