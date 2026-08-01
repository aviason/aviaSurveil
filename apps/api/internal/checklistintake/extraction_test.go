package checklistintake

import "testing"

func TestBuildExtractionReviewPacketUsesBoundedOriginalText(t *testing.T) {
	file := ImportFile{ImportFileID: "file", ImportBatchID: "batch", SHA256: "sha256:file", TerminalManifestDigest: "sha256:manifest"}
	parsed := ParsedPDF{PageCount: 2, Text: "first question\nsecond question", TextDigest: "sha256:text", OutputDigest: "sha256:output", OutputBytes: 128}
	packet, proposals, err := BuildExtractionReviewPacket(file, parsed, "sha256:parser", "AGA_EXTRACTION_REVIEW_V1")
	if err != nil || packet.Outcome != "READY" || len(proposals) != 2 || proposals[0].OriginalText != "first question" {
		t.Fatalf("packet=%+v proposals=%+v err=%v", packet, proposals, err)
	}
}

func TestValidateExtractionDecisionSetRequiresCompleteNonOverlappingCoverage(t *testing.T) {
	proposals := []ExtractionProposal{{ProposalID: "p1", ProposalOrdinal: 1, OriginalText: "one", TextDigest: "sha256:one"}, {ProposalID: "p2", ProposalOrdinal: 2, OriginalText: "two", TextDigest: "sha256:two"}}
	valid := []ExtractionDecisionAction{{DecisionID: "d1", Kind: "ACCEPT", ConsumedProposalIDs: []string{"p1"}, ConsumedProposalDigests: []string{"sha256:one"}, Reason: "accept"}, {DecisionID: "d2", Kind: "EXCLUDE", ConsumedProposalIDs: []string{"p2"}, ConsumedProposalDigests: []string{"sha256:two"}, Reason: "not a question"}}
	if _, err := ValidateExtractionDecisionSet(proposals, valid); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateExtractionDecisionSet(proposals, valid[:1]); err == nil {
		t.Fatal("partial proposal coverage was accepted")
	}
}

func TestImportExistingCandidateRemainsCandidateOnly(t *testing.T) {
	file := ImportFile{ImportFileID: "file", ImportBatchID: "batch", SHA256: "sha256:file", TerminalManifestDigest: "sha256:manifest", InitialIdentityMatchState: IdentityRegisterMatched, InitialCandidateImportState: CandidateEligible}
	packet, proposals, err := BuildExtractionReviewPacket(file, ParsedPDF{PageCount: 1, Text: "question", TextDigest: "sha256:text", OutputDigest: "sha256:output", OutputBytes: 32}, "sha256:parser", "AGA_EXTRACTION_REVIEW_V1")
	if err != nil {
		t.Fatal(err)
	}
	decisions := []ExtractionDecisionAction{{DecisionID: "d1", Kind: "ACCEPT", ConsumedProposalIDs: []string{proposals[0].ProposalID}, ConsumedProposalDigests: []string{proposals[0].TextDigest}, Reason: "accept"}}
	decisionSet, err := ValidateExtractionDecisionSet(proposals, decisions)
	if err != nil {
		t.Fatal(err)
	}
	candidate, questions, err := ImportExistingCandidate(file, packet, decisionSet, "admin", "candidate import")
	if err != nil || candidate.SchemaVersion == "" || len(questions) != 1 || candidate.ContentDigest == "" {
		t.Fatalf("candidate=%+v questions=%+v err=%v", candidate, questions, err)
	}
}
