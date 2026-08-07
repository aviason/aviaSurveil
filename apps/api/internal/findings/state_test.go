package findings_test

import (
	"testing"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/findings"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

func TestAuthorizedClosureIsSeparateManagerReasonRequiredPath(t *testing.T) {
	t.Parallel()
	manager := identity.Principal{Roles: []identity.Role{identity.RoleDepartmentManager}}
	closed, err := findings.AuthorizedClose(findings.AuthorizedCloseInput{
		Actor: manager, Status: findings.StatusPendingClosure, Revision: 8, ExpectedRevision: 8,
		Reason: "Authorized alternate verification documented.",
	})
	if err != nil || closed.Status != findings.StatusClosed || closed.ClosureBasis != findings.ClosureBasisAuthorized {
		t.Fatalf("authorized close = %+v, err = %v", closed, err)
	}
	if _, err := findings.AuthorizedClose(findings.AuthorizedCloseInput{
		Actor: manager, Status: findings.StatusPendingClosure, Revision: 8, ExpectedRevision: 8,
	}); err == nil {
		t.Fatal("reasonless authorized closure accepted")
	}
	if _, err := findings.AuthorizedClose(findings.AuthorizedCloseInput{
		Actor: identity.Principal{Roles: []identity.Role{identity.RoleInspector}}, Status: findings.StatusPendingClosure,
		Revision: 8, ExpectedRevision: 8, Reason: "not authorized",
	}); err == nil {
		t.Fatal("Inspector used Department Manager closure")
	}
}
