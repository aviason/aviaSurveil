package identity

import (
	"testing"
	"time"
)

func TestResolveCurrentChecklistAssignmentsChoosesLatestSuccessorBeforeFiltering(t *testing.T) {
	at := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	assignments := []FunctionalAssignment{
		{AssignmentID: "old", RootID: "root", Version: 1, SubjectID: "subject", MembershipID: "membership", Permission: FunctionalPermissionChecklistReviewer, Scope: FunctionalAssignmentScope{Kind: ScopeDepartmentChecklists, DepartmentID: "D1", OrganizationalUnitID: "U1"}, EffectiveFrom: at.Add(-24 * time.Hour), Status: FunctionalAssignmentActive},
		{AssignmentID: "new-future", RootID: "root", Version: 2, SubjectID: "subject", MembershipID: "membership", Permission: FunctionalPermissionChecklistReviewer, Scope: FunctionalAssignmentScope{Kind: ScopeDepartmentChecklists, DepartmentID: "D1", OrganizationalUnitID: "U1"}, EffectiveFrom: at.Add(time.Hour), Status: FunctionalAssignmentActive},
	}
	current := ResolveCurrentFunctionalAssignments(assignments, at)
	if len(current) != 0 {
		t.Fatalf("future successor must hide prior assignment before effective-date filtering: %+v", current)
	}
}

func TestAuthorizeChecklistFunctionalAssignmentFailsClosedOnMissingScopeAndCrossSubject(t *testing.T) {
	at := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	base := FunctionalAssignment{AssignmentID: "a", RootID: "root", Version: 1, SubjectID: "subject", MembershipID: "membership", Permission: FunctionalPermissionRegulatorySourceOwner, Scope: FunctionalAssignmentScope{Kind: ScopeSourceIdentity, DepartmentID: "D1", OrganizationalUnitID: "U1", SourceID: "source-1", SourceVersionID: "source-v1", ChainRole: "PRIMARY"}, EffectiveFrom: at.Add(-time.Hour), Status: FunctionalAssignmentActive}
	if AuthorizeFunctionalAssignment([]FunctionalAssignment{base}, FunctionalAuthorizationQuery{SubjectID: "other", MembershipID: "membership", Permission: FunctionalPermissionRegulatorySourceOwner, At: at, Scope: base.Scope}) {
		t.Fatal("cross-subject assignment must be denied")
	}
	missing := base
	missing.Scope.SourceVersionID = ""
	if AuthorizeFunctionalAssignment([]FunctionalAssignment{missing}, FunctionalAuthorizationQuery{SubjectID: "subject", MembershipID: "membership", Permission: FunctionalPermissionRegulatorySourceOwner, At: at, Scope: missing.Scope}) {
		t.Fatal("source-owner scope without exact source identity must be denied")
	}
}

func TestAuthorizeChecklistFunctionalAssignmentRequiresInternalMembershipIdentity(t *testing.T) {
	at := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	assignment := FunctionalAssignment{AssignmentID: "a", RootID: "root", Version: 1, SubjectID: "subject", MembershipID: "membership", Permission: FunctionalPermissionChecklistReviewer, Scope: FunctionalAssignmentScope{Kind: ScopeDepartmentChecklists, DepartmentID: "D1", OrganizationalUnitID: "U1"}, EffectiveFrom: at.Add(-time.Hour), Status: FunctionalAssignmentActive}
	if AuthorizeFunctionalAssignment([]FunctionalAssignment{assignment}, FunctionalAuthorizationQuery{SubjectID: "subject", MembershipID: "", Permission: FunctionalPermissionChecklistReviewer, At: at, Scope: assignment.Scope}) {
		t.Fatal("missing internal-CAA membership must be denied")
	}
}
