package agademoworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
)

// Store is the narrow workspace data-plane seam. It exposes no canonical
// provider, identity, assignment, audit, finding, CAP, or evidence operation.
type Store interface {
	LoadAndSeal(context.Context, LoadInput) (WorkspaceSealReceipt, error)
	Snapshot(context.Context) (LoadedWorkspace, error)
	ApplyDraftCommand(context.Context, aga.DraftCommand) (aga.Draft, error)
	AppendQuestionVersion(context.Context, AppendQuestionVersionInput) (WorkspaceQuestionVersion, error)
	PutIdempotencyResponse(context.Context, IdempotencyResponse) (IdempotencyResponse, bool, error)
	GetIdempotencyResponse(context.Context, StoredResponseKey) (IdempotencyResponse, bool, error)
	ResetGeneration(context.Context, ResetInput) (Generation, ResetTombstone, error)
}

// RecommendationSnapshotStore is optional so existing classification-only
// doubles remain narrow. Runtime workspace stores implement it before the
// recommendation command is enabled.
type RecommendationSnapshotStore interface {
	PutRecommendationSnapshot(context.Context, RecommendationSnapshot) (RecommendationSnapshot, bool, error)
	GetRecommendationSnapshot(context.Context, string, string) (RecommendationSnapshot, bool, error)
}

type LifecycleStore interface {
	AppendLifecycleEvent(context.Context, LifecycleEvent) (LifecycleEvent, error)
	GetLifecycleEvents(context.Context, string, string) ([]LifecycleEvent, error)
}

// ValidateRecommendationSnapshot is the append-only boundary for the
// recommendation artifact. The command service builds this value from
// server-derived facts; stores still revalidate it so a caller cannot inject a
// reconstructed or mutable question selection through a store double.
func ValidateRecommendationSnapshot(snapshot RecommendationSnapshot) error {
	recommendation := snapshot.Recommendation
	invalid := func(reason string) error { return fmt.Errorf("%w: %s", ErrWorkspaceAppendOnly, reason) }
	if !workspaceIDPattern.MatchString(recommendation.RecommendationID) || recommendation.Revision < 1 || recommendation.OperationID == "" || recommendation.IdempotencyKey == "" || !workspaceIDPattern.MatchString(recommendation.GenerationID) || recommendation.DraftID == "" || recommendation.DraftRevision < 1 || !validDigest(recommendation.DraftContentDigest) || recommendation.TaxonomyVersion == "" || !validDigest(recommendation.TaxonomyDigest) || recommendation.ClassificationRunID == "" || !validDigest(recommendation.ClassificationRunDigest) || !validDigest(recommendation.AggregateDigest) {
		return invalid("recommendation identity or classification pin")
	}
	if recommendation.OrganizationID == "" || recommendation.ProviderScopeRootID == "" || recommendation.ProviderScopeID == "" || recommendation.ProviderScopeVersion < 1 || !validDigest(recommendation.ProviderScopeProfileDigest) || recommendation.ProviderTypeID == "" || recommendation.ProviderTypeCode != "AERODROME_OPERATOR" || recommendation.DepartmentID == "" || recommendation.OrganizationalUnitID == "" {
		return invalid("recommendation provider scope pin")
	}
	if recommendation.TargetID == "" || recommendation.CanonicalTargetKind == "" || recommendation.TargetProfileCode == "" || recommendation.InspectionProfileCode == "" || recommendation.InspectionTypeCode == "" || recommendation.EffectiveAt.IsZero() {
		return invalid("recommendation target pin")
	}
	if recommendation.ReadinessEventID == "" || !validDigest(recommendation.ReadinessEventDigest) || !validDigest(recommendation.Digest) || snapshot.CreatedAt.IsZero() || !validDigest(snapshot.SnapshotDigest) {
		return invalid("recommendation readiness or digest pin")
	}
	recommendationDigest, recommendationDigestErr := aga.DigestExcludingJSONFields("AGA-DETERMINISTIC-RECOMMENDATION-V1", recommendation, "digest")
	if recommendationDigestErr != nil || recommendation.Digest != recommendationDigest {
		return invalid("recommendation digest")
	}
	seen := make(map[string]struct{}, len(recommendation.Items))
	lastSequence := 0
	lastKey := ""
	for _, item := range recommendation.Items {
		if !item.Current || item.DraftDisposition != aga.DispositionInclude || item.RootSequence < 1 || aga.ValidateQuestionRef(item.QuestionRef) != nil || item.QuestionRef.RootSequence != item.RootSequence || aga.ValidateProjection(aga.FrozenTaxonomy(), item.Projection) != nil {
			return invalid("recommendation item")
		}
		key := item.QuestionRef.Key()
		if _, exists := seen[key]; exists || (item.RootSequence < lastSequence || (item.RootSequence == lastSequence && key <= lastKey)) {
			return invalid("recommendation item ordering")
		}
		seen[key] = struct{}{}
		lastSequence, lastKey = item.RootSequence, key
	}
	snapshotDigest, snapshotDigestErr := aga.DigestExcludingJSONFields("AGA-DEMO-RECOMMENDATION-SNAPSHOT-V1", snapshot, "snapshotDigest")
	if snapshotDigestErr != nil || snapshot.SnapshotDigest != snapshotDigest {
		return invalid("recommendation snapshot digest")
	}
	return nil
}

