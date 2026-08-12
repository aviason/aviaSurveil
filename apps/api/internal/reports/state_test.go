package reports_test

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/reports"
)

func TestReportApprovalBindsExactVersionAndRoleStage(t *testing.T) {
	t.Parallel()
	dm := identity.Principal{Roles: []identity.Role{identity.RoleDepartmentManager}}
	gm := identity.Principal{Roles: []identity.Role{identity.RoleGeneralManager}}
	ed := identity.Principal{Roles: []identity.Role{identity.RoleExecutiveDirector}}

	department, err := reports.Decide(reports.DecideInput{Actor: dm, Status: reports.StatusDepartmentReview, Version: 2, ExpectedVersion: 2, Decision: reports.DecisionForward})
	if err != nil || department.Status != reports.StatusGeneralManagerReview {
		t.Fatalf("department forward = %+v, err = %v", department, err)
	}
	general, err := reports.Decide(reports.DecideInput{Actor: gm, Status: department.Status, Version: 2, ExpectedVersion: 2, Decision: reports.DecisionForward})
	if err != nil || general.Status != reports.StatusExecutiveDirectorReview {
		t.Fatalf("GM forward = %+v, err = %v", general, err)
	}
	issued, err := reports.Decide(reports.DecideInput{Actor: ed, Status: general.Status, Version: 2, ExpectedVersion: 2, Decision: reports.DecisionIssue})
	if err != nil || issued.Status != reports.StatusLocked {
		t.Fatalf("ED issue = %+v, err = %v", issued, err)
	}
	if _, err := reports.Decide(reports.DecideInput{Actor: gm, Status: general.Status, Version: 2, ExpectedVersion: 1, Decision: reports.DecisionIssue}); err == nil {
		t.Fatal("stale/unauthorized report issue accepted")
	}
}

func TestDepartmentAndGeneralManagerCanReturnOnlyAtTheirStages(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		actor  identity.Principal
		status reports.Status
	}{
		{actor: identity.Principal{Roles: []identity.Role{identity.RoleDepartmentManager}}, status: reports.StatusDepartmentReview},
		{actor: identity.Principal{Roles: []identity.Role{identity.RoleGeneralManager}}, status: reports.StatusGeneralManagerReview},
	} {
		result, err := reports.Decide(reports.DecideInput{Actor: test.actor, Status: test.status, Version: 1, ExpectedVersion: 1, Decision: reports.DecisionReturn, Reason: "Revision required."})
		if err != nil || result.Status != reports.StatusReturned {
			t.Errorf("return from %s = %+v, err = %v", test.status, result, err)
		}
	}
}

func TestPrepareReportRequiresTypedImmutableFamilyAndReadiness(t *testing.T) {
	t.Parallel()

	preliminary, err := reports.Prepare(reports.PrepareInput{
		ReportID: "PR-2026-018", Kind: reports.KindPreliminary, Version: 1,
		ContentHash: "sha256:preliminary-v1", Ready: true,
	})
	if err != nil || preliminary.Status != reports.StatusDepartmentReview ||
		preliminary.Kind != reports.KindPreliminary {
		t.Fatalf("prepare Preliminary Report = %+v, err = %v", preliminary, err)
	}

	final, err := reports.Prepare(reports.PrepareInput{
		ReportID: "FR-2026-018", Kind: reports.KindFinal, Version: 1,
		FindingIDs: []string{"FND-2026-018"}, ContentHash: "sha256:final-v1",
		Ready: true,
	})
	if err != nil || final.Status != reports.StatusDepartmentReview ||
		final.Kind != reports.KindFinal {
		t.Fatalf("prepare Final Report = %+v, err = %v", final, err)
	}

	for name, input := range map[string]reports.PrepareInput{
		"untyped family": {
			ReportID: "RPT-UNTYPED", Version: 1, ContentHash: "sha256:missing-kind", Ready: true,
		},
		"not ready": {
			ReportID: "FR-NOT-READY", Kind: reports.KindFinal, Version: 1,
			FindingIDs: []string{"FND-1"}, ContentHash: "sha256:not-ready",
		},
		"missing content hash": {
			ReportID: "PR-NO-HASH", Kind: reports.KindPreliminary, Version: 1, Ready: true,
		},
	} {
		if _, err := reports.Prepare(input); err == nil {
			t.Errorf("%s preparation succeeded", name)
		}
	}
}
