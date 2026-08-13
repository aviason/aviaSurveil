package regulatory

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
)

type ReviewQueueKind string

const (
	SourceReviewQueue ReviewQueueKind = "SOURCE_REVIEW"
	ReviewerQueue     ReviewQueueKind = "CHECKLIST_REVIEW"
)

type ReviewQueueItem struct {
	ReviewItemID  string          `json:"reviewItemId"`
	CandidateID   string          `json:"candidateId"`
	Kind          ReviewQueueKind `json:"kind"`
	ScopeKey      string          `json:"scopeKey"`
	Status        string          `json:"status"`
	CandidateOnly bool            `json:"candidateOnly"`
}

// BuildScopedReviewQueue is intentionally conservative: an empty or
// unassigned scope produces no rows rather than a guessed global queue.
func BuildScopedReviewQueue(assignments []identity.FunctionalAssignment, subjectID string, kind ReviewQueueKind, items []ReviewQueueItem) ([]ReviewQueueItem, error) {
	if strings.TrimSpace(subjectID) == "" || (kind != SourceReviewQueue && kind != ReviewerQueue) {
		return nil, errors.New("review queue scope is incomplete")
	}
	authorized := false
	for _, assignment := range identity.ResolveCurrentFunctionalAssignments(assignments, time.Now().UTC()) {
		if assignment.SubjectID == subjectID {
			authorized = true
			break
		}
	}
	if !authorized {
		return []ReviewQueueItem{}, nil
	}
	output := make([]ReviewQueueItem, 0, len(items))
	for _, item := range items {
		if item.Kind == kind && item.CandidateOnly && strings.TrimSpace(item.ScopeKey) != "" {
			output = append(output, item)
		}
	}
	sort.SliceStable(output, func(i, j int) bool { return output[i].ReviewItemID < output[j].ReviewItemID })
	return output, nil
}