type MemoryStore struct {
	mu                sync.RWMutex
	loaded            LoadedWorkspace
	loadedOnce        bool
	currentGeneration string
	versions          map[string]WorkspaceQuestionVersion
	byRoot            map[string][]string
	responses         map[StoredResponseKey]IdempotencyResponse
	allocator         *aga.SequentialIDAllocator
	resetCount        int
	nonTerminal       bool
	loaderRevoked     bool
	recommendations   map[string]RecommendationSnapshot
	lifecycleEvents   map[string][]LifecycleEvent
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		versions:        make(map[string]WorkspaceQuestionVersion),
		byRoot:          make(map[string][]string),
		responses:       make(map[StoredResponseKey]IdempotencyResponse),
		allocator:       aga.NewSequentialIDAllocator("memory"),
		recommendations: make(map[string]RecommendationSnapshot),
		lifecycleEvents: make(map[string][]LifecycleEvent),
	}
}

func (store *MemoryStore) LoadAndSeal(_ context.Context, input LoadInput) (WorkspaceSealReceipt, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.loadedOnce {
		return WorkspaceSealReceipt{}, ErrWorkspaceSealed
	}
	if err := validateLoadInput(input); err != nil {
		return WorkspaceSealReceipt{}, err
	}
	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	draft, err := aga.NewDraftFromClassification(input.Classification, input.GenerationID)
	if err != nil {
		return WorkspaceSealReceipt{}, fmt.Errorf("create base Draft: %w", err)
	}
	draftRecord := DraftRecord{Draft: draft, CreatedAt: now}
	items := make([]ClassificationItem, 0, len(input.Classification.Items))
	draftItemsByKey := make(map[string]aga.DraftItem, len(draft.Items))
	for _, draftItem := range draft.Items {
		draftItemsByKey[draftItem.QuestionRef.Key()] = draftItem
	}
	for _, item := range input.Classification.Items {
		classificationItem := ClassificationItem{QuestionKey: item.Identity.Key(), Identity: item.Identity, Projection: item.Projection, AgreementConfidence: item.AgreementConfidence, RecommendationState: item.RecommendationState, Governance: item.GovernanceState, ItemSemanticDigest: item.ItemSemanticDigest, CandidateDigest: item.PassOneResultDigest, ChallengeDigest: item.PassTwoResultDigest}
		if draftItem, ok := draftItemsByKey[item.Identity.Key()]; ok {
			classificationItem.DraftAgreementConfidence = draftItem.DraftAgreementConfidence
			classificationItem.DraftRecommendationState = draftItem.RecommendationState
			classificationItem.DraftReviewState = draftItem.ReviewState
			classificationItem.DraftDisposition = draftItem.Disposition
		}
		items = append(items, classificationItem)
	}
	candidate := make([]ClassificationPassRecord, 0, len(input.Classification.CandidateRecords))
	for _, record := range input.Classification.CandidateRecords {
		candidate = append(candidate, passRecord(record))
	}
	challenge := make([]ClassificationPassRecord, 0, len(input.Classification.ChallengeRecords))
	for _, record := range input.Classification.ChallengeRecords {
		challenge = append(challenge, passRecord(record))
	}
	gen := Generation{
		GenerationID: input.GenerationID, State: GenerationActive, ClassificationRunID: input.Classification.ClassificationRunID,
		ClassificationRunDigest: input.Classification.ClassificationRunDigest, TaxonomyVersion: input.Classification.TaxonomyVersion,
		TaxonomyDigest: input.Classification.TaxonomyDigest, FixtureDigest: input.Fixture.ManifestDigest, Revision: 1,
		CreatedAt: now,
	}
	gen.SealDigest = digestValue("AGA-DEMO-WORKSPACE-GENERATION-V1", struct {
		Generation Generation
		Draft      aga.Draft
	}{gen, draft})
	gen, _ = normalizeGenerationSeal(gen)
	run := ClassificationRun{RunID: input.Classification.ClassificationRunID, State: input.Classification.State, TaxonomyVersion: input.Classification.TaxonomyVersion, TaxonomyDigest: input.Classification.TaxonomyDigest, InputDigest: input.Classification.InputDigest, AggregateDigest: input.Classification.AggregateDigest, ClassificationRunDigest: input.Classification.ClassificationRunDigest, Result: cloneJSON(input.Classification), CandidateRecordCount: len(candidate), ChallengeRecordCount: len(challenge), ItemCount: len(items), CreatedAt: now}
	aggregate := digestValue("AGA-DEMO-WORKSPACE-AGGREGATE-V1", struct {
		GenerationID string
		Run          string
		Items        []ClassificationItem
	}{gen.GenerationID, run.ClassificationRunDigest, items})
	seal := WorkspaceSealReceipt{GenerationID: gen.GenerationID, ClassificationRunDigest: run.ClassificationRunDigest, FixtureDigest: input.Fixture.ManifestDigest, WorkspaceAggregateDigest: aggregate, SealedAt: now, LoaderRevoked: false}
	seal.SealDigest = digestValue("AGA-DEMO-WORKSPACE-SEAL-V1", seal)
	store.loaded = LoadedWorkspace{Generation: gen, Taxonomy: input.TaxonomyVersion, Run: run, Items: items, CandidateRecords: candidate, ChallengeRecords: challenge, Draft: draftRecord, Fixture: cloneJSON(input.Fixture), Seal: seal}
	store.currentGeneration = gen.GenerationID
	store.loadedOnce = true
	store.loaderRevoked = false
	return cloneJSON(seal), nil
}

