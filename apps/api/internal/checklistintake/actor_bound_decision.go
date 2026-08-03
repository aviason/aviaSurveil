package checklistintake

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

const (
	MaxActorBoundBatchForms             = 5
	MaxActorBoundBatchQuestionProposals = 250
	actorBoundDecisionKind              = "CANDIDATE_SCOPE_APPROVAL"
)

var (
	ErrActorBoundInvalidCommand      = errors.New("actor-bound candidate scope command is invalid")
	ErrActorBoundContextUnavailable  = errors.New("actor-bound server context is unavailable")
	ErrActorBoundIdempotencyConflict = errors.New("actor-bound candidate scope idempotency conflict")
	ErrActorBoundStaleRevision       = errors.New("actor-bound candidate scope is stale")
)

// ActorBoundActorContext is the canonical actor fact returned by the
// server-side resolver. Membership/session status is never accepted from the
// candidate command itself.
type ActorBoundActorContext struct {
	Principal        identity.Principal
	MembershipID     string
	MembershipActive bool
	SessionRevoked   bool
	SessionExpiresAt time.Time
}

// ActorBoundScopeContext is the canonical immutable review-packet scope
// returned by the server-side resolver. The resolver must load these facts
// from the accepted packet/register, not from request text.
type ActorBoundScopeContext struct {
	ReviewedPackageSHA256 string
	ScopeDigest           string
	FormCodes             []string
}

// ActorBoundResolver is the trust boundary for actor-bound decisions. A
// production implementation must resolve the authenticated session and
// active CAA Admin membership, then resolve the immutable reviewed package and
// its exact form scope. The ledger refuses to append without this resolver.
type ActorBoundResolver interface {
	ResolveActor(context.Context, identity.Principal, string) (ActorBoundActorContext, error)
	ResolveScope(context.Context, string, string) (ActorBoundScopeContext, error)
}

// ActorBoundCandidateScopeCommand contains only the bounded candidate-scope
// facts and user explanation. The authenticated Principal is passed separately
// to Append from server request context so it cannot be supplied in a body.
type ActorBoundCandidateScopeCommand struct {
	MembershipID              string
	OperationID               string
	IdempotencyKey            string
	ReviewedPackageSHA256     string
	ScopeDigest               string
	FormCodes                 []string
	MaxFormsPerBatch          int
	MaxQuestionProposalsBatch int
	ExpectedPriorLeafID       string
	ExpectedPriorDigest       string
	Reason                    string
}

// ActorBoundCandidateScopeDecision is an immutable, append-only security
// record. The authority booleans are deliberately fixed false in this ledger.
type ActorBoundCandidateScopeDecision struct {
	DecisionID                     string
	DecisionRootID                 string
	Revision                       int64
	SupersedesDecisionID           string
	DecisionKind                   string
	ActorSubjectID                 string
	ActorMembershipID              string
	ActorSessionID                 string
	ActorRole                      identity.Role
	OperationID                    string
	IdempotencyKey                 string
	ReviewedPackageSHA256          string
	ScopeDigest                    string
	FormCodes                      []string
	MaxFormsPerBatch               int
	MaxQuestionProposalsBatch      int
	CandidateOnly                  bool
	SourceMappingAuthorized        bool
	SourceAuthorityAttested        bool
	ApplicabilityAuthorized        bool
	RiskDecisionAuthorized         bool
	FunctionalAssignmentAuthorized bool
	TechnicalApprovalAuthorized    bool
	PublicationAuthorized          bool
	ReleaseAuthorized              bool
	ProductionAuthorized           bool
	StopOnHardError                bool
	PerFileReviewRequired          bool
	Reason                         string
	ExpectedPriorLeafID            string
	ExpectedPriorDigest            string
	SemanticPayloadDigest          string
	CreatedAt                      time.Time
	CommandDigest                  string
}

// ActorBoundCandidateScopeLedger is a candidate-only append-only adapter for
// the local profile. It does not synthesize an Admin, source owner, or
// production decision. A server-owned resolver must verify the injected
// authentication context and immutable reviewed scope before any append.
type ActorBoundCandidateScopeLedger struct {
	mu       sync.RWMutex
	resolver ActorBoundResolver
	current  map[string]ActorBoundCandidateScopeDecision
	byKey    map[string]ActorBoundCandidateScopeDecision
	history  map[string][]ActorBoundCandidateScopeDecision
}

func NewActorBoundCandidateScopeLedger(resolver ActorBoundResolver) *ActorBoundCandidateScopeLedger {
	return &ActorBoundCandidateScopeLedger{
		resolver: resolver,
		current:  make(map[string]ActorBoundCandidateScopeDecision),
		byKey:    make(map[string]ActorBoundCandidateScopeDecision),
		history:  make(map[string][]ActorBoundCandidateScopeDecision),
	}
}

