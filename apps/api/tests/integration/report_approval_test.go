package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/application"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/reports"
)

func TestReportApprovalUsesExactVersionAndIssueNeverClosesFinding(t *testing.T) {
	pool := canonicalDatabase(t, "report_approval")
	seedFinding(t, pool, "finding-report", "OPS-2026-010", "airline-xyz")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO report_versions (id, report_id, inspection_id, version, status, snapshot)
		VALUES ('report-version-001', 'report-001', 'audit-cabin-001', 1, 'DRAFT',
			'{"kind":"FINAL","ready":true,"findingIds":["finding-report"],"contentHash":"sha256:hash-001"}')
	`); err != nil {
		t.Fatalf("seed report version: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO report_approval_states (report_version_id, status, revision)
		VALUES ('report-version-001', 'DEPARTMENT_REVIEW', 1)
	`); err != nil {
		t.Fatalf("seed report approval: %v", err)
	}
	service := testService(pool)

	_, err := service.DecideReport(context.Background(), principal("manager-001", "caa", "session-manager", identity.RoleDepartmentManager), application.DecideReportCommand{
		OperationID: "op-report-dm", CorrelationID: "corr-report", ReportVersionID: "report-version-001", ExpectedRevision: 1, Decision: reports.DecisionForward,
	})
	if !errors.Is(err, application.ErrConflict) {
		t.Fatalf("Final Report approval error = %v, want lifecycle gate until Finding closure", err)
	}

	var findingStatus string
	if err := pool.QueryRow(context.Background(), "SELECT status FROM findings WHERE id = 'finding-report'").Scan(&findingStatus); err != nil {
		t.Fatalf("read Finding after report issue: %v", err)
	}
	if findingStatus != "WAITING_FOR_CAP" {
		t.Fatalf("report issue changed Finding status to %q", findingStatus)
	}
	var reportSnapshot string
	if err := pool.QueryRow(context.Background(), "SELECT snapshot::text FROM report_versions WHERE id = 'report-version-001'").Scan(&reportSnapshot); err != nil || reportSnapshot == "" {
		t.Fatalf("immutable report snapshot = %q, err = %v", reportSnapshot, err)
	}
	var reportStatus string
	if err := pool.QueryRow(context.Background(), "SELECT status FROM report_versions WHERE id = 'report-version-001'").Scan(&reportStatus); err != nil || reportStatus != "DRAFT" {
		t.Fatalf("blocked Final Report mutated status to %q, err = %v", reportStatus, err)
	}
}
