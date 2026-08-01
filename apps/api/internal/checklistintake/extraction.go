package checklistintake

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type ExtractionDecisionAction struct {
	DecisionID              string
	Kind                    string
	ConsumedProposalIDs     []string
	ConsumedProposalDigests []string
	ReplacementText         string
	Reason                  string
}

type CandidateSeed struct {
	SeedID     string
	Wording    string
	TextDigest string
	SourceIDs  []string
	Provenance string
}

type ExtractionDecisionSetResult struct {
	DecisionSet ExtractionDecisionSet
	Actions     []ExtractionDecision
	OutputSeeds []CandidateSeed
}

func BuildExtractionReviewPacket(file ImportFile, parsed ParsedPDF, parserReceiptDigest, policyVersion string) (ExtractionReviewPacket, []ExtractionProposal, error) {
	policy := AGAZipPDFV1()
	if strings.TrimSpace(file.ImportFileID) == "" || strings.TrimSpace(file.TerminalManifestDigest) == "" || strings.TrimSpace(parserReceiptDigest) == "" || policyVersion != "AGA_EXTRACTION_REVIEW_V1" {
		return ExtractionReviewPacket{}, nil, errors.New("packet identity is incomplete")
	}
	if parsed.OutputBytes <= 0 || parsed.OutputBytes > policy.MaxParserOutputBytes {
		return ExtractionReviewPacket{}, nil, ErrArchiveLimit
	}
	lines := strings.Split(strings.ReplaceAll(parsed.Text, "\r\n", "\n"), "\n")
	proposals := make([]ExtractionProposal, 0, len(lines))
	var aggregate int64
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len([]byte(line)) > int(policy.MaxProposalTextBytes) {
			return ExtractionReviewPacket{}, nil, ErrArchiveLimit
		}
		aggregate += int64(len([]byte(line)))
		if aggregate > policy.MaxAggregateProposalBytes {
			return ExtractionReviewPacket{}, nil, ErrArchiveLimit
		}
		textSum := sha256.Sum256([]byte(line))
		textDigest := "sha256:" + hex.EncodeToString(textSum[:])
		ordinal := len(proposals) + 1
		seed := sha256.Sum256([]byte(file.ImportFileID + ":proposal:" + fmt.Sprint(ordinal) + ":" + textDigest))
		proposals = append(proposals, ExtractionProposal{ProposalID: "proposal-" + hex.EncodeToString(seed[:8]), PacketID: "", ProposalOrdinal: ordinal, OriginalText: line, TextDigest: textDigest, Page: 1, TextSpan: json.RawMessage(`{}`), ParserProvenance: json.RawMessage(`{"parserOutputDigest":"` + parsed.OutputDigest + `"}`), ProposedBoundaryKind: "LINE", CreatedAt: time.Time{}})
		if len(proposals) > policy.MaxProposalSpans {
			return ExtractionReviewPacket{}, nil, ErrArchiveLimit
		}
	}
	if len(proposals) == 0 {
		return ExtractionReviewPacket{}, nil, errors.New("parser output contains no bounded proposal text")
	}
	packetSeed := sha256.Sum256([]byte(file.ImportFileID + ":packet:" + file.TerminalManifestDigest + ":" + parserReceiptDigest + ":" + parsed.OutputDigest + ":" + policyVersion))
	packetID := "packet-" + hex.EncodeToString(packetSeed[:8])
	for index := range proposals {
		proposals[index].PacketID = packetID
	}
	packetDigestInput, _ := json.Marshal(struct {
		PacketID, FileID, Manifest, Parser, Output, Policy string
		ProposalDigests                                    []string
	}{packetID, file.ImportFileID, file.TerminalManifestDigest, parserReceiptDigest, parsed.OutputDigest, policyVersion, proposalDigests(proposals)})
	packetDigest := sha256.Sum256(packetDigestInput)
	return ExtractionReviewPacket{PacketID: packetID, ImportBatchID: file.ImportBatchID, ImportFileID: file.ImportFileID, TerminalManifestDigest: file.TerminalManifestDigest, ParserReceiptID: parserReceiptDigest, ParserOutputDigest: parsed.OutputDigest, ParserOutputBytes: parsed.OutputBytes, GeneratorPolicyVersion: policyVersion, Outcome: "READY", ProposalCount: len(proposals), PacketDigest: "sha256:" + hex.EncodeToString(packetDigest[:])}, proposals, nil
}