func (ledger *ActorBoundCandidateScopeLedger) Append(ctx context.Context, principal identity.Principal, command ActorBoundCandidateScopeCommand) (ActorBoundCandidateScopeDecision, error) {
	if ledger == nil {
		return ActorBoundCandidateScopeDecision{}, ErrActorBoundContextUnavailable
	}
	if ctx == nil {
		return ActorBoundCandidateScopeDecision{}, fmt.Errorf("%w: request context is required", ErrActorBoundContextUnavailable)
	}
	if err := validateActorBoundPrincipal(principal); err != nil {
		return ActorBoundCandidateScopeDecision{}, err
	}
	formCodes, err := validateActorBoundCandidateScopeCommand(command)
	if err != nil {
		return ActorBoundCandidateScopeDecision{}, err
	}
	if ledger.resolver == nil {
		return ActorBoundCandidateScopeDecision{}, ErrActorBoundContextUnavailable
	}
	actor, err := ledger.resolver.ResolveActor(ctx, principal, command.MembershipID)
	if err != nil {
		return ActorBoundCandidateScopeDecision{}, fmt.Errorf("%w: actor resolver: %v", ErrActorBoundContextUnavailable, err)
	}
	if err := validateResolvedActor(principal, command.MembershipID, actor); err != nil {
		return ActorBoundCandidateScopeDecision{}, err
	}
	scope, err := ledger.resolver.ResolveScope(ctx, command.ReviewedPackageSHA256, command.ScopeDigest)
	if err != nil {
		return ActorBoundCandidateScopeDecision{}, fmt.Errorf("%w: scope resolver: %v", ErrActorBoundContextUnavailable, err)
	}
	resolvedForms, err := validateActorBoundScopeContext(scope)
	if err != nil || !equalActorBoundStrings(formCodes, resolvedForms) || scope.ReviewedPackageSHA256 != command.ReviewedPackageSHA256 || scope.ScopeDigest != command.ScopeDigest {
		return ActorBoundCandidateScopeDecision{}, fmt.Errorf("%w: reviewed package and form scope do not match the immutable server packet", ErrActorBoundContextUnavailable)
	}
	command.FormCodes = formCodes
	commandDigest := actorBoundCommandDigest(command, actor)
	key := command.OperationID + "\x00" + command.IdempotencyKey
	rootID := actorBoundScopeRoot(command)

	ledger.mu.Lock()
	defer ledger.mu.Unlock()
	if existing, ok := ledger.byKey[key]; ok {
		if existing.CommandDigest != commandDigest {
			return ActorBoundCandidateScopeDecision{}, ErrActorBoundIdempotencyConflict
		}
		return cloneActorBoundDecision(existing), nil
	}

	current, hasCurrent := ledger.current[rootID]
	if hasCurrent {
		if command.ExpectedPriorLeafID != current.DecisionID || command.ExpectedPriorDigest != current.SemanticPayloadDigest {
			return ActorBoundCandidateScopeDecision{}, ErrActorBoundStaleRevision
		}
	} else if command.ExpectedPriorLeafID != "" || command.ExpectedPriorDigest != "" {
		return ActorBoundCandidateScopeDecision{}, ErrActorBoundStaleRevision
	}
	revision := int64(1)
	if hasCurrent {
		revision = current.Revision + 1
	}
	semanticDigest := actorBoundSemanticDigest(command, actor, rootID, revision, current)
	decisionID := fmt.Sprintf("actor-decision-%s-%d-%s", strings.TrimPrefix(rootID, "actor-scope-root-"), revision, shortDigest(semanticDigest))
	decision := ActorBoundCandidateScopeDecision{
		DecisionID:                     decisionID,
		DecisionRootID:                 rootID,
		Revision:                       revision,
		DecisionKind:                   actorBoundDecisionKind,
		ActorSubjectID:                 actor.Principal.SubjectID,
		ActorMembershipID:              actor.MembershipID,
		ActorSessionID:                 actor.Principal.SessionID,
		ActorRole:                      actor.Principal.Roles[0],
		OperationID:                    command.OperationID,
		IdempotencyKey:                 command.IdempotencyKey,
		ReviewedPackageSHA256:          command.ReviewedPackageSHA256,
		ScopeDigest:                    command.ScopeDigest,
		FormCodes:                      append([]string(nil), command.FormCodes...),
		MaxFormsPerBatch:               command.MaxFormsPerBatch,
		MaxQuestionProposalsBatch:      command.MaxQuestionProposalsBatch,
		CandidateOnly:                  true,
		SourceMappingAuthorized:        false,
		SourceAuthorityAttested:        false,
		ApplicabilityAuthorized:        false,
		RiskDecisionAuthorized:         false,
		FunctionalAssignmentAuthorized: false,
		TechnicalApprovalAuthorized:    false,
		PublicationAuthorized:          false,
		ReleaseAuthorized:              false,
		ProductionAuthorized:           false,
		StopOnHardError:                true,
		PerFileReviewRequired:          true,
		Reason:                         command.Reason,
		ExpectedPriorLeafID:            command.ExpectedPriorLeafID,
		ExpectedPriorDigest:            command.ExpectedPriorDigest,
		SemanticPayloadDigest:          semanticDigest,
		CreatedAt:                      time.Now().UTC(),
		CommandDigest:                  commandDigest,
	}
	if hasCurrent {
		decision.SupersedesDecisionID = current.DecisionID
	}
	ledger.current[rootID] = decision
	ledger.byKey[key] = decision
	ledger.history[rootID] = append(ledger.history[rootID], decision)
	return cloneActorBoundDecision(decision), nil
}

