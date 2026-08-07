package checklists

import (
	"fmt"
	"strings"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

type Status string

const (
	StatusNotStarted Status = "NOT_STARTED"
	StatusInProgress Status = "IN_PROGRESS"
	StatusSubmitted  Status = "SUBMITTED"
)

type ReopenInput struct {
	Actor            identity.Principal
	Status           Status
	Revision         int64
	ExpectedRevision int64
	Reason           string
}

type SubmitInput struct {
	Actor              identity.Principal
	AssignedSubjectIDs []string
	Status             Status
	Revision           int64
	ExpectedRevision   int64
}

type SubmitResult struct {
	Status   Status
	Revision int64
}

type ReopenResult struct {
	Status   Status
	Revision int64
}

func CanEdit(status Status) bool {
	return status == StatusInProgress
}

func Submit(input SubmitInput) (SubmitResult, error) {
	if !input.Actor.HasRole(identity.RoleInspector) || input.Actor.SubjectID == "" {
		return SubmitResult{}, fmt.Errorf("only an assigned Inspector can submit a checklist")
	}
	assigned := false
	for _, subjectID := range input.AssignedSubjectIDs {
		if subjectID == input.Actor.SubjectID {
			assigned = true
			break
		}
	}
	if !assigned {
		return SubmitResult{}, fmt.Errorf("Inspector is not assigned to this checklist")
	}
	if input.Status != StatusInProgress {
		return SubmitResult{}, fmt.Errorf("only an in-progress checklist can be submitted")
	}
	if input.Revision != input.ExpectedRevision {
		return SubmitResult{}, fmt.Errorf("stale checklist revision")
	}
	return SubmitResult{Status: StatusSubmitted, Revision: input.Revision + 1}, nil
}

func Reopen(input ReopenInput) (ReopenResult, error) {
	if !input.Actor.HasRole(identity.RoleLeadInspector, identity.RoleDepartmentManager) {
		return ReopenResult{}, fmt.Errorf("role cannot reopen a checklist")
	}
	if input.Status != StatusSubmitted {
		return ReopenResult{}, fmt.Errorf("only a submitted checklist can be reopened")
	}
	if input.Revision != input.ExpectedRevision {
		return ReopenResult{}, fmt.Errorf("stale checklist revision")
	}
	if strings.TrimSpace(input.Reason) == "" {
		return ReopenResult{}, fmt.Errorf("reopen reason is required")
	}
	return ReopenResult{Status: StatusInProgress, Revision: input.Revision + 1}, nil
}
