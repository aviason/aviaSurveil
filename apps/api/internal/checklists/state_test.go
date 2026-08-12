package checklists_test

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/checklists"
	"github.com/aviason/aviaSurveil/internal/identity"
)

func TestSubmittedChecklistIsReadOnly(t *testing.T) {
	t.Parallel()
	if checklists.CanEdit(checklists.StatusSubmitted) {
		t.Fatal("submitted checklist remained editable")
	}
	if !checklists.CanEdit(checklists.StatusInProgress) {
		t.Fatal("in-progress checklist is not editable")
	}
	if checklists.CanEdit(checklists.StatusNotStarted) {
		t.Fatal("NOT_STARTED checklist became editable before Inspector start")
	}
}

func TestSubmitRequiresAssignedInspectorInProgressAndExactRevision(t *testing.T) {
	t.Parallel()
	inspector := identity.Principal{SubjectID: "inspector-cabin-001", Roles: []identity.Role{identity.RoleInspector}}
	result, err := checklists.Submit(checklists.SubmitInput{
		Actor: inspector, AssignedSubjectIDs: []string{"inspector-cabin-001"},
		Status: checklists.StatusInProgress, Revision: 3, ExpectedRevision: 3,
	})
	if err != nil || result.Status != checklists.StatusSubmitted || result.Revision != 4 {
		t.Fatalf("valid submit = %+v, err = %v", result, err)
	}
	for name, input := range map[string]checklists.SubmitInput{
		"unassigned": {Actor: inspector, AssignedSubjectIDs: []string{"inspector-other"}, Status: checklists.StatusInProgress, Revision: 3, ExpectedRevision: 3},
		"wrong role": {Actor: identity.Principal{SubjectID: "lead-001", Roles: []identity.Role{identity.RoleLeadInspector}}, AssignedSubjectIDs: []string{"lead-001"}, Status: checklists.StatusInProgress, Revision: 3, ExpectedRevision: 3},
		"submitted":  {Actor: inspector, AssignedSubjectIDs: []string{"inspector-cabin-001"}, Status: checklists.StatusSubmitted, Revision: 3, ExpectedRevision: 3},
		"stale":      {Actor: inspector, AssignedSubjectIDs: []string{"inspector-cabin-001"}, Status: checklists.StatusInProgress, Revision: 3, ExpectedRevision: 2},
	} {
		if _, err := checklists.Submit(input); err == nil {
			t.Errorf("%s checklist submit accepted", name)
		}
	}
}

func TestReopenRequiresPermittedRoleStageReasonAndExactRevision(t *testing.T) {
	t.Parallel()
	lead := identity.Principal{Roles: []identity.Role{identity.RoleLeadInspector}}
	result, err := checklists.Reopen(checklists.ReopenInput{
		Actor:            lead,
		Status:           checklists.StatusSubmitted,
		Revision:         4,
		ExpectedRevision: 4,
		Reason:           "Response requires documented correction.",
	})
	if err != nil || result.Status != checklists.StatusInProgress || result.Revision != 5 {
		t.Fatalf("valid reopen = %+v, err = %v", result, err)
	}
	for name, input := range map[string]checklists.ReopenInput{
		"missing reason": {Actor: lead, Status: checklists.StatusSubmitted, Revision: 4, ExpectedRevision: 4},
		"wrong role":     {Actor: identity.Principal{Roles: []identity.Role{identity.RoleInspector}}, Status: checklists.StatusSubmitted, Revision: 4, ExpectedRevision: 4, Reason: "required"},
		"wrong stage":    {Actor: lead, Status: checklists.StatusInProgress, Revision: 4, ExpectedRevision: 4, Reason: "required"},
		"stale revision": {Actor: lead, Status: checklists.StatusSubmitted, Revision: 4, ExpectedRevision: 3, Reason: "required"},
	} {
		if _, err := checklists.Reopen(input); err == nil {
			t.Errorf("%s reopen accepted", name)
		}
	}
}
