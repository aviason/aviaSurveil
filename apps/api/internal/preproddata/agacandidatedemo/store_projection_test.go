package agacandidatedemo

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSealedPackageProjectionCarriesItsDigestBoundSourceReferenceCount(
	t *testing.T,
) {
	pkg := AcceptedPackage{
		Identity: PackageIdentity{JSONSHA256: "sha256:" + string(make([]byte, 64))},
		SourceCoverage: []SourceProposal{
			{Ref: "SOURCE-A"},
			{Ref: "SOURCE-B"},
		},
	}

	projection := projectionPackage(pkg)
	if projection.SourceReferenceCount != 2 {
		t.Fatalf(
			"sealed package source-reference count = %d, want 2",
			projection.SourceReferenceCount,
		)
	}
}

func TestCandidateProjectionUsesTheFrozenLowerCamelPackageKeys(t *testing.T) {
	form := projectionForm(1, FormCandidate{
		DocumentTitle: "Synthetic title",
		ProposedRisk:  RiskProposal{Band: "PROPOSED_REVIEW_REQUIRED"},
	})
	question := projectionQuestion("FSS-AGA-FORM-SYNTHETIC", 1, QuestionCandidate{
		SourceAuthorityState: "NOT_ATTESTED",
		DecisionState:        "NOT_SUPPLIED",
	})
	encoded, err := json.Marshal([]any{form, question})
	if err != nil {
		t.Fatal(err)
	}
	payload := string(encoded)
	for _, key := range []string{
		`"documentTitle":"Synthetic title"`,
		`"proposedRisk":{"band":"PROPOSED_REVIEW_REQUIRED"`,
		`"sourceAuthorityState":"NOT_ATTESTED"`,
		`"decisionState":"NOT_SUPPLIED"`,
	} {
		if !strings.Contains(payload, key) {
			t.Fatalf("candidate projection omitted frozen key %s: %s", key, payload)
		}
	}
	for _, forbidden := range []string{"DocumentTitle", "ProposedRisk", "SourceAuthorityState", "DecisionState"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("candidate projection emitted Go field name %q: %s", forbidden, payload)
		}
	}
}

func TestFormAndQuestionProjectionsCarryCountsWithoutDuplicatingSourceRows(
	t *testing.T,
) {
	form := FormCandidate{
		FormCode: "FSS-AGA-FORM-SYNTHETIC",
		FormSourceProposals: []SourceProposal{
			{Ref: "SOURCE-A"},
			{Ref: "SOURCE-B"},
		},
		Questions: []QuestionCandidate{{ProposalID: "QUESTION-A"}},
	}
	formProjection := projectionForm(1, form)
	if formProjection.FormSourceProposalCount != 2 ||
		len(formProjection.Form.FormSourceProposals) != 0 ||
		len(formProjection.Form.Questions) != 0 {
		t.Fatalf("sealed form projection retained the wrong source shape: %#v", formProjection)
	}

	question := QuestionCandidate{
		ProposalID: "QUESTION-A",
		SourceProposals: []SourceProposal{
			{Ref: "SOURCE-A"},
			{Ref: "SOURCE-B"},
			{Ref: "SOURCE-C"},
		},
	}
	questionProjection := projectionQuestion(form.FormCode, 1, question)
	if questionProjection.SourceProposalCount != 3 ||
		len(questionProjection.Question.SourceProposals) != 0 {
		t.Fatalf(
			"sealed question projection retained the wrong source shape: %#v",
			questionProjection,
		)
	}
}
