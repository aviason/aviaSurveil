package regulatory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

type OfficialSourceClauseRef struct {
	ClauseID                     string `json:"clauseId"`
	SourceID                     string `json:"sourceId"`
	Role                         string `json:"role"`
	SourceVersionID              string `json:"sourceVersionId"`
	SourceHash                   string `json:"sourceHash"`
	Locator                      string `json:"locator,omitempty"`
	Current                      bool   `json:"current"`
	SourceAuthorityAttestationID string `json:"sourceAuthorityAttestationId"`
}

type OfficialSourceQuestion struct {
	QuestionID string                    `json:"questionId"`
	Wording    string                    `json:"wording"`
	Clauses    []OfficialSourceClauseRef `json:"clauses"`
}

type OfficialSourceDraftRequest struct {
	CandidateID     string                   `json:"candidateId"`
	ProviderScopeID string                   `json:"providerScopeId"`
	TargetID        string                   `json:"targetId"`
	InspectionType  string                   `json:"inspectionType"`
	Questions       []OfficialSourceQuestion `json:"questions"`
}

type OfficialSourceDraft struct {
	CandidateID     string                    `json:"candidateId"`
	Origin          string                    `json:"origin"`
	GenerationRunID string                    `json:"generationRunId"`
	BindingDigest   string                    `json:"bindingDigest"`
	SourceChain     []OfficialSourceClauseRef `json:"sourceChain"`
	Questions       []OfficialSourceQuestion  `json:"questions"`
	Blockers        []string                  `json:"blockers"`
}

// SourceAuthorityAttestationResolver is the server-side read boundary for
// append-only source-owner decisions. Callers may provide only an attestation
// identity; they cannot supply an authority boolean or decision projection.
type SourceAuthorityAttestationResolver interface {
	ResolveSourceAuthorityAttestation(id string) (SourceAuthorityDecision, bool)
}

// SourceAuthorityDecisionSet is a deterministic synthetic/test-profile
// resolver. Production code must replace it with a transaction-bound store
// lookup that rereads the current decision leaf under source locks.
type SourceAuthorityDecisionSet map[string]SourceAuthorityDecision

func (set SourceAuthorityDecisionSet) ResolveSourceAuthorityAttestation(id string) (SourceAuthorityDecision, bool) {
	decision, ok := set[id]
	return decision, ok
}

func CreateOfficialSourceDraft(request OfficialSourceDraftRequest, resolver SourceAuthorityAttestationResolver) (OfficialSourceDraft, error) {
	if strings.TrimSpace(request.CandidateID) == "" || strings.TrimSpace(request.ProviderScopeID) == "" || strings.TrimSpace(request.TargetID) == "" || strings.TrimSpace(request.InspectionType) == "" || len(request.Questions) == 0 {
		return OfficialSourceDraft{}, errors.New("official source draft scope is incomplete")
	}
	if resolver == nil {
		return OfficialSourceDraft{}, errors.New("source authority attestation resolver is required")
	}
	chain := make([]OfficialSourceClauseRef, 0)
	for _, question := range request.Questions {
		if strings.TrimSpace(question.QuestionID) == "" || strings.TrimSpace(question.Wording) == "" || len(question.Clauses) == 0 {
			return OfficialSourceDraft{}, errors.New("official question is incomplete")
		}
		hasAuthority, hasProcedure := false, false
		for _, clause := range question.Clauses {
			if strings.TrimSpace(clause.ClauseID) == "" || strings.TrimSpace(clause.SourceID) == "" || strings.TrimSpace(clause.SourceVersionID) == "" || !strings.HasPrefix(clause.SourceHash, "sha256:") || !clause.Current || strings.TrimSpace(clause.SourceAuthorityAttestationID) == "" {
				return OfficialSourceDraft{}, errors.New("official source chain contains an unaccepted or stale link")
			}
			decision, ok := resolver.ResolveSourceAuthorityAttestation(clause.SourceAuthorityAttestationID)
			if !ok || decision.Outcome != "ACCEPT" || decision.SourceID != clause.SourceID || decision.SourceVersionID != clause.SourceVersionID || decision.SourceHash != clause.SourceHash || decision.ChainRole != clause.Role {
				return OfficialSourceDraft{}, errors.New("official source chain attestation is missing, returned, or mismatched")
			}
			switch clause.Role {
			case "REGULATORY_AUTHORITY":
				hasAuthority = true
			case "CONTROLLED_CAA_PROCEDURE":
				hasProcedure = true
			}
			chain = append(chain, clause)
		}
		if !hasAuthority || !hasProcedure {
			return OfficialSourceDraft{}, errors.New("official source chain lacks required authority and controlled procedure links")
		}
	}
	canonical, _ := json.Marshal(struct {
		Scope OfficialSourceDraftRequest `json:"scope"`
		Chain []OfficialSourceClauseRef  `json:"chain"`
	}{request, chain})
	digest := sha256.Sum256(canonical)
	return OfficialSourceDraft{CandidateID: request.CandidateID, Origin: string(RegulatoryTraceOrigin), BindingDigest: "sha256:" + hex.EncodeToString(digest[:]), SourceChain: chain, Questions: request.Questions, Blockers: []string{}}, nil
}
