package checklistintake

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

type IdentityResolutionCommand struct {
	ExpectedPriorLeafID    string
	ExpectedPriorDigest    string
	SelectedIdentitySource string
	SelectedIdentityValue  string
	TranscriptionReason    string
	TranscriptionReceiptID string
	ActorSubjectID         string
	ActorMembershipID      string
	Reason                 string
	OperationID            string
	IdempotencyKey         string
}

type IdentityResolutionLedger struct {
	mu      sync.Mutex
	current map[string]IdentityResolution
	byKey   map[string]IdentityResolution
}

func NewIdentityResolutionLedger() *IdentityResolutionLedger {
	return &IdentityResolutionLedger{current: make(map[string]IdentityResolution), byKey: make(map[string]IdentityResolution)}
}

func (ledger *IdentityResolutionLedger) Append(file ImportFile, command IdentityResolutionCommand) (IdentityResolution, error) {
	if ledger == nil {
		return IdentityResolution{}, errors.New("identity resolution ledger is not configured")
	}
	if file.InitialIdentityMatchState != IdentityReviewRequired {
		return IdentityResolution{}, errors.New("identity resolution is allowed only for an identity conflict")
	}
	if strings.TrimSpace(file.ImportFileID) == "" || strings.TrimSpace(file.SHA256) == "" || strings.TrimSpace(file.TerminalManifestDigest) == "" || strings.TrimSpace(command.ActorSubjectID) == "" || strings.TrimSpace(command.Reason) == "" || strings.TrimSpace(command.OperationID) == "" || strings.TrimSpace(command.IdempotencyKey) == "" {
		return IdentityResolution{}, errors.New("identity resolution command is incomplete")
	}
	if strings.TrimSpace(command.SelectedIdentityValue) == "" {
		return IdentityResolution{}, errors.New("selected identity is empty")
	}
	switch command.SelectedIdentitySource {
	case "REGISTER":
		if command.SelectedIdentityValue != file.RegisterTitle {
			return IdentityResolution{}, errors.New("selected register identity does not match receipt")
		}
	case "VISIBLE":
		if command.SelectedIdentityValue != file.VisibleTitle {
			return IdentityResolution{}, errors.New("selected visible identity does not match receipt")
		}
	case "HUMAN_TRANSCRIPTION":
		if strings.TrimSpace(command.TranscriptionReason) == "" || strings.TrimSpace(command.TranscriptionReceiptID) == "" {
			return IdentityResolution{}, errors.New("human transcription requires a separate receipt")
		}
	case "PDF_METADATA":
		return IdentityResolution{}, errors.New("PDF metadata is not a trusted human-readable identity")
	default:
		return IdentityResolution{}, errors.New("unsupported identity source")
	}
	key := command.OperationID + "\x00" + command.IdempotencyKey
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	if existing, ok := ledger.byKey[key]; ok {
		if existing.SelectedIdentityValue != command.SelectedIdentityValue || existing.SelectedIdentitySource != command.SelectedIdentitySource {
			return IdentityResolution{}, ErrIdempotencyConflict
		}
		return existing, nil
	}
	current, hasCurrent := ledger.current[file.ImportFileID]
	if hasCurrent {
		if command.ExpectedPriorLeafID != current.ResolutionID || command.ExpectedPriorDigest != current.SemanticPayloadDigest {
			return IdentityResolution{}, ErrStaleRevision
		}
	} else if command.ExpectedPriorLeafID != "" || command.ExpectedPriorDigest != "" {
		return IdentityResolution{}, ErrStaleRevision
	}
	revision := int64(1)
	rootID := "resolution-root-" + shortDigest(file.ImportFileID)
	if hasCurrent {
		revision = current.ResolutionRevision + 1
		rootID = current.ResolutionRootID
	}
	semanticBytes, _ := json.Marshal(struct {
		File, Source, Value, Reason string
		Revision                    int64
	}{file.ImportFileID, command.SelectedIdentitySource, command.SelectedIdentityValue, command.Reason, revision})
	semantic := sha256.Sum256(semanticBytes)
	resolution := IdentityResolution{ResolutionID: fmt.Sprintf("resolution-%s-%d", shortDigest(file.ImportFileID), revision), ResolutionRootID: rootID, ResolutionRevision: revision, ImportFileID: file.ImportFileID, ExpectedPriorLeafID: command.ExpectedPriorLeafID, ExpectedPriorDigest: command.ExpectedPriorDigest, ExpectedFileSHA256: file.SHA256, ExpectedManifestDigest: file.TerminalManifestDigest, SelectedIdentitySource: command.SelectedIdentitySource, SelectedIdentityValue: command.SelectedIdentityValue, TranscriptionReason: command.TranscriptionReason, TranscriptionReceiptID: command.TranscriptionReceiptID, CompetingValues: competingIdentityValues(file), ActorSubjectID: command.ActorSubjectID, ActorMembershipID: command.ActorMembershipID, Reason: command.Reason, OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey, SemanticPayloadDigest: "sha256:" + hex.EncodeToString(semantic[:]), CreatedAt: time.Now().UTC()}
	if hasCurrent {
		resolution.SupersedesResolutionID = current.ResolutionID
	}
	ledger.current[file.ImportFileID] = resolution
	ledger.byKey[key] = resolution
	return resolution, nil
}

func (ledger *IdentityResolutionLedger) Current(fileID string) (IdentityResolution, bool) {
	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	item, ok := ledger.current[fileID]
	return item, ok
}

func competingIdentityValues(file ImportFile) json.RawMessage {
	values := []string{}
	for _, value := range []string{file.RegisterTitle, file.VisibleTitle} {
		if strings.TrimSpace(value) != "" {
			values = append(values, value)
		}
	}
	encoded, _ := json.Marshal(values)
	return encoded
}

func shortDigest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}
