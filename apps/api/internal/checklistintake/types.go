// Package checklistintake contains the candidate-only, append-only intake
// ledger. It deliberately has no source-authority or publication side effects.
package checklistintake

import (
	"encoding/json"
	"time"
)

const (
	PolicyAGAZipPDFV1                = "AGA_ZIP_PDF_V1"
	OriginExistingChecklistCandidate = "EXISTING_CHECKLIST_CANDIDATE"
)

type ImportBatchStatus string

const (
	ImportBatchReceived          ImportBatchStatus = "RECEIVED"
	ImportBatchProcessing        ImportBatchStatus = "PROCESSING"
	ImportBatchInventoryComplete ImportBatchStatus = "INVENTORY_COMPLETE"
	ImportBatchInventoryFailed   ImportBatchStatus = "INVENTORY_FAILED"
)

type ImportFileIdentityState string

const (
	IdentityRegisterMatched ImportFileIdentityState = "REGISTER_MATCHED"
	IdentityReviewRequired  ImportFileIdentityState = "IDENTITY_REVIEW_REQUIRED"
	IdentityNotRegistered   ImportFileIdentityState = "NOT_REGISTERED"
)

type ImportFileCandidateState string

const (
	CandidateEligible         ImportFileCandidateState = "ELIGIBLE"
	CandidateRequiresIdentity ImportFileCandidateState = "REQUIRES_IDENTITY_RESOLUTION"
	CandidateIneligible       ImportFileCandidateState = "INELIGIBLE"
)

type Phase string

const (
	PhaseArchiveValidate Phase = "ARCHIVE_VALIDATE"
	PhaseObjectFinalize  Phase = "OBJECT_FINALIZE"
	PhaseArchiveScan     Phase = "ARCHIVE_SCAN"
	PhaseEntryValidate   Phase = "ENTRY_VALIDATE"
	PhasePDFScan         Phase = "PDF_SCAN"
	PhasePDFParse        Phase = "PDF_PARSE"
	PhaseRegisterParse   Phase = "REGISTER_PARSE"
	PhaseIdentityMatch   Phase = "IDENTITY_MATCH"
	PhaseScratchCleanup  Phase = "SCRATCH_CLEANUP"
)

type AttemptState string

const (
	AttemptSucceeded AttemptState = "SUCCEEDED"
	AttemptFailed    AttemptState = "FAILED"
	AttemptAbandoned AttemptState = "ABANDONED"
)

type ReceiptOutcome string

const (
	ReceiptSucceeded              ReceiptOutcome = "SUCCEEDED"
	ReceiptFailed                 ReceiptOutcome = "FAILED"
	ReceiptAbandonedExhausted     ReceiptOutcome = "ABANDONED_EXHAUSTED"
	ReceiptNotRunDueToPredecessor ReceiptOutcome = "NOT_RUN_DUE_TO_PREDECESSOR"
)

type ImportBatch struct {
	ImportBatchID            string
	OperationID              string
	IdempotencyKey           string
	ExpectedArchiveSHA       string
	ObservedArchiveSHA       string
	ObservedArchiveByteCount int64
	Status                   ImportBatchStatus
	ManifestDigest           string
	IntakeSafetyEligible     bool
	Reason                   string
	CreatedBySubjectID       string
	CreatedAt                time.Time
	FinalizedAt              *time.Time
}

type ImportFile struct {
	ImportFileID                string
	ImportBatchID               string
	Ordinal                     int
	NormalizedPath              string
	OriginalPath                string
	SHA256                      string
	ByteCount                   int64
	MediaType                   string
	InitialIdentityMatchState   ImportFileIdentityState
	InitialCandidateImportState ImportFileCandidateState
	RegisterFormCode            string
	RegisterTitle               string
	VisibleTitle                string
	TerminalManifestDigest      string
	CreatedAt                   time.Time
}

type PhaseReceipt struct {
	ReceiptID     string
	ImportBatchID string
	ImportFileID  string
	Phase         Phase
	InputDigest   string
	PolicyVersion string
	ResultDigest  string
	Outcome       ReceiptOutcome
	ErrorCode     string
	Payload       json.RawMessage
	CreatedAt     time.Time
}

type RegisterEntry struct {
	RegisterEntryID     string
	ImportBatchID       string
	RegisterFileID      string
	Page                int
	RowNumber           int
	Ordinal             int
	FormCode            string
	TitleText           string
	VersionText         string
	StatusText          string
	MatchedImportFileID string
	MatchState          string
	CreatedAt           time.Time
}