func (ledger *ActorBoundCandidateScopeLedger) Current(rootID string) (ActorBoundCandidateScopeDecision, bool) {
	if ledger == nil || strings.TrimSpace(rootID) == "" {
		return ActorBoundCandidateScopeDecision{}, false
	}
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()
	decision, ok := ledger.current[rootID]
	return cloneActorBoundDecision(decision), ok
}

// History returns a detached, revision-ordered copy of the append-only chain.
// Callers cannot mutate the ledger by changing the returned decisions.
func (ledger *ActorBoundCandidateScopeLedger) History(rootID string) ([]ActorBoundCandidateScopeDecision, bool) {
	if ledger == nil || strings.TrimSpace(rootID) == "" {
		return nil, false
	}
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()
	revisions, ok := ledger.history[rootID]
	if !ok {
		return nil, false
	}
	result := make([]ActorBoundCandidateScopeDecision, len(revisions))
	for index, revision := range revisions {
		result[index] = cloneActorBoundDecision(revision)
	}
	return result, true
}

func validateActorBoundPrincipal(principal identity.Principal) error {
	if identity.ValidateApplicationAuthority(principal.OrganizationID, principal.Roles) != nil || !principal.HasRole(identity.RoleAdmin) {
		return fmt.Errorf("%w: Admin authority is required", ErrActorBoundInvalidCommand)
	}
	if strings.TrimSpace(principal.SubjectID) == "" {
		return fmt.Errorf("%w: actor subject is required", ErrActorBoundInvalidCommand)
	}
	if strings.TrimSpace(principal.SessionID) == "" {
		return fmt.Errorf("%w: authenticated actor session is required", ErrActorBoundInvalidCommand)
	}
	return nil
}

func validateResolvedActor(principal identity.Principal, requestedMembershipID string, actor ActorBoundActorContext) error {
	if validateActorBoundPrincipal(actor.Principal) != nil || actor.Principal.SubjectID != principal.SubjectID || actor.Principal.OrganizationID != principal.OrganizationID || actor.Principal.SessionID != principal.SessionID || !sameActorBoundRoles(actor.Principal.Roles, principal.Roles) {
		return fmt.Errorf("%w: resolver actor does not match authenticated principal", ErrActorBoundContextUnavailable)
	}
	if strings.TrimSpace(actor.MembershipID) == "" || actor.MembershipID != strings.TrimSpace(actor.MembershipID) || actor.MembershipID != requestedMembershipID || !actor.MembershipActive {
		return fmt.Errorf("%w: active actor membership is required", ErrActorBoundContextUnavailable)
	}
	if actor.SessionRevoked || actor.SessionExpiresAt.IsZero() || !time.Now().UTC().Before(actor.SessionExpiresAt.UTC()) {
		return fmt.Errorf("%w: active actor session is required", ErrActorBoundContextUnavailable)
	}
	return nil
}

func validateActorBoundScopeContext(scope ActorBoundScopeContext) ([]string, error) {
	if !isSHA256Digest(scope.ReviewedPackageSHA256) || !isSHA256Digest(scope.ScopeDigest) {
		return nil, fmt.Errorf("%w: resolver returned non-canonical package or scope digest", ErrActorBoundContextUnavailable)
	}
	return normalizeActorBoundFormCodes(scope.FormCodes)
}

func normalizeActorBoundFormCodes(formCodes []string) ([]string, error) {
	if len(formCodes) == 0 || len(formCodes) > 52 {
		return nil, fmt.Errorf("%w: form scope must contain one through 52 forms", ErrActorBoundInvalidCommand)
	}
	forms := make([]string, 0, len(formCodes))
	seen := make(map[string]struct{}, len(formCodes))
	for _, code := range formCodes {
		code = strings.TrimSpace(code)
		if code == "" {
			return nil, fmt.Errorf("%w: form scope contains an empty code", ErrActorBoundInvalidCommand)
		}
		if _, exists := seen[code]; exists {
			return nil, fmt.Errorf("%w: form scope contains duplicate code", ErrActorBoundInvalidCommand)
		}
		seen[code] = struct{}{}
		forms = append(forms, code)
	}
	sort.Strings(forms)
	return forms, nil
}

