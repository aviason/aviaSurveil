package checklistgovernance

import (
	"errors"
	"strings"

	"github.com/aviason/aviaSurveil/internal/regulatory"
)

// AuditPackageEligibilityView is a pure, read-only projection. It is kept
// separate from publication and never creates an Audit package.
type AuditPackageEligibilityView struct {
	Eligible bool     `json:"eligible"`
	Blockers []string `json:"blockers"`
	Digest   string   `json:"decisionDigest"`
}

type AuditPackageEligibilityRequest struct {
	Published              bool
	Current                bool
	Applicable             bool
	TechnicalDecisionCount int
	RequiredOwnerCount     int
	PublishedVersionID     string
}

func EvaluateAuditPackageEligibilityView(input AuditPackageEligibilityRequest) (AuditPackageEligibilityView, error) {
	if strings.TrimSpace(input.PublishedVersionID) == "" || input.TechnicalDecisionCount < 0 || input.RequiredOwnerCount < 0 {
		return AuditPackageEligibilityView{}, errors.New("eligibility request is incomplete")
	}
	result := regulatory.EvaluateAuditPackageEligibility(regulatory.AuditPackageEligibilityInput{
		Published: input.Published, Current: input.Current, Applicable: input.Applicable,
		TechnicalDecisionCount: input.TechnicalDecisionCount, RequiredOwnerCount: input.RequiredOwnerCount,
	})
	digest, err := regulatory.CanonicalSHA256(map[string]any{"publishedVersionId": input.PublishedVersionID, "eligible": result.Eligible, "blockers": result.Blockers})
	if err != nil {
		return AuditPackageEligibilityView{}, err
	}
	return AuditPackageEligibilityView{Eligible: result.Eligible, Blockers: result.Blockers, Digest: digest}, nil
}