func ValidateExtractionDecisionSet(proposals []ExtractionProposal, actions []ExtractionDecisionAction) (ExtractionDecisionSetResult, error) {
	if len(proposals) == 0 || len(actions) == 0 {
		return ExtractionDecisionSetResult{}, errors.New("decision set is empty")
	}
	byID := make(map[string]ExtractionProposal, len(proposals))
	for _, proposal := range proposals {
		byID[proposal.ProposalID] = proposal
	}
	consumed := make(map[string]bool, len(proposals))
	validated := make([]ExtractionDecision, 0, len(actions))
	seeds := make([]CandidateSeed, 0, len(proposals))
	for ordinal, action := range actions {
		if strings.TrimSpace(action.Reason) == "" || len(action.ConsumedProposalIDs) == 0 || len(action.ConsumedProposalIDs) != len(action.ConsumedProposalDigests) {
			return ExtractionDecisionSetResult{}, errors.New("decision action is incomplete")
		}
		if action.Kind != "ACCEPT" && action.Kind != "SPLIT" && action.Kind != "MERGE" && action.Kind != "TRANSCRIBE" && action.Kind != "EXCLUDE" {
			return ExtractionDecisionSetResult{}, errors.New("unsupported extraction decision")
		}
		if (action.Kind == "MERGE" && (len(action.ConsumedProposalIDs) < 2 || len(action.ConsumedProposalIDs) > 20)) || (action.Kind != "MERGE" && len(action.ConsumedProposalIDs) != 1) {
			return ExtractionDecisionSetResult{}, errors.New("invalid decision coverage cardinality")
		}
		texts := make([]string, 0, len(action.ConsumedProposalIDs))
		for index, proposalID := range action.ConsumedProposalIDs {
			proposal, exists := byID[proposalID]
			if !exists || consumed[proposalID] || proposal.TextDigest != action.ConsumedProposalDigests[index] {
				return ExtractionDecisionSetResult{}, errors.New("decision proposal digest or coverage mismatch")
			}
			consumed[proposalID] = true
			texts = append(texts, proposal.OriginalText)
		}
		if action.Kind == "MERGE" {
			for index := 1; index < len(action.ConsumedProposalIDs); index++ {
				if byID[action.ConsumedProposalIDs[index]].ProposalOrdinal != byID[action.ConsumedProposalIDs[index-1]].ProposalOrdinal+1 {
					return ExtractionDecisionSetResult{}, errors.New("merge proposals are not adjacent")
				}
			}
		}
		wording := strings.Join(texts, "\n")
		if action.Kind == "TRANSCRIBE" {
			if len([]byte(action.ReplacementText)) == 0 || len([]byte(action.ReplacementText)) > int(AGAZipPDFV1().MaxTranscriptionBytes) {
				return ExtractionDecisionSetResult{}, ErrArchiveLimit
			}
			wording = action.ReplacementText
		}
		decision := ExtractionDecision{DecisionID: action.DecisionID, DecisionSetID: "", DecisionOrdinal: ordinal + 1, DecisionKind: action.Kind, ConsumedProposalIDs: append([]string(nil), action.ConsumedProposalIDs...), ConsumedProposalDigests: append([]string(nil), action.ConsumedProposalDigests...), Reason: action.Reason}
		if action.Kind != "EXCLUDE" {
			seedSum := sha256.Sum256([]byte(action.Kind + ":" + wording))
			seed := CandidateSeed{SeedID: "seed-" + hex.EncodeToString(seedSum[:8]), Wording: wording, TextDigest: "sha256:" + hex.EncodeToString(sha256Bytes([]byte(wording))), SourceIDs: append([]string(nil), action.ConsumedProposalIDs...), Provenance: "SUPPLIED_UNVERIFIED"}
			seeds = append(seeds, seed)
			decision.OutputSeedIDs = []string{seed.SeedID}
		}
		validated = append(validated, decision)
	}
	if len(consumed) != len(proposals) {
		return ExtractionDecisionSetResult{}, errors.New("decision set does not completely cover packet proposals")
	}
	canonical, err := json.Marshal(struct {
		ProposalIDs []string             `json:"proposalIds"`
		Decisions   []ExtractionDecision `json:"decisions"`
	}{proposalIDs(proposals), validated})
	if err != nil {
		return ExtractionDecisionSetResult{}, err
	}
	setSum := sha256.Sum256(canonical)
	return ExtractionDecisionSetResult{DecisionSet: ExtractionDecisionSet{DecisionSetID: "decision-set-" + hex.EncodeToString(setSum[:8]), DecisionSetRootID: "decision-root-" + hex.EncodeToString(setSum[:8]), Revision: 1, SemanticPayloadDigest: "sha256:" + hex.EncodeToString(setSum[:])}, Actions: validated, OutputSeeds: seeds}, nil
}