func sameActorBoundRoles(left, right []identity.Role) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func equalActorBoundStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func validateActorBoundCandidateScopeCommand(command ActorBoundCandidateScopeCommand) ([]string, error) {
	if strings.TrimSpace(command.MembershipID) == "" {
		return nil, fmt.Errorf("%w: actor membership is required", ErrActorBoundInvalidCommand)
	}
	for field, value := range map[string]string{"operation": command.OperationID, "idempotency": command.IdempotencyKey, "reason": command.Reason} {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("%w: %s is required", ErrActorBoundInvalidCommand, field)
		}
	}
	if !isSHA256Digest(command.ReviewedPackageSHA256) || !isSHA256Digest(command.ScopeDigest) {
		return nil, fmt.Errorf("%w: reviewed package and scope digests must be sha256", ErrActorBoundInvalidCommand)
	}
	forms, err := normalizeActorBoundFormCodes(command.FormCodes)
	if err != nil {
		return nil, err
	}
	if command.MaxFormsPerBatch <= 0 || command.MaxFormsPerBatch > MaxActorBoundBatchForms || command.MaxFormsPerBatch > len(forms) {
		return nil, fmt.Errorf("%w: batch form limit must be between one and %d within scope", ErrActorBoundInvalidCommand, MaxActorBoundBatchForms)
	}
	if command.MaxQuestionProposalsBatch <= 0 || command.MaxQuestionProposalsBatch > MaxActorBoundBatchQuestionProposals {
		return nil, fmt.Errorf("%w: batch question limit must be between one and %d", ErrActorBoundInvalidCommand, MaxActorBoundBatchQuestionProposals)
	}
	if (command.ExpectedPriorLeafID == "") != (command.ExpectedPriorDigest == "") {
		return nil, fmt.Errorf("%w: predecessor leaf and digest must be supplied together", ErrActorBoundInvalidCommand)
	}
	return forms, nil
}

func isSHA256Digest(value string) bool {
	if value != strings.TrimSpace(value) || len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	hexDigest := value[len("sha256:"):]
	decoded, err := hex.DecodeString(hexDigest)
	return err == nil && hex.EncodeToString(decoded) == hexDigest
}

func actorBoundScopeRoot(command ActorBoundCandidateScopeCommand) string {
	canonical := strings.Join([]string{"AGA_ACTOR_BOUND_SCOPE_V1", command.ReviewedPackageSHA256, command.ScopeDigest, strings.Join(command.FormCodes, "\x00")}, "\x00")
	digest := sha256.Sum256([]byte(canonical))
	return "actor-scope-root-" + hex.EncodeToString(digest[:8])
}

func actorBoundCommandDigest(command ActorBoundCandidateScopeCommand, actor ActorBoundActorContext) string {
	canonical, _ := json.Marshal(struct {
		Subject, Membership, Session, Role, Operation, Idempotency, Package, Scope, Reason string
		Forms                                                                              []string
		MaxForms, MaxQuestions                                                             int
	}{actor.Principal.SubjectID, actor.MembershipID, actor.Principal.SessionID, string(actor.Principal.Roles[0]), command.OperationID, command.IdempotencyKey, command.ReviewedPackageSHA256, command.ScopeDigest, command.Reason, command.FormCodes, command.MaxFormsPerBatch, command.MaxQuestionProposalsBatch})
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func actorBoundSemanticDigest(command ActorBoundCandidateScopeCommand, actor ActorBoundActorContext, rootID string, revision int64, predecessor ActorBoundCandidateScopeDecision) string {
	canonical, _ := json.Marshal(struct {
		Root, Subject, Membership, Session, Role, Operation, Package, Scope, Reason, PriorLeaf, PriorDigest string
		Forms                                                                                               []string
		Revision, MaxForms, MaxQuestions                                                                    int64
	}{rootID, actor.Principal.SubjectID, actor.MembershipID, actor.Principal.SessionID, string(actor.Principal.Roles[0]), command.OperationID, command.ReviewedPackageSHA256, command.ScopeDigest, command.Reason, predecessor.DecisionID, predecessor.SemanticPayloadDigest, command.FormCodes, revision, int64(command.MaxFormsPerBatch), int64(command.MaxQuestionProposalsBatch)})
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:])
}

func cloneActorBoundDecision(value ActorBoundCandidateScopeDecision) ActorBoundCandidateScopeDecision {
	value.FormCodes = append([]string(nil), value.FormCodes...)
	return value
}