type ObjectIntent struct {
	IntentID       string
	ImportBatchID  string
	ImportFileID   string
	Purpose        string
	ObjectKey      string
	ExpectedSHA256 string
	ExpectedBytes  int64
	State          string
	ObjectVersion  string
	ObservedSHA256 string
	ObservedBytes  int64
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

type Attempt struct {
	AttemptID            string
	AttemptRootID        string
	PredecessorAttemptID string
	Ordinal              int
	Phase                Phase
	ImportBatchID        string
	ImportFileID         string
	InputDigest          string
	PolicyVersion        string
	LeaseOwner           string
	LeaseExpiresAt       *time.Time
	FencingToken         int64
	CreatedAt            time.Time
}

type AttemptEvent struct {
	EventID      string
	AttemptID    string
	State        AttemptState
	ResultDigest string
	ErrorCode    string
	FencingToken int64
	CompletedAt  time.Time
}

type IdentityResolution struct {
	ResolutionID           string
	ResolutionRootID       string
	ResolutionRevision     int64
	SupersedesResolutionID string
	ImportFileID           string
	ExpectedPriorLeafID    string
	ExpectedPriorDigest    string
	ExpectedFileSHA256     string
	ExpectedManifestDigest string
	SelectedIdentitySource string
	SelectedIdentityValue  string
	TranscriptionReason    string
	TranscriptionReceiptID string
	CompetingValues        json.RawMessage
	ActorSubjectID         string
	ActorMembershipID      string
	Reason                 string
	OperationID            string
	IdempotencyKey         string
	SemanticPayloadDigest  string
	CreatedAt              time.Time
}

type ExtractionReviewPacket struct {
	PacketID               string
	ImportBatchID          string
	ImportFileID           string
	TerminalManifestDigest string
	ParserReceiptID        string
	ParserOutputDigest     string
	ParserOutputBytes      int64
	GeneratorPolicyVersion string
	Outcome                string
	ProposalCount          int
	PacketDigest           string
	FailureCode            string
	CreatedBySubjectID     string
	CreatedAt              time.Time
}

type ExtractionProposal struct {
	ProposalID           string
	PacketID             string
	ProposalOrdinal      int
	OriginalText         string
	TextDigest           string
	Page                 int
	Section              string
	RowLocator           string
	RegionLocator        string
	TextSpan             json.RawMessage
	ParserProvenance     json.RawMessage
	ProposedBoundaryKind string
	CreatedAt            time.Time
}

type ExtractionDecisionSet struct {
	DecisionSetID           string
	DecisionSetRootID       string
	Revision                int64
	SupersedesDecisionSetID string
	PacketID                string
	ImportFileID            string
	TerminalManifestDigest  string
	ParserOutputDigest      string
	ExpectedPriorLeafID     string
	ExpectedPriorDigest     string
	ActorSubjectID          string
	Reason                  string
	OperationID             string
	IdempotencyKey          string
	SemanticPayloadDigest   string
	CreatedAt               time.Time
}

type ExtractionDecision struct {
	DecisionID              string
	DecisionSetID           string
	DecisionOrdinal         int
	DecisionKind            string
	ConsumedProposalIDs     []string
	ConsumedProposalDigests []string
	OutputSeedIDs           []string
	OutputPayload           json.RawMessage
	Reason                  string
	CreatedAt               time.Time
}

type ExistingCandidate struct {
	ExistingCandidateID     string
	CandidateRootID         string
	Revision                int64
	SupersedesCandidateID   string
	ContentDigest           string
	ImportBatchID           string
	ImportFileID            string
	ExtractionPacketID      string
	ExtractionDecisionSetID string
	IdentityBasis           string
	ResolutionID            string
	Origin                  string
	SchemaVersion           string
	SourceFileSHA256        string
	FormCode                string
	Title                   string
	QuestionCount           int
	CreatedBySubjectID      string
	Reason                  string
	CreatedAt               time.Time
}

type ExistingCandidateQuestion struct {
	ExistingCandidateID string
	QuestionID          string
	Ordinal             int
	Wording             string
	SourceLocators      json.RawMessage
	OperationalIntent   string
	ExpectedEvidence    string
	ResultHistory       string
	Applicability       string
	ScopeClassification string
	ProvenanceState     string
	CreatedAt           time.Time
}