func proposalIDs(proposals []ExtractionProposal) []string {
	ids := make([]string, 0, len(proposals))
	for _, proposal := range proposals {
		ids = append(ids, proposal.ProposalID)
	}
	return ids
}

func ImportExistingCandidate(file ImportFile, packet ExtractionReviewPacket, decisionSet ExtractionDecisionSetResult, actorSubjectID, reason string) (ExistingCandidate, []ExistingCandidateQuestion, error) {
	if file.InitialCandidateImportState != CandidateEligible || packet.Outcome != "READY" || packet.ImportFileID != file.ImportFileID || packet.TerminalManifestDigest != file.TerminalManifestDigest || strings.TrimSpace(actorSubjectID) == "" || strings.TrimSpace(reason) == "" || len(decisionSet.OutputSeeds) == 0 {
		return ExistingCandidate{}, nil, errors.New("candidate import prerequisites are not satisfied")
	}
	questions := make([]ExistingCandidateQuestion, 0, len(decisionSet.OutputSeeds))
	for ordinal, seed := range decisionSet.OutputSeeds {
		questions = append(questions, ExistingCandidateQuestion{ExistingCandidateID: "", QuestionID: seed.SeedID, Ordinal: ordinal + 1, Wording: seed.Wording, SourceLocators: json.RawMessage(`[]`), ProvenanceState: "SUPPLIED_UNVERIFIED"})
	}
	content, _ := json.Marshal(questions)
	contentSum := sha256.Sum256(content)
	rootSum := sha256.Sum256([]byte(file.ImportFileID + ":candidate:" + hex.EncodeToString(contentSum[:])))
	candidateID := "candidate-" + hex.EncodeToString(rootSum[:8])
	for index := range questions {
		questions[index].ExistingCandidateID = candidateID
	}
	return ExistingCandidate{ExistingCandidateID: candidateID, CandidateRootID: candidateID, Revision: 1, ContentDigest: "sha256:" + hex.EncodeToString(contentSum[:]), ImportBatchID: file.ImportBatchID, ImportFileID: file.ImportFileID, ExtractionPacketID: packet.PacketID, ExtractionDecisionSetID: decisionSet.DecisionSet.DecisionSetID, IdentityBasis: string(file.InitialIdentityMatchState), Origin: OriginExistingChecklistCandidate, SchemaVersion: "AGA_EXISTING_CHECKLIST_CANDIDATE_V1", SourceFileSHA256: file.SHA256, FormCode: file.RegisterFormCode, QuestionCount: len(questions), CreatedBySubjectID: actorSubjectID, Reason: reason}, questions, nil
}

func proposalDigests(proposals []ExtractionProposal) []string {
	output := make([]string, 0, len(proposals))
	for _, proposal := range proposals {
		output = append(output, proposal.TextDigest)
	}
	return output
}

func sha256Bytes(value []byte) []byte {
	digest := sha256.Sum256(value)
	return digest[:]
}

var _ = sort.Slice
