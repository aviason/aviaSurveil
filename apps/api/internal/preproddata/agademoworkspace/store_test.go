package agademoworkspace

import (
	"context"
	"errors"
	"testing"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
)

func TestWorkspaceCommandsAreAppendOnly(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.AppendQuestionVersion(context.Background(), AppendQuestionVersionInput{GenerationID: "aga-ws-generation-missing", Body: "candidate", ActorSubjectID: "actor", ReasonCode: "REWORD_REVIEW", Action: QuestionVersionAdd})
	if !errors.Is(err, ErrWorkspaceNotSealed) {
		t.Fatalf("append before seal error = %v", err)
	}
}

func TestWorkspaceQuestionReferencesRoundTripExactly(t *testing.T) {
	store := NewMemoryStore()
	if err := store.ValidateReferenceRoundTrip(agaBaseReferenceForTest()); err != nil {
		t.Fatal(err)
	}
}

func TestWorkspaceQuestionIdentityConstraintsAreClosed(t *testing.T) {
	if err := (StoredResponseKey{}).Validate(); err == nil {
		t.Fatal("empty idempotency key accepted")
	}
}

func TestWorkspaceResetIsForwardOnly(t *testing.T) {
	store := NewMemoryStore()
	if _, _, err := store.ResetGeneration(context.Background(), ResetInput{}); !errors.Is(err, ErrWorkspaceNotSealed) {
		t.Fatalf("reset before seal error = %v", err)
	}
}

func agaBaseReferenceForTest() aga.QuestionRef {
	return aga.BaseQuestionReference(aga.BaseIdentity{PackageVersion: aga.FrozenPackageVersion, PackageJSONSHA256: aga.FrozenPackageJSONSHA256, FormCode: "FSS-AGA-FORM-001", ProposalID: "synthetic-proposal", Ordinal: 1, TextDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000"})
}
