//go:build canonicaltest

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/application"
	"github.com/aviason/aviaSurveil/internal/documents"
	"github.com/aviason/aviaSurveil/internal/httpapi"
	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/reports"
	"github.com/aviason/aviaSurveil/internal/testprofile"
)

func TestReportIssueQueuesAndRendersOneImmutableAuthorizedDocument(t *testing.T) {
	pool := canonicalDatabase(t, "report_document_workflow")
	seedFinding(t, pool, "finding-report-document", "OPS-2026-018", "airline-xyz")
	if _, err := pool.Exec(context.Background(), `
		UPDATE findings
		SET status = 'CLOSED', next_action = 'Closed after accepted Evidence verification'
		WHERE id = 'finding-report-document'
	`); err != nil {
		t.Fatalf("close Finding before Final Report: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO report_versions (id, report_id, inspection_id, version, status, snapshot, created_at)
		VALUES (
			'report-final-v1', 'FR-2026-018', 'audit-cabin-001', 1,
			'EXECUTIVE_DIRECTOR_REVIEW',
			'{"kind":"FINAL","ready":true,"findingIds":["finding-report-document"],"contentHash":"sha256:final-v1","responseDueDate":"2026-08-30","caaVisibleComment":"Submit the response by the stated date."}',
			$1
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed Final Report version: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO report_approval_states (report_version_id, status, revision, updated_at)
		VALUES ('report-final-v1', 'EXECUTIVE_DIRECTOR_REVIEW', 1, $1)
	`, canonicalNow); err != nil {
		t.Fatalf("seed Final Report approval: %v", err)
	}

	service := testService(pool)
	command := application.DecideReportCommand{
		OperationID: "op-report-document-issue", CorrelationID: "corr-report-document",
		ReportVersionID: "report-final-v1", ExpectedRevision: 1,
		Decision: reports.DecisionIssue, Reason: "Issue the exact immutable Final Report version.",
	}
	actor := principal("executive-001", "caa", "session-executive", identity.RoleExecutiveDirector)
	issued, err := service.DecideReport(context.Background(), actor, command)
	if err != nil || issued.Status != reports.StatusLocked {
		t.Fatalf("issue Final Report = %+v, err = %v", issued, err)
	}
	replayed, err := service.DecideReport(context.Background(), actor, command)
	if err != nil || replayed.ReportVersionID != issued.ReportVersionID ||
		replayed.Status != issued.Status || replayed.Revision != issued.Revision ||
		replayed.IssuedAt == nil || issued.IssuedAt == nil ||
		!replayed.IssuedAt.Equal(*issued.IssuedAt) {
		t.Fatalf("replay Final Report issue = %+v, err = %v", replayed, err)
	}
	duplicate := command
	duplicate.OperationID = "op-report-document-issue-duplicate"
	duplicate.ExpectedRevision = issued.Revision
	if _, err := service.DecideReport(context.Background(), actor, duplicate); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("duplicate Final Report issue error = %v", err)
	}

	var jobs, requests, decisions int
	if err := pool.QueryRow(context.Background(), `
		SELECT
			(SELECT count(*) FROM document_render_jobs WHERE idempotency_key = 'report-render:report-final-v1'),
			(SELECT count(*) FROM outbox_messages WHERE topic = 'document.render_requested' AND aggregate_id = 'report-final-v1'),
			(SELECT count(*) FROM report_decisions WHERE report_version_id = 'report-final-v1')
	`).Scan(&jobs, &requests, &decisions); err != nil {
		t.Fatalf("read report render effects: %v", err)
	}
	if jobs != 1 || requests != 1 || decisions != 1 {
		t.Fatalf("report render effects = jobs %d requests %d decisions %d", jobs, requests, decisions)
	}

	var findingStatus string
	if err := pool.QueryRow(context.Background(),
		"SELECT status FROM findings WHERE id = 'finding-report-document'",
	).Scan(&findingStatus); err != nil {
		t.Fatalf("read Finding after report issue: %v", err)
	}
	if findingStatus != "CLOSED" {
		t.Fatalf("Final Report issue changed the already-closed Finding to %q", findingStatus)
	}

	objects := newMemoryObjectStore()
	documentService := documents.NewService(pool, objects, documents.Dependencies{
		Renderer: documents.DeterministicPDFRenderer{},
		Bucket:   "avia-documents", Clock: func() time.Time { return canonicalNow },
	})
	processed, err := documentService.ProcessNext(context.Background())
	if err != nil || !processed {
		t.Fatalf("process document render = %t, err = %v", processed, err)
	}
	processed, err = documentService.ProcessNext(context.Background())
	if err != nil || processed {
		t.Fatalf("repeat document render = %t, err = %v", processed, err)
	}

	var documentVersionID, hash, objectMetadataID string
	var version, size int64
	if err := pool.QueryRow(context.Background(), `
		SELECT id, version, sha256, size_bytes, object_metadata_id
		FROM document_versions
		WHERE document_id = (SELECT document_id FROM document_render_jobs WHERE idempotency_key = 'report-render:report-final-v1')
	`).Scan(&documentVersionID, &version, &hash, &size, &objectMetadataID); err != nil {
		t.Fatalf("read rendered DocumentVersion: %v", err)
	}
	if version != 1 || !strings.HasPrefix(hash, "sha256:") || size <= 0 || objectMetadataID == "" {
		t.Fatalf("rendered DocumentVersion = id %q version %d hash %q size %d metadata %q",
			documentVersionID, version, hash, size, objectMetadataID)
	}
	var bucket, key, objectState string
	if err := pool.QueryRow(context.Background(), `
		SELECT bucket_name, object_key, object_state
		FROM object_metadata WHERE id = $1
	`, objectMetadataID).Scan(&bucket, &key, &objectState); err != nil {
		t.Fatalf("read rendered private object metadata: %v", err)
	}
	if bucket != "avia-documents" || key == "" || objectState != "CANONICAL" ||
		!objects.Has(bucket, key) {
		t.Fatalf("rendered private object = %s/%s state %s exists %t",
			bucket, key, objectState, objects.Has(bucket, key))
	}

	auditee := principal("auditee-xyz", "airline-xyz", "session-auditee", identity.RoleAuditee)
	download, err := documentService.AuthorizeDownload(
		context.Background(), auditee, documentVersionID,
	)
	if err != nil || download.FileName != "FR-2026-018.pdf" ||
		!strings.HasPrefix(download.URL, "memory://download/") {
		t.Fatalf("authorized Auditee download = %+v, err = %v", download, err)
	}
	other := principal("auditee-other", "airline-other", "session-other", identity.RoleAuditee)
	if _, err := documentService.AuthorizeDownload(
		context.Background(), other, documentVersionID,
	); !errors.Is(err, documents.ErrForbidden) {
		t.Fatalf("cross-organization Document download error = %v", err)
	}
	seedFinding(t, pool, "finding-download-foreign", "OPS-2026-020", "airline-other")
	if _, err := pool.Exec(context.Background(), `
		UPDATE document_render_jobs
		SET input_snapshot = jsonb_set(
			input_snapshot,
			'{findingIds}',
			'["finding-download-foreign"]'::jsonb
		)
		WHERE idempotency_key = 'report-render:report-final-v1'
	`); err != nil {
		t.Fatalf("seed legacy unsafe Document render snapshot: %v", err)
	}
	if _, err := documentService.AuthorizeDownload(
		context.Background(), auditee, documentVersionID,
	); !errors.Is(err, documents.ErrForbidden) {
		t.Fatalf("same-organization unsafe Report Document download error = %v", err)
	}
}

func TestReportDecisionRejectsAnEarlierImmutableFamilyVersion(t *testing.T) {
	pool := canonicalDatabase(t, "report_exact_family_version")
	for _, version := range []struct {
		id      string
		version int
		hash    string
	}{
		{id: "report-preliminary-v1", version: 1, hash: "sha256:preliminary-v1"},
		{id: "report-preliminary-v2", version: 2, hash: "sha256:preliminary-v2"},
	} {
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO report_versions (id, report_id, inspection_id, version, status, snapshot, created_at)
			VALUES ($1, 'PR-2026-018', 'audit-cabin-001', $2, 'DEPARTMENT_REVIEW',
				jsonb_build_object('kind', 'PRELIMINARY', 'ready', true, 'findingIds', '[]'::jsonb, 'contentHash', $3::text), $4)
		`, version.id, version.version, version.hash, canonicalNow); err != nil {
			t.Fatalf("seed Preliminary Report version %d: %v", version.version, err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO report_approval_states (report_version_id, status, revision, updated_at)
			VALUES ($1, 'DEPARTMENT_REVIEW', 1, $2)
		`, version.id, canonicalNow); err != nil {
			t.Fatalf("seed Preliminary Report approval %d: %v", version.version, err)
		}
	}

	_, err := testService(pool).DecideReport(
		context.Background(),
		principal("manager-001", "caa", "session-manager", identity.RoleDepartmentManager),
		application.DecideReportCommand{
			OperationID: "op-report-earlier-version", CorrelationID: "corr-report-earlier-version",
			ReportVersionID: "report-preliminary-v1", ExpectedRevision: 1,
			Decision: reports.DecisionForward, Reason: "Attempt to advance an earlier immutable version.",
		},
	)
	if !errors.Is(err, application.ErrConflict) {
		t.Fatalf("earlier immutable Report decision error = %v", err)
	}
}

func TestReportDecisionRejectsFamilyKindDriftAndForeignFindingIdentity(t *testing.T) {
	t.Run("immutable family kind drift", func(t *testing.T) {
		pool := canonicalDatabase(t, "report_family_kind_drift")
		for _, version := range []struct {
			id      string
			version int
			kind    string
		}{
			{id: "report-family-v1", version: 1, kind: "PRELIMINARY"},
			{id: "report-family-v2", version: 2, kind: "FINAL"},
		} {
			if _, err := pool.Exec(context.Background(), `
				INSERT INTO report_versions (
					id, report_id, inspection_id, version, status, snapshot, created_at
				) VALUES (
					$1, 'REPORT-FAMILY-DRIFT', 'audit-cabin-001', $2::int,
					'DEPARTMENT_REVIEW',
					jsonb_build_object(
						'kind', $3::text, 'ready', true, 'findingIds', '[]'::jsonb,
						'contentHash', ('sha256:family-' || ($2::int)::text)
					),
					$4
				)
			`, version.id, version.version, version.kind, canonicalNow); err != nil {
				t.Fatalf("seed Report family version %d: %v", version.version, err)
			}
			if _, err := pool.Exec(context.Background(), `
				INSERT INTO report_approval_states (
					report_version_id, status, revision, updated_at
				) VALUES ($1, 'DEPARTMENT_REVIEW', 1, $2)
			`, version.id, canonicalNow); err != nil {
				t.Fatalf("seed Report family approval %d: %v", version.version, err)
			}
		}
		_, err := testService(pool).DecideReport(
			context.Background(),
			principal("manager-001", "caa", "session-manager", identity.RoleDepartmentManager),
			application.DecideReportCommand{
				OperationID: "op-report-family-kind-drift", CorrelationID: "corr-report-family-kind-drift",
				ReportVersionID: "report-family-v2", ExpectedRevision: 1,
				Decision: reports.DecisionForward, Reason: "Typed report family must remain immutable.",
			},
		)
		if !errors.Is(err, application.ErrConflict) {
			t.Fatalf("Report family kind drift error = %v", err)
		}
	})

	t.Run("foreign Finding identity", func(t *testing.T) {
		pool := canonicalDatabase(t, "report_foreign_finding")
		seedFinding(t, pool, "finding-report-foreign", "OPS-2026-019", "airline-other")
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO report_versions (
				id, report_id, inspection_id, version, status, snapshot, created_at
			) VALUES (
				'report-foreign-finding-v1', 'FR-FOREIGN-FINDING', 'audit-cabin-001', 1,
				'EXECUTIVE_DIRECTOR_REVIEW',
				'{"kind":"FINAL","ready":true,"findingIds":["finding-report-foreign"],"contentHash":"sha256:foreign-finding"}',
				$1
			)
		`, canonicalNow); err != nil {
			t.Fatalf("seed foreign-Finding Report: %v", err)
		}
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO report_approval_states (
				report_version_id, status, revision, updated_at
			) VALUES ('report-foreign-finding-v1', 'EXECUTIVE_DIRECTOR_REVIEW', 1, $1)
		`, canonicalNow); err != nil {
			t.Fatalf("seed foreign-Finding Report approval: %v", err)
		}
		_, err := testService(pool).DecideReport(
			context.Background(),
			principal("executive-001", "caa", "session-executive", identity.RoleExecutiveDirector),
			application.DecideReportCommand{
				OperationID: "op-report-foreign-finding", CorrelationID: "corr-report-foreign-finding",
				ReportVersionID: "report-foreign-finding-v1", ExpectedRevision: 1,
				Decision: reports.DecisionIssue, Reason: "Foreign Finding must fail closed.",
			},
		)
		if !errors.Is(err, application.ErrConflict) {
			t.Fatalf("foreign-Finding Report issue error = %v", err)
		}
		var jobs int
		if err := pool.QueryRow(
			context.Background(),
			"SELECT count(*) FROM document_render_jobs",
		).Scan(&jobs); err != nil {
			t.Fatalf("count foreign-Finding render jobs: %v", err)
		}
		if jobs != 0 {
			t.Fatalf("foreign-Finding Report render jobs = %d", jobs)
		}
	})
}

func TestReportDocumentHTTPUsesExactIdentityAndAuditeeSafeProjections(t *testing.T) {
	pool := canonicalDatabase(t, "report_document_http")
	if err := testprofile.Reset(context.Background(), pool, canonicalNow); err != nil {
		t.Fatalf("reset canonical report HTTP profile: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE report_approval_states
		SET status = 'LOCKED', revision = 4, issued_at = $1, updated_at = $1
		WHERE report_version_id = 'PR-2026-018-V1'
	`, canonicalNow); err != nil {
		t.Fatalf("seed issued Preliminary Report approval before Final Report flow: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO report_versions (
			id, report_id, inspection_id, version, status, snapshot, created_at
		) VALUES (
			'RPT-UNSAFE-FINDING-V1', 'RPT-UNSAFE-FINDING', 'AUD-2026-001', 1,
			'LOCKED',
			'{"kind":"FINAL","ready":true,"findingIds":["FND-SKYCARGO-2026-099"],"contentHash":"sha256:unsafe-cross-organization"}',
			$1
		)
	`, canonicalNow); err != nil {
		t.Fatalf("seed unsafe cross-organization Report: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO report_approval_states (
			report_version_id, status, revision, issued_at, updated_at
		) VALUES ('RPT-UNSAFE-FINDING-V1', 'LOCKED', 2, $1, $1)
	`, canonicalNow); err != nil {
		t.Fatalf("seed unsafe Report approval: %v", err)
	}

	objects := newMemoryObjectStore()
	documentService := documents.NewService(
		pool,
		objects,
		documents.Dependencies{
			Renderer: documents.DeterministicPDFRenderer{},
			Bucket:   "generated-documents", Clock: func() time.Time { return canonicalNow },
			WorkerID: "document-http-test",
		},
	)
	api := httpapi.NewCanonicalAPI(httpapi.CanonicalAPIDependencies{
		Pool: pool, Application: testService(pool), Documents: documentService,
		Clock: func() time.Time { return canonicalNow },
	})
	handler := httpapi.NewCanonicalTestBoundary("task-7-token").Protect(api.Handler())
	request := func(method, path, body, subjectID string) *httptest.ResponseRecorder {
		httpRequest := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		httpRequest.Header.Set(httpapi.CanonicalTestTokenHeader, "task-7-token")
		httpRequest.Header.Set(httpapi.CanonicalTestSubjectHeader, subjectID)
		if body != "" {
			httpRequest.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httpRequest)
		return response
	}

	drift := request(http.MethodPost, "/v1/report-versions/WRONG-REPORT/decisions", `{
		"operationId":"OP-REPORT-DRIFT",
		"reportVersionId":"RPT-CAB-2026-001-V1",
		"expectedReportVersionRevision":1,
		"decision":"ISSUE_AND_LOCK",
		"reason":"Path and body must match."
	}`, "USR-ED-ZARA")
	if drift.Code != http.StatusUnprocessableEntity {
		t.Fatalf("Report path/body drift status=%d body=%s", drift.Code, drift.Body.String())
	}

	unreleased := request(
		http.MethodGet,
		"/v1/auditee/report-versions/RPT-CAB-2026-001-V1",
		"",
		"USR-AUDITEE-FLY",
	)
	if unreleased.Code != http.StatusForbidden {
		t.Fatalf("unreleased Auditee Report status=%d body=%s", unreleased.Code, unreleased.Body.String())
	}
	unreleasedDocuments := request(
		http.MethodGet,
		"/v1/documents?organizationId=ORG-FLY-NAMIBIA",
		"",
		"USR-AUDITEE-FLY",
	)
	if unreleasedDocuments.Code != http.StatusOK ||
		strings.Contains(unreleasedDocuments.Body.String(), "RPT-CAB-2026-001-V1") {
		t.Fatalf(
			"unreleased Auditee documents status=%d body=%s",
			unreleasedDocuments.Code,
			unreleasedDocuments.Body.String(),
		)
	}

	dmForwarded := request(http.MethodPost, "/v1/report-versions/RPT-CAB-2026-001-V1/decisions", `{
		"operationId":"OP-REPORT-HTTP-DM-FORWARD",
		"reportVersionId":"RPT-CAB-2026-001-V1",
		"expectedReportVersionRevision":1,
		"decision":"FORWARD",
		"reason":"Forward the exact canonical Final Report version to the General Manager."
	}`, "USR-MANAGER-NORA")
	if dmForwarded.Code != http.StatusOK ||
		!strings.Contains(dmForwarded.Body.String(), `"status":"GM_REVIEW"`) {
		t.Fatalf("Department Manager forward Report status=%d body=%s", dmForwarded.Code, dmForwarded.Body.String())
	}
	gmForwarded := request(http.MethodPost, "/v1/report-versions/RPT-CAB-2026-001-V1/decisions", `{
		"operationId":"OP-REPORT-HTTP-GM-FORWARD",
		"reportVersionId":"RPT-CAB-2026-001-V1",
		"expectedReportVersionRevision":2,
		"decision":"FORWARD",
		"reason":"Forward the exact canonical Final Report version to the Executive Director."
	}`, "USR-GM-OMAR")
	if gmForwarded.Code != http.StatusOK ||
		!strings.Contains(gmForwarded.Body.String(), `"status":"EXECUTIVE_DIRECTOR_REVIEW"`) {
		t.Fatalf("General Manager forward Report status=%d body=%s", gmForwarded.Code, gmForwarded.Body.String())
	}
	issued := request(http.MethodPost, "/v1/report-versions/RPT-CAB-2026-001-V1/decisions", `{
		"operationId":"OP-REPORT-HTTP-ISSUE",
		"reportVersionId":"RPT-CAB-2026-001-V1",
		"expectedReportVersionRevision":3,
		"decision":"ISSUE_AND_LOCK",
		"reason":"Issue the exact canonical Final Report version."
	}`, "USR-ED-ZARA")
	if issued.Code != http.StatusOK ||
		!strings.Contains(issued.Body.String(), `"status":"LOCKED"`) ||
		!strings.Contains(issued.Body.String(), `"findingIds":[]`) {
		t.Fatalf("issue Report status=%d body=%s", issued.Code, issued.Body.String())
	}
	pendingDocument := request(
		http.MethodGet,
		"/v1/documents/RPT-CAB-2026-001-V1",
		"",
		"USR-AUDITEE-FLY",
	)
	if pendingDocument.Code != http.StatusOK ||
		!strings.Contains(pendingDocument.Body.String(), `"renderStatus":"PENDING"`) ||
		strings.Contains(pendingDocument.Body.String(), `"downloadUrl"`) {
		t.Fatalf(
			"pending generated Document status=%d body=%s",
			pendingDocument.Code,
			pendingDocument.Body.String(),
		)
	}
	processed, err := documentService.ProcessNext(context.Background())
	if err != nil || !processed {
		t.Fatalf("process HTTP generated Document = %t, err %v", processed, err)
	}

	for _, check := range []struct {
		path       string
		required   []string
		prohibited []string
	}{
		{
			path: "/v1/auditee/report-versions?kind=FINAL",
			required: []string{
				`"reportVersionId":"RPT-CAB-2026-001-V1"`,
				`"kind":"FINAL"`,
			},
			prohibited: []string{"RPT-UNSAFE-FINDING-V1", "FND-SKYCARGO-2026-099"},
		},
		{
			path: "/v1/auditee/report-versions/RPT-CAB-2026-001-V1",
			required: []string{
				`"reportVersionId":"RPT-CAB-2026-001-V1"`,
				`"status":"LOCKED"`,
			},
		},
		{
			path: "/v1/documents",
			required: []string{
				`"id":"RPT-CAB-2026-001-V1"`,
				`"publicReviewResult":"RELEASED"`,
				`"downloadFileName":"RPT-CAB-2026-001.pdf"`,
				`"renderStatus":"SUCCEEDED"`,
			},
			prohibited: []string{
				"RPT-UNSAFE-FINDING-V1", "FND-SKYCARGO-2026-099", `"downloadUrl"`,
			},
		},
		{
			path: "/v1/documents/RPT-CAB-2026-001-V1",
			required: []string{
				`"id":"RPT-CAB-2026-001-V1"`,
				`"kind":"REPORT"`,
				`"renderStatus":"SUCCEEDED"`,
				`"documentVersionId":`,
				`"downloadUrl":"memory://download/generated-documents/`,
				`"rendererHash":"sha256:`,
				`"templateHash":"sha256:`,
				`"sourceHash":"sha256:`,
			},
		},
	} {
		response := request(http.MethodGet, check.path, "", "USR-AUDITEE-FLY")
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", check.path, response.Code, response.Body.String())
		}
		for _, required := range check.required {
			if !strings.Contains(response.Body.String(), required) {
				t.Errorf("GET %s missing %s in %s", check.path, required, response.Body.String())
			}
		}
		for _, prohibited := range check.prohibited {
			if strings.Contains(response.Body.String(), prohibited) {
				t.Errorf("GET %s disclosed %s in %s", check.path, prohibited, response.Body.String())
			}
		}
	}

	unsafe := request(
		http.MethodGet,
		"/v1/auditee/report-versions/RPT-UNSAFE-FINDING-V1",
		"",
		"USR-AUDITEE-FLY",
	)
	if unsafe.Code != http.StatusForbidden {
		t.Fatalf("unsafe Auditee Report status=%d body=%s", unsafe.Code, unsafe.Body.String())
	}
}
