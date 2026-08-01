package regulatory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

type CandidateDraftQuestion struct {
	QuestionID string `json:"questionId"`
	Wording    string `json:"wording"`
}

type CandidateDraftInput struct {
	CandidateID       string                   `json:"candidateId"`
	CandidateRevision int64                    `json:"candidateRevision"`
	CandidateDigest   string                   `json:"candidateDigest"`
	ProviderScopeID   string                   `json:"providerScopeId"`
	TargetID          string                   `json:"targetId"`
	InspectionType    string                   `json:"inspectionType"`
	Questions         []CandidateDraftQuestion `json:"questions"`
}

type CandidateDraft struct {
	DraftID              string                   `json:"draftId"`
	Origin               string                   `json:"origin"`
	GenerationRunID      string                   `json:"generationRunId"`
	CandidateID          string                   `json:"candidateId"`
	CandidateRevision    int64                    `json:"candidateRevision"`
	CandidateDigest      string                   `json:"candidateDigest"`
	OwnerResolutionState string                   `json:"ownerResolutionState"`
	Blockers             []string                 `json:"blockers"`
	Questions            []CandidateDraftQuestion `json:"questions"`
}

func CreateDraftFromExistingCandidate(input CandidateDraftInput) (CandidateDraft, error) {
	if strings.TrimSpace(input.CandidateID) == "" || input.CandidateRevision <= 0 || !strings.HasPrefix(input.CandidateDigest, "sha256:") || strings.TrimSpace(input.ProviderScopeID) == "" || strings.TrimSpace(input.TargetID) == "" || strings.TrimSpace(input.InspectionType) == "" || len(input.Questions) == 0 {
		return CandidateDraft{}, errors.New("candidate draft input is incomplete")
	}
	for _, question := range input.Questions {
		if strings.TrimSpace(question.QuestionID) == "" || strings.TrimSpace(question.Wording) == "" {
			return CandidateDraft{}, errors.New("candidate draft question is incomplete")
		}
	}
	digest, _ := json.Marshal(input)
	draftSum := sha256.Sum256(digest)
	return CandidateDraft{DraftID: "draft-" + hex.EncodeToString(draftSum[:8]), Origin: string(ExistingChecklistCandidateOrigin), CandidateID: input.CandidateID, CandidateRevision: input.CandidateRevision, CandidateDigest: input.CandidateDigest, OwnerResolutionState: "REVIEW_REQUIRED", Blockers: []string{SourceMappingRequired, "OWNER_RESOLUTION_REQUIRED"}, Questions: append([]CandidateDraftQuestion(nil), input.Questions...)}, nil
}

type HybridReconciliationInput struct {
	PredecessorID     string
	PredecessorDigest string
	Questions         []CandidateDraftQuestion
}

type ReconciliationDiff struct {
	QuestionID   string `json:"questionId"`
	Before       string `json:"before"`
	After        string `json:"after"`
	BeforeDigest string `json:"beforeDigest"`
	AfterDigest  string `json:"afterDigest"`
	Outcome      string `json:"outcome"`
}

type HybridReconciledDraft struct {
	Origin            string               `json:"origin"`
	PredecessorID     string               `json:"predecessorId"`
	PredecessorDigest string               `json:"predecessorDigest"`
	Diffs             []ReconciliationDiff `json:"diffs"`
	BindingRequired   bool                 `json:"bindingRequired"`
}

func CreateHybridReconciliation(input HybridReconciliationInput, before []CandidateDraftQuestion) (HybridReconciledDraft, error) {
	if strings.TrimSpace(input.PredecessorID) == "" || !strings.HasPrefix(input.PredecessorDigest, "sha256:") {
		return HybridReconciledDraft{}, errors.New("hybrid predecessor identity is incomplete")
	}
	old := make(map[string]CandidateDraftQuestion, len(before))
	for _, question := range before {
		old[question.QuestionID] = question
	}
	seen := make(map[string]bool, len(input.Questions))
	diffs := make([]ReconciliationDiff, 0, len(before)+len(input.Questions))
	for _, question := range input.Questions {
		if strings.TrimSpace(question.QuestionID) == "" {
			return HybridReconciledDraft{}, errors.New("hybrid question identity is empty")
		}
		seen[question.QuestionID] = true
		previous := old[question.QuestionID]
		beforeDigest := digestText(previous.Wording)
		afterDigest := digestText(question.Wording)
		outcome := "UNCHANGED"
		if previous.QuestionID == "" {
			outcome = "ADDED"
		} else if previous.Wording != question.Wording {
			outcome = "CHANGED"
		}
		diffs = append(diffs, ReconciliationDiff{QuestionID: question.QuestionID, Before: previous.Wording, After: question.Wording, BeforeDigest: beforeDigest, AfterDigest: afterDigest, Outcome: outcome})
	}
	for _, question := range before {
		if !seen[question.QuestionID] {
			diffs = append(diffs, ReconciliationDiff{QuestionID: question.QuestionID, Before: question.Wording, BeforeDigest: digestText(question.Wording), Outcome: "REMOVED"})
		}
	}
	return HybridReconciledDraft{Origin: string(HybridReconciledOrigin), PredecessorID: input.PredecessorID, PredecessorDigest: input.PredecessorDigest, Diffs: diffs, BindingRequired: true}, nil
}

func digestText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
