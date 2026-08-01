package checklistintake

import "testing"

func TestIdentityResolutionRequiresExactCurrentFileFactsAndAppendsCASLeaf(t *testing.T) {
	file := ImportFile{ImportFileID: "file", SHA256: "sha256:file", TerminalManifestDigest: "sha256:manifest", InitialIdentityMatchState: IdentityReviewRequired, RegisterTitle: "Register title", VisibleTitle: "Visible title"}
	ledger := NewIdentityResolutionLedger()
	first, err := ledger.Append(file, IdentityResolutionCommand{SelectedIdentitySource: "REGISTER", SelectedIdentityValue: "Register title", ActorSubjectID: "admin", Reason: "resolve conflict", OperationID: "op-1", IdempotencyKey: "idem-1"})
	if err != nil || first.ResolutionRevision != 1 {
		t.Fatalf("first resolution=%+v err=%v", first, err)
	}
	if _, err := ledger.Append(file, IdentityResolutionCommand{ExpectedPriorLeafID: "wrong", ExpectedPriorDigest: "sha256:wrong", SelectedIdentitySource: "VISIBLE", SelectedIdentityValue: "Visible title", ActorSubjectID: "admin", Reason: "stale", OperationID: "op-2", IdempotencyKey: "idem-2"}); err == nil {
		t.Fatal("stale identity resolution was accepted")
	}
}

func TestIdentityResolutionRejectsNotRegisteredAndUnprovenHumanTranscription(t *testing.T) {
	ledger := NewIdentityResolutionLedger()
	file := ImportFile{ImportFileID: "file", SHA256: "sha256:file", TerminalManifestDigest: "sha256:manifest", InitialIdentityMatchState: IdentityNotRegistered}
	if _, err := ledger.Append(file, IdentityResolutionCommand{SelectedIdentitySource: "HUMAN_TRANSCRIPTION", SelectedIdentityValue: "invented", ActorSubjectID: "admin", Reason: "reason", TranscriptionReason: "receipt", TranscriptionReceiptID: "receipt", OperationID: "op", IdempotencyKey: "idem"}); err == nil {
		t.Fatal("NOT_REGISTERED file was resolved")
	}
	file.InitialIdentityMatchState = IdentityReviewRequired
	if _, err := ledger.Append(file, IdentityResolutionCommand{SelectedIdentitySource: "HUMAN_TRANSCRIPTION", SelectedIdentityValue: "transcribed", ActorSubjectID: "admin", Reason: "reason", OperationID: "op", IdempotencyKey: "idem"}); err == nil {
		t.Fatal("human transcription without a separate receipt was accepted")
	}
}
