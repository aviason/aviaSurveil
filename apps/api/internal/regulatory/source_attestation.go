package regulatory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

type SourceAuthorityDecisionInput struct {
	SubjectID          string
	MembershipID       string
	SourceID           string
	SourceVersionID    string
	SourceHash         string
	SourceClass        string
	ChainRole          string
	CurrentnessEventID string
	Outcome            string
	At                 time.Time
	Reason             string
}

type SourceAuthorityDecision struct {
	DecisionID            string
	Outcome               string
	SourceID              string
	SourceVersionID       string
	SourceHash            string
	ChainRole             string
	DecisionSubjectDigest string
	CreatedAt             time.Time
}

func AppendSourceAuthorityDecision(assignments []identity.FunctionalAssignment, input SourceAuthorityDecisionInput) (SourceAuthorityDecision, error) {
	if input.Outcome != "ACCEPT" && input.Outcome != "RETURN" || strings.TrimSpace(input.SubjectID) == "" || strings.TrimSpace(input.MembershipID) == "" || strings.TrimSpace(input.SourceID) == "" || strings.TrimSpace(input.SourceVersionID) == "" || !strings.HasPrefix(input.SourceHash, "sha256:") || strings.TrimSpace(input.ChainRole) == "" || strings.TrimSpace(input.Reason) == "" {
		return SourceAuthorityDecision{}, errors.New("source authority decision is incomplete")
	}
	if !identity.AuthorizeFunctionalAssignment(assignments, identity.FunctionalAuthorizationQuery{SubjectID: input.SubjectID, MembershipID: input.MembershipID, Permission: identity.FunctionalPermissionRegulatorySourceOwner, At: input.At, Scope: identity.FunctionalAssignmentScope{Kind: identity.ScopeSourceIdentity, SourceID: input.SourceID, SourceVersionID: input.SourceVersionID, ChainRole: input.ChainRole, DepartmentID: scopeDepartment(assignments, input), OrganizationalUnitID: scopeUnit(assignments, input)}}) {
		return SourceAuthorityDecision{}, errors.New("source-owner assignment does not cover exact source identity")
	}
	canonical, _ := json.Marshal(input)
	sum := sha256.Sum256(canonical)
	return SourceAuthorityDecision{DecisionID: "source-decision-" + hex.EncodeToString(sum[:8]), Outcome: input.Outcome, SourceID: input.SourceID, SourceVersionID: input.SourceVersionID, SourceHash: input.SourceHash, ChainRole: input.ChainRole, DecisionSubjectDigest: "sha256:" + hex.EncodeToString(sum[:]), CreatedAt: input.At}, nil
}

func scopeDepartment(assignments []identity.FunctionalAssignment, input SourceAuthorityDecisionInput) string {
	for _, assignment := range identity.ResolveCurrentFunctionalAssignments(assignments, input.At) {
		if assignment.SubjectID == input.SubjectID && assignment.MembershipID == input.MembershipID && assignment.Permission == identity.FunctionalPermissionRegulatorySourceOwner && assignment.Scope.SourceID == input.SourceID && assignment.Scope.SourceVersionID == input.SourceVersionID && assignment.Scope.ChainRole == input.ChainRole {
			return assignment.Scope.DepartmentID
		}
	}
	return ""
}

func scopeUnit(assignments []identity.FunctionalAssignment, input SourceAuthorityDecisionInput) string {
	for _, assignment := range identity.ResolveCurrentFunctionalAssignments(assignments, input.At) {
		if assignment.SubjectID == input.SubjectID && assignment.MembershipID == input.MembershipID && assignment.Permission == identity.FunctionalPermissionRegulatorySourceOwner && assignment.Scope.SourceID == input.SourceID && assignment.Scope.SourceVersionID == input.SourceVersionID && assignment.Scope.ChainRole == input.ChainRole {
			return assignment.Scope.OrganizationalUnitID
		}
	}
	return ""
}

type MappingAttestationInput struct {
	SubjectID                string
	MembershipID             string
	ReviewedSourceSetID      string
	ReviewedSourceSetVersion int64
	ReviewedSourceSetDigest  string
	ProviderScopeID          string
	TargetID                 string
	InspectionType           string
	DepartmentID             string
	OrganizationalUnitID     string
	CandidateID              string
	CandidateRevision        int64
	CandidateDigest          string
	CompleteChainDigest      string
	Outcome                  string
	At                       time.Time
	Reason                   string
}

type MappingAttestation struct {
	DecisionID            string
	Outcome               string
	CandidateID           string
	CandidateDigest       string
	CompleteChainDigest   string
	DecisionSubjectDigest string
}

func AppendSourceMappingAttestation(assignments []identity.FunctionalAssignment, input MappingAttestationInput) (MappingAttestation, error) {
	if input.Outcome != "ACCEPT" && input.Outcome != "RETURN" || strings.TrimSpace(input.CandidateID) == "" || input.CandidateRevision <= 0 || !strings.HasPrefix(input.CandidateDigest, "sha256:") || strings.TrimSpace(input.CompleteChainDigest) == "" || strings.TrimSpace(input.Reason) == "" {
		return MappingAttestation{}, errors.New("source mapping attestation is incomplete")
	}
	if !identity.AuthorizeFunctionalAssignment(assignments, identity.FunctionalAuthorizationQuery{SubjectID: input.SubjectID, MembershipID: input.MembershipID, Permission: identity.FunctionalPermissionRegulatorySourceOwner, At: input.At, Scope: identity.FunctionalAssignmentScope{Kind: identity.ScopeReviewedSourceSet, ReviewedSourceSetID: input.ReviewedSourceSetID, ReviewedSourceSetVersion: input.ReviewedSourceSetVersion, ReviewedSourceSetDigest: input.ReviewedSourceSetDigest, ProviderScopeID: input.ProviderScopeID, TargetID: input.TargetID, InspectionType: input.InspectionType, DepartmentID: input.DepartmentID, OrganizationalUnitID: input.OrganizationalUnitID}}) {
		return MappingAttestation{}, errors.New("reviewed-source-set assignment does not cover complete candidate mapping")
	}
	canonical, _ := json.Marshal(input)
	sum := sha256.Sum256(canonical)
	return MappingAttestation{DecisionID: "mapping-decision-" + hex.EncodeToString(sum[:8]), Outcome: input.Outcome, CandidateID: input.CandidateID, CandidateDigest: input.CandidateDigest, CompleteChainDigest: input.CompleteChainDigest, DecisionSubjectDigest: "sha256:" + hex.EncodeToString(sum[:])}, nil
}

type AuditPackageEligibilityInput struct {
	Published              bool
	Current                bool
	Applicable             bool
	TechnicalDecisionCount int
	RequiredOwnerCount     int
}

type AuditPackageEligibility struct {
	Eligible bool
	Blockers []string
}

func EvaluateAuditPackageEligibility(input AuditPackageEligibilityInput) AuditPackageEligibility {
	blockers := make([]string, 0, 5)
	if !input.Published {
		blockers = append(blockers, "NOT_PUBLISHED")
	}
	if !input.Current {
		blockers = append(blockers, "SOURCE_NOT_CURRENT")
	}
	if !input.Applicable {
		blockers = append(blockers, "NOT_APPLICABLE")
	}
	if input.RequiredOwnerCount == 0 || input.TechnicalDecisionCount < input.RequiredOwnerCount {
		blockers = append(blockers, "TECHNICAL_APPROVAL_REQUIRED")
	}
	return AuditPackageEligibility{Eligible: len(blockers) == 0, Blockers: blockers}
}
