package checklistintake

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

// ExtractionReviewLedger is a small append-only candidate-only adapter used by
// the local profile. It deliberately stores private proposal text separately
// from any auditee projection and never creates a source-authority decision.
type ExtractionReviewLedger struct {
	mu        sync.RWMutex
	packets   map[string]ExtractionReviewPacket
	proposals map[string][]ExtractionProposal
	decisions map[string]ExtractionDecisionSetResult
}

func NewExtractionReviewLedger() *ExtractionReviewLedger {
	return &ExtractionReviewLedger{
		packets:   make(map[string]ExtractionReviewPacket),
		proposals: make(map[string][]ExtractionProposal),
		decisions: make(map[string]ExtractionDecisionSetResult),
	}
}

func (ledger *ExtractionReviewLedger) Put(packet ExtractionReviewPacket, proposals []ExtractionProposal) error {
	if ledger == nil || strings.TrimSpace(packet.PacketID) == "" || packet.Outcome != "READY" || len(proposals) != packet.ProposalCount {
		return errors.New("extraction review packet is incomplete")
	}
	for index, proposal := range proposals {
		if proposal.PacketID != packet.PacketID || proposal.ProposalOrdinal != index+1 || strings.TrimSpace(proposal.OriginalText) == "" {
			return errors.New("extraction proposal order or packet identity is invalid")
		}
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	if existing, ok := ledger.packets[packet.PacketID]; ok {
		if existing.PacketDigest != packet.PacketDigest {
			return ErrIdempotencyConflict
		}
		return nil
	}
	ledger.packets[packet.PacketID] = packet
	ledger.proposals[packet.PacketID] = append([]ExtractionProposal(nil), proposals...)
	return nil
}

func (ledger *ExtractionReviewLedger) AppendDecisionSet(packetID string, proposals []ExtractionProposal, actions []ExtractionDecisionAction) (ExtractionDecisionSetResult, error) {
	if ledger == nil {
		return ExtractionDecisionSetResult{}, errors.New("extraction review ledger is not configured")
	}
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	packet, ok := ledger.packets[packetID]
	if !ok || packet.Outcome != "READY" {
		return ExtractionDecisionSetResult{}, errors.New("extraction review packet is not ready")
	}
	stored := ledger.proposals[packetID]
	if len(stored) != len(proposals) {
		return ExtractionDecisionSetResult{}, errors.New("extraction proposal set does not match packet")
	}
	result, err := ValidateExtractionDecisionSet(proposals, actions)
	if err != nil {
		return ExtractionDecisionSetResult{}, err
	}
	result.DecisionSet.PacketID = packetID
	result.DecisionSet.TerminalManifestDigest = packet.TerminalManifestDigest
	result.DecisionSet.ParserOutputDigest = packet.ParserOutputDigest
	if existing, ok := ledger.decisions[packetID]; ok {
		if existing.DecisionSet.SemanticPayloadDigest != result.DecisionSet.SemanticPayloadDigest {
			return ExtractionDecisionSetResult{}, ErrIdempotencyConflict
		}
		return existing, nil
	}
	ledger.decisions[packetID] = result
	return result, nil
}

func (ledger *ExtractionReviewLedger) Page(packetID string, cursor, limit int) (ExtractionReviewPacket, []ExtractionProposal, bool) {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()
	packet, ok := ledger.packets[packetID]
	if !ok {
		return ExtractionReviewPacket{}, nil, false
	}
	items := append([]ExtractionProposal(nil), ledger.proposals[packetID]...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].ProposalOrdinal < items[j].ProposalOrdinal })
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(items) {
		cursor = len(items)
	}
	if limit <= 0 || limit > len(items)-cursor {
		limit = len(items) - cursor
	}
	return packet, items[cursor : cursor+limit], true
}
