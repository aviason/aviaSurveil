package inspections

import (
	"fmt"

	"github.com/aviason/aviaSurveil/internal/identity"
)

type PotentialFindingContext struct {
	AuditID                 string
	QuestionAuditID         string
	ResponseAuditID         string
	QuestionID              string
	ResponseQuestionID      string
	AssignedInspectorUserID string
}

func CanAnswerQuestion(principal identity.Principal, assignedSubjectIDs []string) bool {
	if !principal.HasRole(identity.RoleInspector) || principal.SubjectID == "" {
		return false
	}
	for _, subjectID := range assignedSubjectIDs {
		if principal.SubjectID == subjectID {
			return true
		}
	}
	return false
}

func ValidatePotentialFindingContext(context PotentialFindingContext, principal identity.Principal) error {
	if !principal.HasRole(identity.RoleInspector) {
		return fmt.Errorf("only an Inspector can create a Potential Finding")
	}
	if context.AuditID == "" || context.QuestionID == "" {
		return fmt.Errorf("Audit and question identity are required")
	}
	if context.AuditID != context.QuestionAuditID || context.AuditID != context.ResponseAuditID {
		return fmt.Errorf("Potential Finding context crosses Audit scope")
	}
	if context.QuestionID != context.ResponseQuestionID {
		return fmt.Errorf("Potential Finding context crosses question scope")
	}
	if context.AssignedInspectorUserID != principal.SubjectID {
		return fmt.Errorf("Inspector is not assigned to the question")
	}
	return nil
}
