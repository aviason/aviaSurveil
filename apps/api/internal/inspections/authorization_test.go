package inspections_test

import (
	"testing"

	"github.com/aviason/aviaSurveil/internal/identity"
	"github.com/aviason/aviaSurveil/internal/inspections"
)

func TestInspectorCanAnswerOnlyAssignedQuestion(t *testing.T) {
	t.Parallel()
	principal := identity.Principal{SubjectID: "inspector-cabin-001", Roles: []identity.Role{identity.RoleInspector}}
	if !inspections.CanAnswerQuestion(principal, []string{"inspector-cabin-001"}) {
		t.Fatal("assigned Inspector denied")
	}
	if inspections.CanAnswerQuestion(principal, []string{"inspector-other"}) {
		t.Fatal("unassigned Inspector allowed")
	}
}

func TestPotentialFindingContextMustMatchAuditQuestionAndResponse(t *testing.T) {
	t.Parallel()
	err := inspections.ValidatePotentialFindingContext(inspections.PotentialFindingContext{
		AuditID:                 "audit-cabin-001",
		QuestionAuditID:         "audit-cabin-001",
		ResponseAuditID:         "audit-other",
		QuestionID:              "q-cabin-crew-training",
		ResponseQuestionID:      "q-cabin-crew-training",
		AssignedInspectorUserID: "inspector-cabin-001",
	}, identity.Principal{SubjectID: "inspector-cabin-001", Roles: []identity.Role{identity.RoleInspector}})
	if err == nil {
		t.Fatal("cross-Audit response accepted")
	}
}
