package checklistintake

import "testing"

func TestExtractionReviewLedgerIsAppendOnlyAndPacketBound(t *testing.T) {
	file := ImportFile{ImportFileID: "FILE-048", ImportBatchID: "BATCH-1", TerminalManifestDigest: "sha256:manifest"}
	packet, proposals, err := BuildExtractionReviewPacket(file, ParsedPDF{Text: "Question one\nQuestion two", OutputBytes: 64, OutputDigest: "sha256:parser"}, "receipt-1", "AGA_EXTRACTION_REVIEW_V1")
	if err != nil {
		t.Fatal(err)
	}
	ledger := NewExtractionReviewLedger()
	if err := ledger.Put(packet, proposals); err != nil {
		t.Fatal(err)
	}
	if err := ledger.Put(packet, proposals); err != nil {
		t.Fatalf("identical replay should be stable: %v", err)
	}
	if _, err := ledger.AppendDecisionSet(packet.PacketID, proposals, []ExtractionDecisionAction{{Kind: "ACCEPT", ConsumedProposalIDs: []string{proposals[0].ProposalID}, ConsumedProposalDigests: []string{proposals[0].TextDigest}, Reason: "review"}, {Kind: "ACCEPT", ConsumedProposalIDs: []string{proposals[1].ProposalID}, ConsumedProposalDigests: []string{proposals[1].TextDigest}, Reason: "review"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.AppendDecisionSet(packet.PacketID, proposals, []ExtractionDecisionAction{{Kind: "ACCEPT", ConsumedProposalIDs: []string{proposals[0].ProposalID}, ConsumedProposalDigests: []string{proposals[0].TextDigest}, Reason: "different"}, {Kind: "ACCEPT", ConsumedProposalIDs: []string{proposals[1].ProposalID}, ConsumedProposalDigests: []string{proposals[1].TextDigest}, Reason: "different"}}); err == nil {
		t.Fatal("divergent decision set must not overwrite the current leaf")
	}
}