func (store *MemoryStore) Snapshot(_ context.Context) (LoadedWorkspace, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	if !store.loadedOnce {
		return LoadedWorkspace{}, ErrWorkspaceNotSealed
	}
	return cloneJSON(store.loaded), nil
}

func (store *MemoryStore) PutRecommendationSnapshot(_ context.Context, snapshot RecommendationSnapshot) (RecommendationSnapshot, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if !store.loadedOnce {
		return RecommendationSnapshot{}, false, ErrWorkspaceNotSealed
	}
	if snapshot.Recommendation.GenerationID != store.currentGeneration || ValidateRecommendationSnapshot(snapshot) != nil {
		return RecommendationSnapshot{}, false, ErrWorkspaceAppendOnly
	}
	if existing, ok := store.recommendations[snapshot.Recommendation.RecommendationID]; ok {
		if canonical(existing) != canonical(snapshot) {
			return RecommendationSnapshot{}, false, ErrWorkspaceIdempotency
		}
		return cloneJSON(existing), true, nil
	}
	store.recommendations[snapshot.Recommendation.RecommendationID] = cloneJSON(snapshot)
	return cloneJSON(snapshot), false, nil
}

func (store *MemoryStore) GetRecommendationSnapshot(_ context.Context, generationID, recommendationID string) (RecommendationSnapshot, bool, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	if !store.loadedOnce || generationID != store.currentGeneration {
		return RecommendationSnapshot{}, false, ErrWorkspaceNotSealed
	}
	snapshot, ok := store.recommendations[recommendationID]
	if !ok {
		return RecommendationSnapshot{}, false, nil
	}
	return cloneJSON(snapshot), true, nil
}

