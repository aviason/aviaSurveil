package regulatory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

// ReconciliationField is an immutable, candidate-only explanation of a
// successor Draft. It never asserts regulatory applicability or authority.
type ReconciliationField struct {
	QuestionID string `json:"questionId"`
	Field      string `json:"field"`
	Before     string `json:"before"`
	After      string `json:"after"`
	BeforeHash string `json:"beforeHash"`
	AfterHash  string `json:"afterHash"`
	Outcome    string `json:"outcome"`
}

type ReconciliationResult struct {
	PredecessorID     string                `json:"predecessorId"`
	PredecessorDigest string                `json:"predecessorDigest"`
	Fields            []ReconciliationField `json:"fields"`
	Digest            string                `json:"digest"`
	SourceMapping     string                `json:"sourceMapping"`
}

// ComputeReconciliation compares immutable question snapshots by question ID.
// Missing source links remain explicit gaps; they are never synthesized from
// AGA wording or metadata.
func ComputeReconciliation(predecessorID, predecessorDigest string, before []CandidateDraftQuestion, after []OfficialSourceQuestion) (ReconciliationResult, error) {
	if strings.TrimSpace(predecessorID) == "" || !strings.HasPrefix(predecessorDigest, "sha256:") {
		return ReconciliationResult{}, errors.New("reconciliation predecessor identity is incomplete")
	}
	old := make(map[string]string, len(before))
	for _, question := range before {
		if strings.TrimSpace(question.QuestionID) == "" || old[question.QuestionID] != "" {
			return ReconciliationResult{}, errors.New("candidate question identity is not unique")
		}
		old[question.QuestionID] = question.Wording
	}
	current := make(map[string]string, len(after))
	for _, question := range after {
		if strings.TrimSpace(question.QuestionID) == "" || strings.TrimSpace(question.Wording) == "" || current[question.QuestionID] != "" {
			return ReconciliationResult{}, errors.New("official question identity is not unique or wording is empty")
		}
		current[question.QuestionID] = question.Wording
	}
	ids := make([]string, 0, len(old)+len(current))
	seen := make(map[string]bool, len(old)+len(current))
	for id := range old {
		seen[id] = true
		ids = append(ids, id)
	}
	for id := range current {
		if !seen[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	fields := make([]ReconciliationField, 0, len(ids))
	for _, id := range ids {
		beforeValue, beforeOK := old[id]
		afterValue, afterOK := current[id]
		outcome := "UNCHANGED"
		switch {
		case !beforeOK:
			outcome = "ADDED"
		case !afterOK:
			outcome = "REMOVED"
		case beforeValue != afterValue:
			outcome = "CHANGED"
		}
		fields = append(fields, ReconciliationField{QuestionID: id, Field: "wording", Before: beforeValue, After: afterValue, BeforeHash: textDigest(beforeValue), AfterHash: textDigest(afterValue), Outcome: outcome})
	}
	canonical, err := json.Marshal(fields)
	if err != nil {
		return ReconciliationResult{}, err
	}
	sum := sha256.Sum256(canonical)
	return ReconciliationResult{PredecessorID: predecessorID, PredecessorDigest: predecessorDigest, Fields: fields, Digest: "sha256:" + hex.EncodeToString(sum[:]), SourceMapping: SourceMappingRequired}, nil
}

func textDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}