func (store *MemoryStore) AppendLifecycleEvent(_ context.Context, event LifecycleEvent) (LifecycleEvent, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if !store.loadedOnce {
		return LifecycleEvent{}, ErrWorkspaceNotSealed
	}
	if event.LifecycleID == "" || event.EventID == "" || event.OperationID == "" || event.CommandKey == "" || event.EventType == "" || event.ActorSubjectID == "" || len(event.Payload) == 0 || event.CreatedAt.IsZero() || event.Sequence < 1 || (event.Sequence == 1 && event.PreviousDigest != "") || (event.Sequence > 1 && !validDigest(event.PreviousDigest)) || !validDigest(event.EventDigest) {
		return LifecycleEvent{}, ErrWorkspaceAppendOnly
	}
	var payload struct {
		GenerationID string `json:"generationId"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil || payload.GenerationID != store.currentGeneration {
		return LifecycleEvent{}, ErrWorkspaceGeneration
	}
	current := store.lifecycleEvents[event.LifecycleID]
	if event.Sequence != len(current)+1 {
		return LifecycleEvent{}, ErrWorkspaceCAS
	}
	if len(current) > 0 && event.PreviousDigest != current[len(current)-1].EventDigest {
		return LifecycleEvent{}, ErrWorkspaceCAS
	}
	expected, err := aga.DigestExcludingJSONFields("AGA-DEMO-LIFECYCLE-EVENT-V1", event, "eventDigest")
	if err != nil || event.EventDigest != expected {
		return LifecycleEvent{}, ErrWorkspaceAppendOnly
	}
	store.lifecycleEvents[event.LifecycleID] = append(current, cloneJSON(event))
	return cloneJSON(event), nil
}

func (store *MemoryStore) GetLifecycleEvents(_ context.Context, generationID, lifecycleID string) ([]LifecycleEvent, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	if !store.loadedOnce {
		return nil, ErrWorkspaceNotSealed
	}
	if generationID != store.currentGeneration {
		return nil, ErrWorkspaceGeneration
	}
	events := store.lifecycleEvents[lifecycleID]
	if len(events) == 0 {
		return nil, nil
	}
	return cloneJSON(events), nil
}

func (store *MemoryStore) ApplyDraftCommand(_ context.Context, command aga.DraftCommand) (aga.Draft, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if !store.loadedOnce {
		return aga.Draft{}, ErrWorkspaceNotSealed
	}
	if command.ExpectedGenerationID != store.currentGeneration {
		return aga.Draft{}, ErrWorkspaceGeneration
	}
	updated, err := aga.ApplyDraftCommand(store.loaded.Draft.Draft, command, store.allocator)
	if err != nil {
		return aga.Draft{}, err
	}
	store.loaded.Draft.Draft = cloneJSON(updated)
	store.syncClassificationDraftState(updated)
	return cloneJSON(updated), nil
}

func (store *MemoryStore) syncClassificationDraftState(draft aga.Draft) {
	byKey := make(map[string]aga.DraftItem, len(draft.Items))
	for _, item := range draft.Items {
		byKey[item.QuestionRef.Key()] = item
	}
	for index := range store.loaded.Items {
		item := &store.loaded.Items[index]
		draftItem, ok := byKey[item.QuestionKey]
		if !ok {
			continue
		}
		item.DraftAgreementConfidence = draftItem.DraftAgreementConfidence
		item.DraftRecommendationState = draftItem.RecommendationState
		item.DraftReviewState = draftItem.ReviewState
		item.DraftDisposition = draftItem.Disposition
	}
}

func (store *MemoryStore) AppendQuestionVersion(_ context.Context, input AppendQuestionVersionInput) (WorkspaceQuestionVersion, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if !store.loadedOnce {
		return WorkspaceQuestionVersion{}, ErrWorkspaceNotSealed
	}
	if input.GenerationID != store.currentGeneration {
		return WorkspaceQuestionVersion{}, ErrWorkspaceGeneration
	}
	if input.ActorSubjectID == "" || trimReason(input.ReasonCode) == "" || strings.TrimSpace(input.Body) == "" {
		return WorkspaceQuestionVersion{}, ErrWorkspaceAppendOnly
	}
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	input.Now = input.Now.UTC()
	if input.BodyDigest == "" {
		input.BodyDigest = bodyDigest(input.Body)
	}
	if input.BodyDigest != bodyDigest(input.Body) || !validDigest(input.BodyDigest) {
		return WorkspaceQuestionVersion{}, ErrWorkspaceAppendOnly
	}
	if input.Action != QuestionVersionAdd && input.Action != QuestionVersionReword {
		return WorkspaceQuestionVersion{}, ErrWorkspaceAppendOnly
	}
	if input.Action == QuestionVersionAdd && input.ParentQuestionKey != nil {
		return WorkspaceQuestionVersion{}, ErrWorkspaceAppendOnly
	}
	if input.Action == QuestionVersionReword && input.ParentQuestionKey == nil {
		return WorkspaceQuestionVersion{}, ErrWorkspaceAppendOnly
	}
	parent, err := store.resolveParent(input.ParentQuestionKey)
	if err != nil {
		return WorkspaceQuestionVersion{}, err
	}
	if parent != nil {
		if parent.GenerationID != input.GenerationID {
			return WorkspaceQuestionVersion{}, aga.ErrCrossGenerationParent
		}
		if parent.BodyDigest == input.BodyDigest {
			return WorkspaceQuestionVersion{}, aga.ErrByteIdenticalReword
		}
		if parent.Origin == aga.QuestionOriginWorkspace {
			p := store.versions[parent.VersionID]
			if !p.CurrentLeaf {
				return WorkspaceQuestionVersion{}, aga.ErrNonCurrentQuestion
			}
			if input.RootID == "" {
				input.RootID = p.RootID
			}
			if input.RootSequence == 0 {
				input.RootSequence = p.RootSequence
			}
		}
		if parent.Origin == aga.QuestionOriginBase && input.RootSequence == 0 {
			input.RootSequence = baseRootSequence(parent.Base)
		}
	}
	if input.RootID == "" {
		input.RootID = store.allocator.NextRootID()
	}
	if input.VersionID == "" {
		input.VersionID = store.allocator.NextVersionID()
	}
	if input.ProposalID == "" {
		input.ProposalID = store.allocator.NextProposalID()
	}
	if input.RootSequence == 0 {
		input.RootSequence = len(store.byRoot) + 1
	}
	if !workspaceIDPattern.MatchString(input.RootID) || !workspaceIDPattern.MatchString(input.VersionID) || !workspaceIDPattern.MatchString(input.ProposalID) || input.RootSequence < 1 {
		return WorkspaceQuestionVersion{}, ErrWorkspaceAppendOnly
	}
	if _, exists := store.versions[input.VersionID]; exists {
		return WorkspaceQuestionVersion{}, aga.ErrWorkspaceIdentityAlias
	}
	for _, versionID := range store.byRoot[input.RootID] {
		if old := store.versions[versionID]; old.ProposalID == input.ProposalID || old.RootSequence != input.RootSequence && parent != nil && parent.Origin == aga.QuestionOriginWorkspace {
			return WorkspaceQuestionVersion{}, aga.ErrWorkspaceIdentityAlias
		}
	}
	if parent != nil && parent.Origin == aga.QuestionOriginBase {
		// A Base reword gets a new workspace root, but retains the accepted
		// package order. The parent is stored as a typed Base key only.
		if input.RootID == "" {
			input.RootID = store.allocator.NextRootID()
		}
	}
	version := WorkspaceQuestionVersion{GenerationID: input.GenerationID, RootID: input.RootID, VersionID: input.VersionID, ProposalID: input.ProposalID, RootSequence: input.RootSequence, BodyDigest: input.BodyDigest, Body: input.Body, ParentQuestionKey: cloneJSON(input.ParentQuestionKey), ActorSubjectID: input.ActorSubjectID, CreatedAt: input.Now, ReasonCode: trimReason(input.ReasonCode), CurrentLeaf: true}
	for _, versionID := range store.byRoot[version.RootID] {
		old := store.versions[versionID]
		old.CurrentLeaf = false
		store.versions[versionID] = old
	}
	store.versions[version.VersionID] = version
	store.byRoot[version.RootID] = append(store.byRoot[version.RootID], version.VersionID)
	return cloneJSON(version), nil
}

type resolvedParent struct {
	Origin                                      aga.QuestionOrigin
	GenerationID, RootID, VersionID, BodyDigest string
	Base                                        *aga.BaseIdentity
}

func (store *MemoryStore) resolveParent(parent *aga.ParentQuestionKey) (*resolvedParent, error) {
	if parent == nil {
		return nil, nil
	}
	if parent.Base != nil {
		if err := parent.Base.Validate(); err != nil {
			return nil, aga.ErrParentQuestionKey
		}
		return &resolvedParent{Origin: aga.QuestionOriginBase, Base: parent.Base, BodyDigest: parent.Base.TextDigest, GenerationID: store.currentGeneration}, nil
	}
	if parent.WorkspaceGenerationID == "" || parent.WorkspaceRootID == "" || parent.WorkspaceVersionID == "" || parent.WorkspaceProposalID == "" || parent.WorkspaceRootSequence < 1 || !validDigest(parent.WorkspaceBodyDigest) {
		return nil, aga.ErrParentQuestionKey
	}
	version, ok := store.versions[parent.WorkspaceVersionID]
	if !ok {
		return nil, aga.ErrMissingParent
	}
	if version.GenerationID != parent.WorkspaceGenerationID {
		return nil, aga.ErrCrossGenerationParent
	}
	if version.RootID != parent.WorkspaceRootID || version.ProposalID != parent.WorkspaceProposalID || version.RootSequence != parent.WorkspaceRootSequence || version.BodyDigest != parent.WorkspaceBodyDigest {
		return nil, aga.ErrCrossRootParent
	}
	if !version.CurrentLeaf {
		return nil, aga.ErrNonCurrentQuestion
	}
	return &resolvedParent{Origin: aga.QuestionOriginWorkspace, GenerationID: version.GenerationID, RootID: version.RootID, VersionID: version.VersionID, BodyDigest: version.BodyDigest}, nil
}

func baseRootSequence(identity *aga.BaseIdentity) int {
	if identity == nil {
		return 0
	}
	// Base ordering is deterministic and remains lower than appended roots.
	// The full Draft ordering is reconstructed by the service from the sealed
	// package ordinal; this stable fallback is only used by the memory seam.
	return identity.Ordinal
}

func (store *MemoryStore) PutIdempotencyResponse(_ context.Context, response IdempotencyResponse) (IdempotencyResponse, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	key := StoredResponseKey{GenerationID: response.GenerationID, ActorSubjectID: response.ActorSubjectID, OperationID: response.OperationID, IdempotencyKey: response.IdempotencyKey}
	if err := key.Validate(); err != nil {
		return IdempotencyResponse{}, false, err
	}
	if !validDigest(response.CommandHash) || !validDigest(response.AuthorizationScopeDigest) {
		return IdempotencyResponse{}, false, ErrWorkspaceIdempotency
	}
	if existing, ok := store.responses[key]; ok {
		if existing.CommandHash != response.CommandHash || existing.AuthorizationScopeDigest != response.AuthorizationScopeDigest {
			return IdempotencyResponse{}, false, ErrWorkspaceIdempotency
		}
		return cloneJSON(existing), true, nil
	}
	store.responses[key] = cloneJSON(response)
	return cloneJSON(response), false, nil
}

func (store *MemoryStore) GetIdempotencyResponse(_ context.Context, key StoredResponseKey) (IdempotencyResponse, bool, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	if err := key.Validate(); err != nil {
		return IdempotencyResponse{}, false, err
	}
	response, ok := store.responses[key]
	return cloneJSON(response), ok, nil
}

func (store *MemoryStore) ResetGeneration(_ context.Context, input ResetInput) (Generation, ResetTombstone, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if !store.loadedOnce {
		return Generation{}, ResetTombstone{}, ErrWorkspaceNotSealed
	}
	current := store.loaded.Generation
	if input.ExpectedGenerationID != current.GenerationID || input.ExpectedGenerationRevision != current.Revision || input.ExpectedGenerationSealDigest != current.SealDigest {
		return Generation{}, ResetTombstone{}, ErrWorkspaceCAS
	}
	if trimReason(input.ReasonCode) == "" || input.ActorSubjectID == "" {
		return Generation{}, ResetTombstone{}, ErrWorkspaceAppendOnly
	}
	if store.nonTerminal {
		return Generation{}, ResetTombstone{}, ErrWorkspaceAppendOnly
	}
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	input.Now = input.Now.UTC()
	store.resetCount++
	newID := fmt.Sprintf("aga-ws-generation-reset-%d", store.resetCount)
	newGen := Generation{GenerationID: newID, State: GenerationActive, ClassificationRunID: current.ClassificationRunID, ClassificationRunDigest: current.ClassificationRunDigest, TaxonomyVersion: current.TaxonomyVersion, TaxonomyDigest: current.TaxonomyDigest, FixtureDigest: current.FixtureDigest, Revision: 1, CreatedAt: input.Now, ResetFromGenerationID: current.GenerationID}
	newGen.SealDigest = digestValue("AGA-DEMO-WORKSPACE-GENERATION-V1", newGen)
	tombstone := ResetTombstone{TombstoneID: fmt.Sprintf("aga-ws-tombstone-%d", store.resetCount), FromGenerationID: current.GenerationID, ToGenerationID: newID, ExpectedGenerationID: input.ExpectedGenerationID, ExpectedRevision: input.ExpectedGenerationRevision, ExpectedSealDigest: input.ExpectedGenerationSealDigest, ReasonCode: trimReason(input.ReasonCode), ActorSubjectID: input.ActorSubjectID, CreatedAt: input.Now}
	tombstone.TombstoneDigest = digestValue("AGA-DEMO-WORKSPACE-RESET-TOMBSTONE-V1", tombstone)
	old := current
	old.State = GenerationReset
	store.loaded.Generation = old
	store.loaded.Generation = newGen
	store.currentGeneration = newID
	return cloneJSON(newGen), cloneJSON(tombstone), nil
}

func (store *MemoryStore) SetNonTerminalLifecycle(value bool) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.nonTerminal = value
}
func (store *MemoryStore) LoaderRevoked() bool {
	store.mu.RLock()
	defer store.mu.RUnlock()
	return store.loaderRevoked
}
func (store *MemoryStore) MarkLoaderRevoked() {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.loaderRevoked = true
	store.loaded.Seal.LoaderRevoked = true
}

func validateLoadInput(input LoadInput) error {
	if !workspaceIDPattern.MatchString(input.GenerationID) {
		return fmt.Errorf("%w: generation id", ErrWorkspaceInput)
	}
	result := input.Classification
	if result.State != aga.ClassificationRunSealed || result.ClassificationRunID == "" || result.TaxonomyVersion == "" || !validDigest(result.TaxonomyDigest) || !validDigest(result.InputDigest) || !validDigest(result.AggregateDigest) || !validDigest(result.ClassificationRunDigest) {
		return fmt.Errorf("%w: sealed classification result", ErrWorkspaceInput)
	}
	if len(result.CandidateRecords) != aga.FrozenBaseQuestionCount || len(result.ChallengeRecords) != aga.FrozenBaseQuestionCount || len(result.Items) != aga.FrozenBaseQuestionCount {
		return fmt.Errorf("%w: classification counts", ErrWorkspaceInput)
	}
	if input.TaxonomyVersion.Version == "" || input.TaxonomyVersion.Version != result.TaxonomyVersion || input.TaxonomyVersion.Digest != result.TaxonomyDigest || !input.TaxonomyVersion.Sealed {
		return fmt.Errorf("%w: taxonomy pin", ErrWorkspaceInput)
	}
	if err := input.Fixture.Validate(); err != nil {
		return err
	}
	if input.ExpectedPackageDigest != "" && input.ExpectedPackageDigest != result.RunReceipt.FixedInputDigests.PackageJSONSHA256 {
		return fmt.Errorf("%w: package digest", ErrWorkspaceInput)
	}
	seen := make(map[string]struct{}, len(result.Items))
	for _, item := range result.Items {
		if err := item.Identity.Validate(); err != nil {
			return fmt.Errorf("%w: item identity", ErrWorkspaceInput)
		}
		if _, ok := seen[item.Identity.Key()]; ok {
			return fmt.Errorf("%w: duplicate identity", ErrWorkspaceInput)
		}
		seen[item.Identity.Key()] = struct{}{}
	}
	return nil
}

func normalizeGenerationSeal(generation Generation) (Generation, error) {
	copy := generation
	if copy.SealDigest == "" {
		copy.SealDigest = digestValue("AGA-DEMO-WORKSPACE-GENERATION-V1", copy)
	}
	return copy, nil
}

func passRecord(record aga.PassProposalRecord) ClassificationPassRecord {
	return ClassificationPassRecord{Identity: record.Identity, RunID: record.ClassificationRunID, PassRole: record.PassRole, PassRunID: record.PassRunID, PassResultDigest: record.PassResultDigest, PromptDigest: record.PromptDigest, ModelDescriptorDigest: record.ModelDescriptorDigest, InputDigest: record.InputDigest, ProposalProjection: record.ProposalProjection, RationaleCodes: append([]string(nil), record.RationaleCodes...), ConfidenceEvidence: append([]aga.ConfidenceEvidence(nil), record.ConfidenceEvidence...), SourceRefs: append([]aga.SourceReference(nil), record.SourceRefs...)}
}

func bodyDigest(body string) string {
	hash := sha256.Sum256([]byte(body))
	return "sha256:" + hex.EncodeToString(hash[:])
}

func (store *MemoryStore) CurrentVersions() []WorkspaceQuestionVersion {
	store.mu.RLock()
	defer store.mu.RUnlock()
	output := make([]WorkspaceQuestionVersion, 0, len(store.versions))
	for _, value := range store.versions {
		output = append(output, cloneJSON(value))
	}
	sort.Slice(output, func(i, j int) bool {
		if output[i].RootSequence == output[j].RootSequence {
			return output[i].VersionID < output[j].VersionID
		}
		return output[i].RootSequence < output[j].RootSequence
	})
	return output
}

func (store *MemoryStore) ValidateReferenceRoundTrip(reference aga.QuestionRef) error {
	encoded, err := json.Marshal(reference)
	if err != nil {
		return err
	}
	var decoded aga.QuestionRef
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return err
	}
	if reference.Key() != decoded.Key() {
		return ErrWorkspaceAppendOnly
	}
	return nil
}
