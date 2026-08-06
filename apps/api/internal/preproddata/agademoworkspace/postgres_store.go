package agademoworkspace

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/jackc/pgx/v5"
)

// PostgresStore is the runtime/loader seam for the sibling schema. The type
// accepts only the dedicated workspace pool; callers cannot pass the normal
// API or accepted-overlay reader without failing construction.
type PostgresStore struct {
	pool    *database.Pool
	command bool

	runtimeMu                sync.Mutex
	runtimeClassification    aga.ClassificationResult
	runtimeClassificationID  string
	runtimeBaseWorkspace     LoadedWorkspace
	runtimeBaseReady         bool
	runtimeBaseDraft         aga.Draft
	runtimeBaseGeneration    string
	runtimeCurrentDraft      aga.Draft
	runtimeCurrentGeneration string
	runtimeCurrentRevision   int
	runtimeCurrentDigest     string
	runtimeCurrentReady      bool
}

func NewPostgresReader(pool *database.Pool) (*PostgresStore, error) {
	if pool == nil {
		return nil, fmt.Errorf("workspace reader PostgreSQL pool is required")
	}
	return &PostgresStore{pool: pool}, nil
}

func NewPostgresCommandStore(pool *database.Pool) (*PostgresStore, error) {
	if pool == nil {
		return nil, fmt.Errorf("workspace command PostgreSQL pool is required")
	}
	return &PostgresStore{pool: pool, command: true}, nil
}

func (store *PostgresStore) Preflight(ctx context.Context) error {
	if store == nil || store.pool == nil {
		return fmt.Errorf("workspace PostgreSQL store is required")
	}
	var schemaCount, tableCount int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM pg_namespace WHERE nspname = $1`, WorkspaceSchemaName).Scan(&schemaCount); err != nil {
		return fmt.Errorf("inspect workspace schema: %w", err)
	}
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM pg_tables WHERE schemaname = $1`, WorkspaceSchemaName).Scan(&tableCount); err != nil {
		return fmt.Errorf("inspect workspace tables: %w", err)
	}
	if schemaCount != 1 || tableCount < len(WorkspaceSchemaObjectNames()) {
		return fmt.Errorf("workspace schema is absent or incomplete")
	}
	return nil
}

func (store *PostgresStore) LoadAndSeal(ctx context.Context, input LoadInput) (WorkspaceSealReceipt, error) {
	if store == nil || store.pool == nil || !store.command {
		return WorkspaceSealReceipt{}, fmt.Errorf("workspace loader requires command-capable store")
	}
	if err := validateLoadInput(input); err != nil {
		return WorkspaceSealReceipt{}, err
	}
	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	draft, err := agaNewDraft(input.Classification, input.GenerationID)
	if err != nil {
		return WorkspaceSealReceipt{}, fmt.Errorf("create base draft: %w", err)
	}
	draftItemsByKey := make(map[string]aga.DraftItem, len(draft.Items))
	for _, draftItem := range draft.Items {
		draftItemsByKey[draftItem.QuestionRef.Key()] = draftItem
	}
	items := make([]ClassificationItem, 0, len(input.Classification.Items))
	for _, item := range input.Classification.Items {
		questionKey := aga.BaseQuestionReference(item.Identity).Key()
		classificationItem := ClassificationItem{
			QuestionKey: questionKey, Identity: item.Identity, Projection: item.Projection,
			AgreementConfidence: item.AgreementConfidence, RecommendationState: item.RecommendationState,
			Governance: item.GovernanceState, ItemSemanticDigest: item.ItemSemanticDigest,
			CandidateDigest: item.PassOneResultDigest, ChallengeDigest: item.PassTwoResultDigest,
		}
		if draftItem, ok := draftItemsByKey[questionKey]; ok {
			classificationItem.DraftAgreementConfidence = draftItem.DraftAgreementConfidence
			classificationItem.DraftRecommendationState = draftItem.RecommendationState
			classificationItem.DraftReviewState = draftItem.ReviewState
			classificationItem.DraftDisposition = draftItem.Disposition
		}
		items = append(items, classificationItem)
	}
	aggregate := digestValue("AGA-DEMO-WORKSPACE-AGGREGATE-V1", struct {
		GenerationID string
		Run          string
		Items        []ClassificationItem
	}{input.GenerationID, input.Classification.ClassificationRunDigest, items})
	seal := WorkspaceSealReceipt{GenerationID: input.GenerationID, ClassificationRunDigest: input.Classification.ClassificationRunDigest, FixtureDigest: input.Fixture.ManifestDigest, WorkspaceAggregateDigest: aggregate, SealedAt: now, LoaderRevoked: false}
	seal.SealDigest = digestValue("AGA-DEMO-WORKSPACE-SEAL-V1", seal)
	generation := Generation{GenerationID: input.GenerationID, State: GenerationActive, ClassificationRunID: input.Classification.ClassificationRunID, ClassificationRunDigest: input.Classification.ClassificationRunDigest, TaxonomyVersion: input.Classification.TaxonomyVersion, TaxonomyDigest: input.Classification.TaxonomyDigest, FixtureDigest: input.Fixture.ManifestDigest, Revision: 1, SealDigest: seal.SealDigest, CreatedAt: now}
	generation.SealDigest = digestValue("AGA-DEMO-WORKSPACE-GENERATION-V1", struct {
		Generation Generation
		Draft      aga.Draft
	}{generation, draft})
	classificationPassRecords := make([]map[string]any, 0, len(input.Classification.CandidateRecords)+len(input.Classification.ChallengeRecords))
	for _, record := range append(append([]aga.PassProposalRecord(nil), input.Classification.CandidateRecords...), input.Classification.ChallengeRecords...) {
		pass := passRecord(record)
		classificationPassRecords = append(classificationPassRecords, map[string]any{
			"classification_run_id": pass.RunID, "pass_role": pass.PassRole, "identity_key": pass.Identity.Key(),
			"pass_run_id": pass.PassRunID, "pass_result_digest": pass.PassResultDigest,
			"payload": pass, "canonical_payload": canonical(pass), "row_digest": digestJSON(pass),
		})
	}
	providerScopes, providerTargets := workspaceProviderRows(input.GenerationID, now)
	payload := map[string]any{
		"generations": []any{map[string]any{
			"generation_id": generation.GenerationID, "state": generation.State,
			"classification_run_id": generation.ClassificationRunID, "classification_run_digest": generation.ClassificationRunDigest,
			"taxonomy_version": generation.TaxonomyVersion, "taxonomy_digest": generation.TaxonomyDigest,
			"fixture_digest": generation.FixtureDigest, "revision": generation.Revision, "seal_digest": generation.SealDigest,
			"created_at": generation.CreatedAt,
		}},
		"taxonomyVersions": []any{map[string]any{
			"taxonomy_version": input.TaxonomyVersion.Version, "taxonomy_digest": input.TaxonomyVersion.Digest,
			"package_digest": input.TaxonomyVersion.PackageDigest, "published_at": input.TaxonomyVersion.PublishedAt, "sealed": input.TaxonomyVersion.Sealed,
		}},
		"classificationRuns": []any{map[string]any{
			"classification_run_id": input.Classification.ClassificationRunID, "state": input.Classification.State,
			"taxonomy_version": input.Classification.TaxonomyVersion, "taxonomy_digest": input.Classification.TaxonomyDigest,
			"input_digest": input.Classification.InputDigest, "aggregate_digest": input.Classification.AggregateDigest,
			"classification_run_digest": input.Classification.ClassificationRunDigest,
			"candidate_record_count":    len(input.Classification.CandidateRecords), "challenge_record_count": len(input.Classification.ChallengeRecords),
			"item_count": len(items), "payload": input.Classification, "created_at": now,
		}},
		"classificationPassRecords": classificationPassRecords,
		"classificationItems":       workspaceClassificationItemRows(input.Classification.ClassificationRunID, items),
		"drafts": []any{map[string]any{
			"generation_id": input.GenerationID, "draft_id": draft.DraftID, "revision": draft.Revision,
			"content_digest": draft.ContentDigest, "state": draft.State, "payload": draft, "created_at": now,
			"canonical_payload": canonical(draft), "row_digest": digestJSON(draft),
		}},
		"authorityBindings": workspaceAuthorityBindingRows(input.GenerationID, input.Fixture),
		"providerScopes":    providerScopes,
		"providerTargets":   providerTargets,
		"workspaceSeals": []any{map[string]any{
			"generation_id": seal.GenerationID, "classification_run_digest": seal.ClassificationRunDigest,
			"fixture_digest": seal.FixtureDigest, "workspace_aggregate_digest": seal.WorkspaceAggregateDigest,
			"seal_digest": seal.SealDigest, "sealed_at": seal.SealedAt, "loader_revoked": seal.LoaderRevoked,
			"fixture_payload": input.Fixture,
		}},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return WorkspaceSealReceipt{}, fmt.Errorf("encode workspace load payload: %w", err)
	}
	var status json.RawMessage
	err = store.pool.QueryRow(ctx, `SELECT preprod_aga_demo_workspace.workspace_load($1::jsonb)`, encoded).Scan(&status)
	if err != nil {
		return WorkspaceSealReceipt{}, err
	}
	return seal, nil
}

func (store *PostgresStore) Snapshot(ctx context.Context) (LoadedWorkspace, error) {
	if store == nil || store.pool == nil {
		return LoadedWorkspace{}, fmt.Errorf("workspace PostgreSQL store is required")
	}
	store.runtimeMu.Lock()
	if !store.command && store.runtimeBaseReady {
		cached := store.runtimeBaseWorkspace
		store.runtimeMu.Unlock()
		return cached, nil
	}
	store.runtimeMu.Unlock()
	var payload []byte
	if err := store.pool.QueryRow(ctx, `SELECT preprod_aga_demo_workspace.workspace_query('{}'::jsonb)`).Scan(&payload); err != nil {
		return LoadedWorkspace{}, fmt.Errorf("read workspace projection: %w", err)
	}
	var workspace LoadedWorkspace
	if err := json.Unmarshal(payload, &workspace); err != nil {
		return LoadedWorkspace{}, fmt.Errorf("decode workspace projection: %w", err)
	}
	if workspace.Seal.GenerationID == "" {
		return LoadedWorkspace{}, ErrWorkspaceNotSealed
	}
	store.runtimeMu.Lock()
	store.runtimeClassification = workspace.Run.Result
	store.runtimeClassificationID = workspace.Run.Result.ClassificationRunID
	store.runtimeBaseWorkspace = workspace
	store.runtimeBaseReady = true
	store.runtimeMu.Unlock()
	return workspace, nil
}

// UpdateCachedReaderDraft advances the reader's in-process projection after a
// command has committed a new append-only Draft. The immutable sealed items
// are copied before their transient Draft metadata is refreshed, so a query
// already holding the previous value remains stable and no second full
// workspace snapshot is decoded for every page request.
func (store *PostgresStore) UpdateCachedReaderDraft(generationID string, draft aga.Draft) {
	if store == nil || store.pool == nil || store.command {
		return
	}
	store.runtimeMu.Lock()
	defer store.runtimeMu.Unlock()
	if !store.runtimeBaseReady || store.runtimeBaseWorkspace.Generation.GenerationID != generationID {
		return
	}
	next := store.runtimeBaseWorkspace
	next.Items = mergeDraftMetadata(store.runtimeBaseWorkspace.Items, draft)
	next.Draft.Draft = draft
	store.runtimeBaseWorkspace = next
}

// InvalidateCachedReaderSnapshot is used after an explicit generation reset;
// the next authorized read must reload the new immutable generation.
func (store *PostgresStore) InvalidateCachedReaderSnapshot() {
	if store == nil || store.pool == nil || store.command {
		return
	}
	store.runtimeMu.Lock()
	store.runtimeBaseReady = false
	store.runtimeMu.Unlock()
}

func (store *PostgresStore) ListCurrentWorkspaceQuestionVersions(ctx context.Context, generationID string) ([]WorkspaceQuestionVersion, error) {
	if store == nil || store.pool == nil || strings.TrimSpace(generationID) == "" {
		return nil, fmt.Errorf("workspace question version reader requires store and generation")
	}
	var payload []byte
	if err := queryWorkspaceJSON(ctx, store.pool, map[string]any{
		"operation":    "GET_CURRENT_WORKSPACE_QUESTION_VERSIONS",
		"generationId": generationID,
	}, &payload); err != nil {
		return nil, err
	}
	if len(payload) == 0 || string(payload) == "null" {
		return []WorkspaceQuestionVersion{}, nil
	}
	var versions []WorkspaceQuestionVersion
	if err := json.Unmarshal(payload, &versions); err != nil {
		return nil, fmt.Errorf("decode workspace question versions: %w", err)
	}
	return versions, nil
}

// SnapshotForRuntimeCommand reads only the mutable command state and reuses
// the already-validated immutable classification run. The normal Snapshot
// projection intentionally remains complete for readers and evidence; this
// narrow projection prevents every one of the many append-only Draft commands
// from rebuilding the 1,310-item pass graph from PostgreSQL.
func (store *PostgresStore) SnapshotForRuntimeCommand(ctx context.Context) (LoadedWorkspace, error) {
	if store == nil || store.pool == nil || !store.command {
		return LoadedWorkspace{}, fmt.Errorf("workspace runtime command store is required")
	}
	store.runtimeMu.Lock()
	classification := store.runtimeClassification
	classificationID := store.runtimeClassificationID
	baseWorkspace := store.runtimeBaseWorkspace
	baseReady := store.runtimeBaseReady
	store.runtimeMu.Unlock()
	if classificationID == "" {
		workspace, err := store.Snapshot(ctx)
		if err != nil {
			return LoadedWorkspace{}, err
		}
		classification = workspace.Run.Result
		classificationID = workspace.Run.Result.ClassificationRunID
	}
	var meta struct {
		Generation struct {
			GenerationID            string `json:"generationId"`
			State                   string `json:"state"`
			ClassificationRunID     string `json:"classificationRunId"`
			ClassificationRunDigest string `json:"classificationRunDigest"`
			TaxonomyVersion         string `json:"taxonomyVersion"`
			TaxonomyDigest          string `json:"taxonomyDigest"`
			FixtureDigest           string `json:"fixtureDigest"`
			Revision                int    `json:"revision"`
			SealDigest              string `json:"sealDigest"`
		} `json:"generation"`
		Draft struct {
			DraftID       string `json:"draftId"`
			Revision      int    `json:"revision"`
			ContentDigest string `json:"contentDigest"`
		} `json:"draft"`
	}
	if err := queryWorkspaceJSON(ctx, store.pool, map[string]any{"operation": "RUNTIME_COMMAND_META"}, &meta); err != nil {
		return LoadedWorkspace{}, fmt.Errorf("read runtime command projection: %w", err)
	}
	if meta.Generation.GenerationID == "" || meta.Draft.DraftID == "" || meta.Draft.Revision < 1 || meta.Draft.ContentDigest == "" {
		return LoadedWorkspace{}, ErrWorkspaceNotSealed
	}
	generation := Generation{GenerationID: meta.Generation.GenerationID, State: meta.Generation.State, ClassificationRunID: meta.Generation.ClassificationRunID, ClassificationRunDigest: meta.Generation.ClassificationRunDigest, TaxonomyVersion: meta.Generation.TaxonomyVersion, TaxonomyDigest: meta.Generation.TaxonomyDigest, FixtureDigest: meta.Generation.FixtureDigest, Revision: meta.Generation.Revision, SealDigest: meta.Generation.SealDigest}
	store.runtimeMu.Lock()
	if store.runtimeCurrentReady && store.runtimeCurrentGeneration == generation.GenerationID && store.runtimeCurrentRevision == meta.Draft.Revision && store.runtimeCurrentDigest == meta.Draft.ContentDigest && store.runtimeClassificationID == classificationID && baseReady && baseWorkspace.Generation.GenerationID == generation.GenerationID {
		current := store.runtimeCurrentDraft
		store.runtimeMu.Unlock()
		return mergeRuntimeWorkspace(baseWorkspace, generation, classification, current, baseWorkspace.Seal), nil
	}
	store.runtimeMu.Unlock()
	if generation.ClassificationRunID != classificationID || generation.ClassificationRunDigest != classification.ClassificationRunDigest {
		workspace, err := store.Snapshot(ctx)
		if err != nil {
			return LoadedWorkspace{}, err
		}
		classification = workspace.Run.Result
		classificationID = workspace.Run.Result.ClassificationRunID
		generation = workspace.Generation
		baseWorkspace = workspace
		baseReady = true
	}
	if !baseReady || baseWorkspace.Generation.GenerationID != generation.GenerationID {
		workspace, err := store.Snapshot(ctx)
		if err != nil {
			return LoadedWorkspace{}, err
		}
		classification = workspace.Run.Result
		classificationID = workspace.Run.Result.ClassificationRunID
		generation = workspace.Generation
		baseWorkspace = workspace
		baseReady = true
	}
	store.runtimeMu.Lock()
	if store.runtimeBaseGeneration != generation.GenerationID || store.runtimeClassificationID != classificationID {
		base, err := aga.NewDraftFromClassification(classification, generation.GenerationID)
		if err != nil {
			store.runtimeMu.Unlock()
			return LoadedWorkspace{}, fmt.Errorf("create runtime sealed draft context: %w", err)
		}
		store.runtimeBaseDraft = base
		store.runtimeBaseGeneration = generation.GenerationID
		store.runtimeClassification = classification
		store.runtimeClassificationID = classificationID
	}
	base := store.runtimeBaseDraft
	store.runtimeMu.Unlock()
	var state struct {
		Generation Generation           `json:"generation"`
		Draft      DraftRecord          `json:"draft"`
		Seal       WorkspaceSealReceipt `json:"seal"`
	}
	if err := queryWorkspaceJSON(ctx, store.pool, map[string]any{"operation": "RUNTIME_COMMAND_STATE"}, &state); err != nil {
		return LoadedWorkspace{}, fmt.Errorf("read runtime command state: %w", err)
	}
	hydrated, err := aga.HydrateDraftForRuntimeFromSealedDraft(state.Draft.Draft, base)
	if err != nil {
		return LoadedWorkspace{}, fmt.Errorf("hydrate runtime draft context: %w", err)
	}
	state.Draft.Draft = hydrated
	store.runtimeMu.Lock()
	store.runtimeCurrentDraft = hydrated
	store.runtimeCurrentGeneration = state.Generation.GenerationID
	store.runtimeCurrentRevision = hydrated.Revision
	store.runtimeCurrentDigest = hydrated.ContentDigest
	store.runtimeCurrentReady = true
	store.runtimeMu.Unlock()
	return mergeRuntimeWorkspace(baseWorkspace, state.Generation, classification, hydrated, state.Seal), nil
}

func mergeRuntimeWorkspace(base LoadedWorkspace, generation Generation, classification aga.ClassificationResult, draft aga.Draft, seal WorkspaceSealReceipt) LoadedWorkspace {
	base.Generation = generation
	// Runtime commands reuse the immutable sealed item projection for identity,
	// governance, and classification facts. Overlay the append-only Draft
	// metadata before filters consume it; otherwise a current-disposition
	// filter would inspect the sealed base state and silently miss items that a
	// previous batch command changed.
	base.Items = mergeDraftMetadata(base.Items, draft)
	// Preserve the complete sealed run projection exactly as the normal
	// workspace query exposes it. Only the nested immutable result is refreshed
	// from the validated runtime cache; in particular, do not synthesize a new
	// RunID spelling that would change the setup digest contract.
	base.Run.Result = classification
	base.Draft = DraftRecord{Draft: draft, CreatedAt: base.Draft.CreatedAt}
	if seal.GenerationID != "" {
		base.Seal = seal
	}
	return base
}

func mergeDraftMetadata(items []ClassificationItem, draft aga.Draft) []ClassificationItem {
	merged := append([]ClassificationItem(nil), items...)
	byKey := make(map[string]aga.DraftItem, len(draft.Items))
	for _, draftItem := range draft.Items {
		byKey[draftItemQuestionKey(draftItem)] = draftItem
	}
	for index := range merged {
		item := &merged[index]
		draftItem, found := byKey[item.QuestionKey]
		if !found {
			continue
		}
		item.DraftAgreementConfidence = draftItem.DraftAgreementConfidence
		item.DraftRecommendationState = draftItem.RecommendationState
		item.DraftReviewState = draftItem.ReviewState
		item.DraftDisposition = draftItem.Disposition
	}
	return merged
}

func draftItemQuestionKey(item aga.DraftItem) string {
	return item.QuestionRef.Key()
}

// queryWorkspaceJSON is the only read seam used by the runtime workspace
// store. The dedicated reader/command roles can execute the projection
// function, while neither role receives direct table access.
func queryWorkspaceJSON(ctx context.Context, pool *database.Pool, input any, destination any) error {
	encoded, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("encode workspace query: %w", err)
	}
	var payload []byte
	if err := pool.QueryRow(ctx, `SELECT preprod_aga_demo_workspace.workspace_query($1::jsonb)`, encoded).Scan(&payload); err != nil {
		return err
	}
	if err := json.Unmarshal(payload, destination); err != nil {
		return fmt.Errorf("decode workspace query: %w", err)
	}
	return nil
}

// callWorkspaceCommand is the only runtime write seam. The function is
// SECURITY DEFINER and is granted explicitly to the command role; the role
// itself has no table DML privileges.
func callWorkspaceCommand(ctx context.Context, pool *database.Pool, input any) error {
	encoded, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("encode workspace command: %w", err)
	}
	var status []byte
	if err := pool.QueryRow(ctx, `SELECT preprod_aga_demo_workspace.workspace_command($1::jsonb)`, encoded).Scan(&status); err != nil {
		return err
	}
	return nil
}

func callWorkspaceReset(ctx context.Context, pool *database.Pool, input any) error {
	encoded, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("encode workspace reset: %w", err)
	}
	var status []byte
	if err := pool.QueryRow(ctx, `SELECT preprod_aga_demo_workspace.workspace_reset($1::jsonb)`, encoded).Scan(&status); err != nil {
		return err
	}
	return nil
}

func workspaceClassificationItemRows(runID string, items []ClassificationItem) []any {
	rows := make([]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, map[string]any{
			"classification_run_id": runID, "identity_key": item.Identity.Key(), "payload": item,
			"canonical_payload": canonical(item), "row_digest": digestJSON(item),
		})
	}
	return rows
}

func workspaceAuthorityBindingRows(generationID string, fixture FixtureManifest) []any {
	accounts := make(map[string]FixtureAccount, len(fixture.Accounts))
	for _, account := range fixture.Accounts {
		accounts[account.Slot] = account
	}
	rows := make([]any, 0, len(fixture.Bindings))
	for _, binding := range fixture.Bindings {
		account := accounts[binding.SubjectSlot]
		operationRoles := append([]string(nil), binding.OperationRoles...)
		organizationID := binding.OrganizationID
		if account.OrganizationID != "" {
			organizationID = account.OrganizationID
		}
		providerScopeID := workspaceProviderScopeForOrganization(organizationID)
		bindingDigest := binding.BindingDigest
		if !validDigest(bindingDigest) {
			bindingDigest = digestValue("AGA-DEMO-WORKSPACE-AUTHORITY-BINDING-V1", map[string]any{
				"bindingId": binding.BindingID, "subjectSlot": binding.SubjectSlot, "membershipSlot": binding.MembershipSlot,
				"organizationId": organizationID, "departmentId": binding.DepartmentID,
				"organizationalUnitId": binding.OrganizationalUnitID, "operationRoles": operationRoles,
				"providerScopeId": providerScopeID,
				"subjectId":       account.SubjectID, "membershipId": account.MembershipID,
				"membershipVersion": account.MembershipVersion, "membershipDigest": account.MembershipDigest,
				"roles": account.Roles, "active": binding.Active,
			})
		}
		payload := map[string]any{
			"bindingId": binding.BindingID, "subjectSlot": binding.SubjectSlot, "membershipSlot": binding.MembershipSlot,
			"organizationId": organizationID, "departmentId": binding.DepartmentID,
			"organizationalUnitId": binding.OrganizationalUnitID, "operationRoles": operationRoles,
			"generationId": generationID, "providerScopeId": providerScopeID,
			"bindingDigest": bindingDigest, "active": binding.Active,
			"subjectId": account.SubjectID, "membershipId": account.MembershipID,
			"membershipVersion": account.MembershipVersion, "membershipDigest": account.MembershipDigest,
			"roles": account.Roles,
		}
		rows = append(rows, map[string]any{
			"binding_id": binding.BindingID, "generation_id": generationID, "subject_slot": binding.SubjectSlot,
			"membership_slot": binding.MembershipSlot, "organization_id": organizationID,
			"department_id": binding.DepartmentID, "organizational_unit_id": binding.OrganizationalUnitID,
			"operation_roles": operationRoles, "binding_digest": bindingDigest, "active": binding.Active,
			"payload": payload,
		})
	}
	return rows
}

func workspaceProviderScopeForOrganization(organizationID string) string {
	for _, scope := range DefaultFixtureTemplate().Scopes {
		scopeOrganizationID := "AGA-DEMO-OTHER-ORG"
		if scope.OrganizationSlot == "MATCHING" {
			scopeOrganizationID = "AGA-DEMO-CAA"
		}
		if scopeOrganizationID == organizationID {
			return scope.ProviderScopeID
		}
	}
	return ""
}

func workspaceProviderRows(generationID string, now time.Time) ([]any, []any) {
	template := DefaultFixtureTemplate()
	scopes := make([]any, 0, len(template.Scopes))
	targets := make([]any, 0, len(template.Scopes))
	for _, scope := range template.Scopes {
		organizationID := "AGA-DEMO-OTHER-ORG"
		if scope.OrganizationSlot == "MATCHING" {
			organizationID = "AGA-DEMO-CAA"
		}
		effectiveFrom := now.UTC()
		operationQualifiers := []aga.Qualifier{{Key: "OPERATION_STATUS", Value: "ACTIVE"}}
		activityQualifiers := []aga.Qualifier{{Key: "ACTIVITY_TYPE", Value: "EMERGENCY_RESPONSE"}}
		profile := aga.ProviderScopeFact{
			GenerationID: generationID, OrganizationID: organizationID,
			ProviderScopeRootID: scope.ProviderScopeRootID, ProviderScopeID: scope.ProviderScopeID,
			ProviderScopeVersion: scope.ProviderScopeVersion, ProviderTypeID: scope.ProviderTypeCode,
			ProviderTypeCode: scope.ProviderTypeCode, Status: ProviderScopeStatusActive,
			EffectiveFrom: effectiveFrom, DepartmentID: template.DepartmentID,
			OrganizationalUnitID: template.OrganizationalUnitID,
			Targets:              []aga.TypedTarget{{ID: scope.TargetID, Kind: scope.CanonicalTargetKind, ProfileCode: scope.TargetProfileCode}},
			OperationQualifiers:  operationQualifiers, ActivityQualifiers: activityQualifiers,
		}
		profileDigest := aga.ComputeProviderScopeProfileDigest(profile)
		payload := map[string]any{
			"generationId": generationID, "organizationId": organizationID,
			"providerScopeRootId": scope.ProviderScopeRootID, "providerScopeId": scope.ProviderScopeID,
			"providerScopeVersion": scope.ProviderScopeVersion, "providerTypeId": scope.ProviderTypeCode,
			"providerTypeCode": scope.ProviderTypeCode, "status": ProviderScopeStatusActive,
			"effectiveFrom": effectiveFrom, "departmentId": template.DepartmentID, "organizationalUnitId": template.OrganizationalUnitID,
			"operationQualifiers": operationQualifiers, "activityQualifiers": activityQualifiers,
			"targets":       []ProviderTarget{{TargetID: scope.TargetID, CanonicalKind: scope.CanonicalTargetKind, ProfileCode: scope.TargetProfileCode}},
			"profileDigest": profileDigest,
		}
		scopes = append(scopes, map[string]any{
			"generation_id": generationID, "provider_scope_root_id": scope.ProviderScopeRootID, "provider_scope_id": scope.ProviderScopeID,
			"provider_scope_version": scope.ProviderScopeVersion, "provider_type_id": scope.ProviderTypeCode,
			"provider_type_code": scope.ProviderTypeCode, "organization_id": organizationID, "profile_digest": profileDigest,
			"payload": payload,
		})
		targets = append(targets, map[string]any{
			"generation_id": generationID, "provider_scope_id": scope.ProviderScopeID, "provider_scope_version": scope.ProviderScopeVersion,
			"target_id": scope.TargetID, "canonical_target_kind": scope.CanonicalTargetKind, "target_profile_code": scope.TargetProfileCode,
			"payload": map[string]any{"targetId": scope.TargetID, "canonicalTargetKind": scope.CanonicalTargetKind, "targetProfileCode": scope.TargetProfileCode},
		})
	}
	return scopes, targets
}

// validateResetReconstruction proves that the reset payload is a complete,
// generation-scoped reconstruction from the immutable fixture before it enters
// the SECURITY DEFINER reset function. The SQL function repeats the cardinality
// and generation checks because the command role can invoke that function
// directly; keeping the validation on both sides makes a malformed caller
// fail before publication and preserves the database boundary as the final
// authority.
func validateResetReconstruction(generationID string, fixture FixtureManifest, authorityBindings, providerScopes, providerTargets []any) error {
	if !workspaceIDPattern.MatchString(generationID) {
		return fmt.Errorf("%w: reset generation id", ErrWorkspaceInput)
	}
	if err := fixture.Validate(); err != nil {
		return fmt.Errorf("%w: reset fixture", err)
	}
	template := DefaultFixtureTemplate()
	if len(authorityBindings) != len(fixture.Bindings) || len(authorityBindings) == 0 || len(providerScopes) != len(template.Scopes) || len(providerTargets) != len(template.Scopes) {
		return fmt.Errorf("%w: reset reconstruction counts", ErrWorkspaceInput)
	}
	seenBindings := make(map[string]struct{}, len(authorityBindings))
	for _, row := range authorityBindings {
		values, ok := row.(map[string]any)
		if !ok || values["generation_id"] != generationID {
			return fmt.Errorf("%w: reset authority binding generation", ErrWorkspaceInput)
		}
		bindingID, _ := values["binding_id"].(string)
		if bindingID == "" {
			return fmt.Errorf("%w: reset authority binding id", ErrWorkspaceInput)
		}
		if _, duplicate := seenBindings[bindingID]; duplicate {
			return fmt.Errorf("%w: reset authority binding duplicate", ErrWorkspaceInput)
		}
		seenBindings[bindingID] = struct{}{}
	}
	seenScopes := make(map[string]struct{}, len(providerScopes))
	for _, row := range providerScopes {
		values, ok := row.(map[string]any)
		if !ok || values["generation_id"] != generationID {
			return fmt.Errorf("%w: reset provider scope generation", ErrWorkspaceInput)
		}
		scopeID, _ := values["provider_scope_id"].(string)
		if scopeID == "" {
			return fmt.Errorf("%w: reset provider scope id", ErrWorkspaceInput)
		}
		if _, duplicate := seenScopes[scopeID]; duplicate {
			return fmt.Errorf("%w: reset provider scope duplicate", ErrWorkspaceInput)
		}
		seenScopes[scopeID] = struct{}{}
	}
	seenTargets := make(map[string]struct{}, len(providerTargets))
	for _, row := range providerTargets {
		values, ok := row.(map[string]any)
		if !ok || values["generation_id"] != generationID {
			return fmt.Errorf("%w: reset provider target generation", ErrWorkspaceInput)
		}
		scopeID, _ := values["provider_scope_id"].(string)
		targetID, _ := values["target_id"].(string)
		key := scopeID + "\x00" + targetID
		if _, scopeExists := seenScopes[scopeID]; scopeID == "" || targetID == "" || !scopeExists {
			return fmt.Errorf("%w: reset provider target identity", ErrWorkspaceInput)
		}
		if _, duplicate := seenTargets[key]; duplicate {
			return fmt.Errorf("%w: reset provider target duplicate", ErrWorkspaceInput)
		}
		seenTargets[key] = struct{}{}
	}
	return nil
}

func (store *PostgresStore) PutRecommendationSnapshot(ctx context.Context, snapshot RecommendationSnapshot) (RecommendationSnapshot, bool, error) {
	if store == nil || store.pool == nil || !store.command {
		return RecommendationSnapshot{}, false, fmt.Errorf("workspace recommendation snapshot requires command store")
	}
	if ValidateRecommendationSnapshot(snapshot) != nil {
		return RecommendationSnapshot{}, false, ErrWorkspaceAppendOnly
	}
	if existing, found, err := store.GetRecommendationSnapshot(ctx, snapshot.Recommendation.GenerationID, snapshot.Recommendation.RecommendationID); err != nil {
		return RecommendationSnapshot{}, false, err
	} else if found {
		if canonical(existing) != canonical(snapshot) {
			return RecommendationSnapshot{}, false, ErrWorkspaceIdempotency
		}
		return existing, true, nil
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return RecommendationSnapshot{}, false, err
	}
	input := map[string]any{
		"operation": "APPEND_RECOMMENDATION", "recommendationId": snapshot.Recommendation.RecommendationID,
		"generationId": snapshot.Recommendation.GenerationID, "draftId": snapshot.Recommendation.DraftID,
		"draftRevision": snapshot.Recommendation.DraftRevision, "recommendationDigest": snapshot.Recommendation.Digest,
		"snapshotDigest": snapshot.SnapshotDigest, "payload": json.RawMessage(payload), "createdAt": snapshot.CreatedAt,
	}
	if err := callWorkspaceCommand(ctx, store.pool, input); err != nil {
		return RecommendationSnapshot{}, false, err
	}
	return snapshot, false, nil
}

func (store *PostgresStore) GetRecommendationSnapshot(ctx context.Context, generationID, recommendationID string) (RecommendationSnapshot, bool, error) {
	if store == nil || store.pool == nil {
		return RecommendationSnapshot{}, false, fmt.Errorf("workspace recommendation snapshot requires store")
	}
	var payload json.RawMessage
	if err := queryWorkspaceJSON(ctx, store.pool, map[string]any{"operation": "GET_RECOMMENDATION", "generationId": generationID, "recommendationId": recommendationID}, &payload); err != nil {
		return RecommendationSnapshot{}, false, err
	}
	if string(payload) == "null" || len(payload) == 0 {
		return RecommendationSnapshot{}, false, nil
	}
	var snapshot RecommendationSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return RecommendationSnapshot{}, false, err
	}
	return snapshot, true, nil
}

func (store *PostgresStore) ListRecommendationSnapshots(ctx context.Context, generationID string) ([]RecommendationSnapshot, error) {
	if store == nil || store.pool == nil {
		return nil, fmt.Errorf("workspace recommendation snapshot requires store")
	}
	var raw json.RawMessage
	if err := queryWorkspaceJSON(ctx, store.pool, map[string]any{"operation": "GET_CURRENT_RECOMMENDATIONS", "generationId": generationID}, &raw); err != nil {
		return nil, err
	}
	if string(raw) == "null" || len(raw) == 0 {
		return []RecommendationSnapshot{}, nil
	}
	var snapshots []RecommendationSnapshot
	if err := json.Unmarshal(raw, &snapshots); err != nil {
		return nil, err
	}
	return snapshots, nil
}

func (store *PostgresStore) PutSelectionBatchPreview(ctx context.Context, record SelectionBatchPreviewRecord) (SelectionBatchPreviewRecord, bool, error) {
	if store == nil || store.pool == nil || !store.command {
		return SelectionBatchPreviewRecord{}, false, fmt.Errorf("workspace batch preview requires command store")
	}
	if err := validateSelectionBatchPreviewRecord(record); err != nil {
		return SelectionBatchPreviewRecord{}, false, err
	}
	if existing, found, err := store.GetSelectionBatchPreview(ctx, record.GenerationID, record.PreviewID); err != nil {
		return SelectionBatchPreviewRecord{}, false, err
	} else if found {
		if canonical(existing) != canonical(record) {
			return SelectionBatchPreviewRecord{}, false, ErrWorkspaceIdempotency
		}
		return existing, true, nil
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return SelectionBatchPreviewRecord{}, false, err
	}
	if err := callWorkspaceCommand(ctx, store.pool, map[string]any{
		"operation": "APPEND_BATCH_PREVIEW", "previewId": record.PreviewID, "generationId": record.GenerationID,
		"draftId": record.DraftID, "draftRevision": record.DraftRevision, "previewDigest": record.PreviewDigest,
		"expiresAt": record.ExpiresAt, "payload": json.RawMessage(payload),
	}); err != nil {
		return SelectionBatchPreviewRecord{}, false, err
	}
	return record, false, nil
}

func (store *PostgresStore) GetSelectionBatchPreview(ctx context.Context, generationID, previewID string) (SelectionBatchPreviewRecord, bool, error) {
	if store == nil || store.pool == nil {
		return SelectionBatchPreviewRecord{}, false, fmt.Errorf("workspace batch preview requires store")
	}
	var payload json.RawMessage
	if err := queryWorkspaceJSON(ctx, store.pool, map[string]any{"operation": "GET_BATCH_PREVIEW", "generationId": generationID, "previewId": previewID}, &payload); err != nil {
		return SelectionBatchPreviewRecord{}, false, err
	}
	if string(payload) == "null" || len(payload) == 0 {
		return SelectionBatchPreviewRecord{}, false, nil
	}
	var record SelectionBatchPreviewRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return SelectionBatchPreviewRecord{}, false, err
	}
	return record, true, nil
}

func (store *PostgresStore) ConsumeSelectionBatchPreview(ctx context.Context, generationID, previewID, previewDigest string, now time.Time) (SelectionBatchPreviewRecord, error) {
	if store == nil || store.pool == nil || !store.command {
		return SelectionBatchPreviewRecord{}, fmt.Errorf("workspace batch preview requires command store")
	}
	if strings.TrimSpace(previewID) == "" || strings.TrimSpace(previewDigest) == "" {
		return SelectionBatchPreviewRecord{}, ErrWorkspaceCAS
	}
	if err := callWorkspaceCommand(ctx, store.pool, map[string]any{
		"operation": "CONSUME_BATCH_PREVIEW", "generationId": generationID, "previewId": previewID,
		"previewDigest": previewDigest, "consumedAt": now.UTC(),
	}); err != nil {
		return SelectionBatchPreviewRecord{}, err
	}
	record, found, err := store.GetSelectionBatchPreview(ctx, generationID, previewID)
	if err != nil {
		return SelectionBatchPreviewRecord{}, err
	}
	if !found || record.ConsumedAt == nil {
		return SelectionBatchPreviewRecord{}, ErrWorkspaceCAS
	}
	return record, nil
}

func (store *PostgresStore) ApplyDraftCommand(ctx context.Context, command aga.DraftCommand) (aga.Draft, error) {
	if store == nil || store.pool == nil || !store.command {
		return aga.Draft{}, fmt.Errorf("workspace Draft command requires command store")
	}
	workspace, err := store.Snapshot(ctx)
	if err != nil {
		return aga.Draft{}, err
	}
	draft, err := aga.HydrateDraftForRuntime(workspace.Draft.Draft, workspace.Run.Result)
	if err != nil {
		return aga.Draft{}, err
	}
	updated, err := aga.ApplyDraftCommand(draft, command, aga.NewSequentialIDAllocator("postgres"))
	if err != nil {
		return aga.Draft{}, err
	}
	if err := callWorkspaceCommand(ctx, store.pool, map[string]any{
		"operation": "APPEND_DRAFT", "generationId": updated.GenerationID, "draftId": updated.DraftID,
		"revision": updated.Revision, "contentDigest": updated.ContentDigest, "state": updated.State,
		"payload": updated, "createdAt": command.CreatedAt.UTC(), "canonicalPayload": canonical(updated), "rowDigest": digestJSON(updated),
	}); err != nil {
		return aga.Draft{}, err
	}
	store.runtimeMu.Lock()
	store.runtimeCurrentDraft = updated
	store.runtimeCurrentGeneration = updated.GenerationID
	store.runtimeCurrentRevision = updated.Revision
	store.runtimeCurrentDigest = updated.ContentDigest
	store.runtimeCurrentReady = true
	store.runtimeMu.Unlock()
	return updated, nil
}

func (store *PostgresStore) ApplyDraftCommandsAtomically(ctx context.Context, draft aga.Draft, commands []aga.DraftCommand) (aga.Draft, error) {
	if store == nil || store.pool == nil || !store.command {
		return aga.Draft{}, fmt.Errorf("workspace batch Draft command requires command store")
	}
	if len(commands) == 0 {
		return aga.Draft{}, ErrWorkspaceAppendOnly
	}
	if commands[0].ExpectedRevision != draft.Revision || commands[0].ExpectedContentDigest != draft.ContentDigest {
		return aga.Draft{}, ErrWorkspaceCAS
	}
	current := draft
	for index := range commands {
		command := commands[index]
		command.ExpectedRevision = current.Revision
		command.ExpectedContentDigest = current.ContentDigest
		updated, err := aga.ApplyDraftCommandFromValidatedRuntime(current, command, aga.NewSequentialIDAllocator("postgres-batch"))
		if err != nil {
			return aga.Draft{}, err
		}
		current = updated
	}
	if err := callWorkspaceCommand(ctx, store.pool, map[string]any{
		"operation": "APPEND_BATCH_DRAFT", "generationId": current.GenerationID, "draftId": current.DraftID,
		"expectedRevision": draft.Revision, "expectedContentDigest": draft.ContentDigest,
		"revision": current.Revision, "contentDigest": current.ContentDigest, "state": current.State,
		"payload": current, "createdAt": time.Now().UTC(), "canonicalPayload": canonical(current), "rowDigest": digestJSON(current),
	}); err != nil {
		return aga.Draft{}, err
	}
	store.runtimeMu.Lock()
	store.runtimeCurrentDraft = current
	store.runtimeCurrentGeneration = current.GenerationID
	store.runtimeCurrentRevision = current.Revision
	store.runtimeCurrentDigest = current.ContentDigest
	store.runtimeCurrentReady = true
	store.runtimeMu.Unlock()
	return current, nil
}

func (store *PostgresStore) ExecuteSelectionBatch(ctx context.Context, record SelectionBatchPreviewRecord, draft aga.Draft, commands []aga.DraftCommand, now time.Time) (aga.Draft, SelectionBatchPreviewRecord, error) {
	if store == nil || store.pool == nil || !store.command {
		return aga.Draft{}, SelectionBatchPreviewRecord{}, fmt.Errorf("workspace batch execution requires command store")
	}
	if runtime, err := store.SnapshotForRuntimeCommand(ctx); err == nil {
		draft = runtime.Draft.Draft
	} else {
		return aga.Draft{}, SelectionBatchPreviewRecord{}, err
	}
	if len(commands) == 0 || commands[0].ExpectedRevision != draft.Revision || commands[0].ExpectedContentDigest != draft.ContentDigest {
		return aga.Draft{}, SelectionBatchPreviewRecord{}, ErrWorkspaceCAS
	}
	current, err := aga.ApplyDraftDispositionBatchFromValidatedRuntime(draft, commands)
	if err != nil {
		return aga.Draft{}, SelectionBatchPreviewRecord{}, err
	}
	payload, err := json.Marshal(current)
	if err != nil {
		return aga.Draft{}, SelectionBatchPreviewRecord{}, err
	}
	consumed := now.UTC()
	err = callWorkspaceCommand(ctx, store.pool, map[string]any{
		"operation":    "APPEND_BATCH_DRAFT_AND_CONSUME",
		"generationId": record.GenerationID, "draftId": draft.DraftID,
		"expectedRevision": draft.Revision, "expectedContentDigest": draft.ContentDigest,
		"previewId": record.PreviewID, "previewDigest": record.PreviewDigest,
		"consumedAt": consumed, "revision": current.Revision, "contentDigest": current.ContentDigest,
		"state": current.State, "payload": json.RawMessage(payload),
		"createdAt": consumed, "canonicalPayload": canonical(current), "rowDigest": digestJSON(current),
	})
	if err != nil {
		return aga.Draft{}, SelectionBatchPreviewRecord{}, err
	}
	record.ConsumedAt = &consumed
	store.runtimeMu.Lock()
	store.runtimeCurrentDraft = current
	store.runtimeCurrentGeneration = current.GenerationID
	store.runtimeCurrentRevision = current.Revision
	store.runtimeCurrentDigest = current.ContentDigest
	store.runtimeCurrentReady = true
	store.runtimeMu.Unlock()
	return current, record, nil
}

// ApplyDraftCommandFromRuntime persists a command against the already-loaded
// runtime draft. The service has performed authorization, idempotency lookup,
// generation CAS, and sealed-context hydration before reaching this seam.
func (store *PostgresStore) ApplyDraftCommandFromRuntime(ctx context.Context, draft aga.Draft, command aga.DraftCommand) (aga.Draft, error) {
	if store == nil || store.pool == nil || !store.command {
		return aga.Draft{}, fmt.Errorf("workspace Draft command requires command store")
	}
	updated, err := aga.ApplyDraftCommandFromValidatedRuntime(draft, command, aga.NewSequentialIDAllocator("postgres"))
	if err != nil {
		return aga.Draft{}, err
	}
	if err := callWorkspaceCommand(ctx, store.pool, map[string]any{
		"operation": "APPEND_DRAFT", "generationId": updated.GenerationID, "draftId": updated.DraftID,
		"revision": updated.Revision, "contentDigest": updated.ContentDigest, "state": updated.State,
		"payload": updated, "createdAt": command.CreatedAt.UTC(), "canonicalPayload": canonical(updated), "rowDigest": digestJSON(updated),
	}); err != nil {
		return aga.Draft{}, err
	}
	store.runtimeMu.Lock()
	store.runtimeCurrentDraft = updated
	store.runtimeCurrentGeneration = updated.GenerationID
	store.runtimeCurrentRevision = updated.Revision
	store.runtimeCurrentDigest = updated.ContentDigest
	store.runtimeCurrentReady = true
	store.runtimeMu.Unlock()
	return updated, nil
}

func (store *PostgresStore) rememberRuntimeClassification(classification aga.ClassificationResult) {
	if classification.ClassificationRunID == "" {
		return
	}
	store.runtimeMu.Lock()
	defer store.runtimeMu.Unlock()
	store.runtimeClassification = classification
	store.runtimeClassificationID = classification.ClassificationRunID
}

func (store *PostgresStore) AppendQuestionVersion(ctx context.Context, input AppendQuestionVersionInput) (WorkspaceQuestionVersion, error) {
	if store == nil || store.pool == nil || !store.command {
		return WorkspaceQuestionVersion{}, fmt.Errorf("workspace question append requires command store")
	}
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	input.Now = input.Now.UTC()
	if input.BodyDigest == "" {
		input.BodyDigest = bodyDigest(input.Body)
	}
	if input.RootID == "" {
		input.RootID = "aga-ws-root-runtime"
	}
	if input.VersionID == "" {
		input.VersionID = "aga-ws-version-runtime"
	}
	if input.ProposalID == "" {
		input.ProposalID = "aga-ws-proposal-runtime"
	}
	if input.RootSequence == 0 {
		input.RootSequence = 1
	}
	version := WorkspaceQuestionVersion{GenerationID: input.GenerationID, RootID: input.RootID, VersionID: input.VersionID, ProposalID: input.ProposalID, RootSequence: input.RootSequence, BodyDigest: input.BodyDigest, Body: input.Body, ParentQuestionKey: input.ParentQuestionKey, ActorSubjectID: input.ActorSubjectID, CreatedAt: input.Now, ReasonCode: trimReason(input.ReasonCode), CurrentLeaf: true}
	if err := callWorkspaceCommand(ctx, store.pool, map[string]any{
		"operation": "APPEND_QUESTION_VERSION", "generationId": version.GenerationID, "rootId": version.RootID,
		"versionId": version.VersionID, "proposalId": version.ProposalID, "rootSequence": version.RootSequence,
		"bodyDigest": version.BodyDigest, "body": version.Body, "parentQuestionKey": version.ParentQuestionKey,
		"actorSubjectId": version.ActorSubjectID, "createdAt": version.CreatedAt, "reasonCode": version.ReasonCode,
		"payload": version, "canonicalPayload": canonical(version), "rowDigest": digestJSON(version),
	}); err != nil {
		return WorkspaceQuestionVersion{}, fmt.Errorf("append workspace question version: %w", err)
	}
	return version, nil
}

func (store *PostgresStore) PutIdempotencyResponse(ctx context.Context, response IdempotencyResponse) (IdempotencyResponse, bool, error) {
	if store == nil || store.pool == nil || !store.command {
		return IdempotencyResponse{}, false, fmt.Errorf("workspace idempotency requires command store")
	}
	key := StoredResponseKey{GenerationID: response.GenerationID, ActorSubjectID: response.ActorSubjectID, OperationID: response.OperationID, IdempotencyKey: response.IdempotencyKey}
	if err := key.Validate(); err != nil {
		return IdempotencyResponse{}, false, err
	}
	if existing, found, err := store.GetIdempotencyResponse(ctx, key); err != nil {
		return IdempotencyResponse{}, false, err
	} else if found {
		if existing.CommandHash != response.CommandHash || existing.AuthorizationScopeDigest != response.AuthorizationScopeDigest {
			return IdempotencyResponse{}, false, ErrWorkspaceIdempotency
		}
		return existing, true, nil
	}
	if err := callWorkspaceCommand(ctx, store.pool, map[string]any{
		"operation": "APPEND_IDEMPOTENCY", "generationId": response.GenerationID, "actorSubjectId": response.ActorSubjectID,
		"operationId": response.OperationID, "idempotencyKey": response.IdempotencyKey, "commandHash": response.CommandHash,
		"authorizationScopeDigest": response.AuthorizationScopeDigest, "statusCode": response.StatusCode,
		"response": json.RawMessage(response.Response), "createdAt": response.CreatedAt,
	}); err != nil {
		return IdempotencyResponse{}, false, err
	}
	return response, false, nil
}

func (store *PostgresStore) GetIdempotencyResponse(ctx context.Context, key StoredResponseKey) (IdempotencyResponse, bool, error) {
	if store == nil || store.pool == nil {
		return IdempotencyResponse{}, false, fmt.Errorf("workspace PostgreSQL store is required")
	}
	if err := key.Validate(); err != nil {
		return IdempotencyResponse{}, false, err
	}
	var raw json.RawMessage
	if err := queryWorkspaceJSON(ctx, store.pool, map[string]any{"operation": "GET_IDEMPOTENCY", "generationId": key.GenerationID, "actorSubjectId": key.ActorSubjectID, "operationId": key.OperationID, "idempotencyKey": key.IdempotencyKey}, &raw); err != nil {
		return IdempotencyResponse{}, false, err
	}
	if string(raw) == "null" || len(raw) == 0 {
		return IdempotencyResponse{}, false, nil
	}
	var response IdempotencyResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return IdempotencyResponse{}, false, err
	}
	return response, true, nil
}

func (store *PostgresStore) ResetGeneration(ctx context.Context, input ResetInput) (Generation, ResetTombstone, error) {
	if store == nil || store.pool == nil || !store.command {
		return Generation{}, ResetTombstone{}, fmt.Errorf("workspace reset requires command store")
	}
	workspace, err := store.Snapshot(ctx)
	if err != nil {
		return Generation{}, ResetTombstone{}, err
	}
	current := workspace.Generation
	if current.State != GenerationActive {
		return Generation{}, ResetTombstone{}, ErrWorkspaceCAS
	}
	if current.GenerationID != input.ExpectedGenerationID || current.Revision != input.ExpectedGenerationRevision || current.SealDigest != input.ExpectedGenerationSealDigest {
		return Generation{}, ResetTombstone{}, ErrWorkspaceCAS
	}
	if trimReason(input.ReasonCode) == "" || input.ActorSubjectID == "" {
		return Generation{}, ResetTombstone{}, ErrWorkspaceAppendOnly
	}
	if input.Now.IsZero() {
		input.Now = time.Now().UTC()
	}
	input.Now = input.Now.UTC()
	newGenerationID := fmt.Sprintf("aga-ws-generation-reset-%d", input.Now.UnixNano())
	draft, err := freshResetDraft(workspace, newGenerationID)
	if err != nil {
		return Generation{}, ResetTombstone{}, err
	}
	resetItems := mergeDraftMetadata(workspace.Items, draft)
	workspaceAggregateDigest := digestValue("AGA-DEMO-WORKSPACE-AGGREGATE-V1", struct {
		GenerationID string
		Run          string
		Items        []ClassificationItem
	}{newGenerationID, workspace.Run.ClassificationRunDigest, resetItems})
	workspaceSeal := WorkspaceSealReceipt{
		GenerationID: newGenerationID, ClassificationRunDigest: current.ClassificationRunDigest,
		FixtureDigest: current.FixtureDigest, WorkspaceAggregateDigest: workspaceAggregateDigest,
		SealedAt: input.Now, LoaderRevoked: true,
	}
	workspaceSeal.SealDigest = digestValue("AGA-DEMO-WORKSPACE-SEAL-V1", workspaceSeal)
	newGeneration := Generation{GenerationID: newGenerationID, State: GenerationActive, ClassificationRunID: current.ClassificationRunID, ClassificationRunDigest: current.ClassificationRunDigest, TaxonomyVersion: current.TaxonomyVersion, TaxonomyDigest: current.TaxonomyDigest, FixtureDigest: current.FixtureDigest, Revision: 1, SealDigest: workspaceSeal.SealDigest, CreatedAt: input.Now, ResetFromGenerationID: current.GenerationID}
	newGeneration.SealDigest = digestValue("AGA-DEMO-WORKSPACE-GENERATION-V1", struct {
		Generation Generation
		Draft      aga.Draft
	}{newGeneration, draft})
	tombstone := ResetTombstone{TombstoneID: fmt.Sprintf("aga-ws-tombstone-%d", input.Now.UnixNano()), FromGenerationID: current.GenerationID, ToGenerationID: newGeneration.GenerationID, ExpectedGenerationID: input.ExpectedGenerationID, ExpectedRevision: input.ExpectedGenerationRevision, ExpectedSealDigest: input.ExpectedGenerationSealDigest, ReasonCode: trimReason(input.ReasonCode), ActorSubjectID: input.ActorSubjectID, CreatedAt: input.Now}
	tombstone.TombstoneDigest = digestValue("AGA-DEMO-WORKSPACE-RESET-TOMBSTONE-V1", tombstone)
	providerScopes, providerTargets := workspaceProviderRows(newGenerationID, input.Now)
	authorityBindings := workspaceAuthorityBindingRows(newGenerationID, workspace.Fixture)
	if err := validateResetReconstruction(newGenerationID, workspace.Fixture, authorityBindings, providerScopes, providerTargets); err != nil {
		return Generation{}, ResetTombstone{}, err
	}
	err = callWorkspaceReset(ctx, store.pool, map[string]any{
		"tombstoneId": tombstone.TombstoneID, "fromGenerationId": tombstone.FromGenerationID,
		"toGenerationId": tombstone.ToGenerationID, "expectedGenerationId": tombstone.ExpectedGenerationID,
		"expectedRevision": tombstone.ExpectedRevision, "expectedSealDigest": tombstone.ExpectedSealDigest,
		"reasonCode": tombstone.ReasonCode, "actorSubjectId": tombstone.ActorSubjectID, "createdAt": tombstone.CreatedAt,
		"tombstoneDigest": tombstone.TombstoneDigest, "classificationRunId": newGeneration.ClassificationRunID,
		"classificationRunDigest": newGeneration.ClassificationRunDigest, "taxonomyVersion": newGeneration.TaxonomyVersion,
		"taxonomyDigest": newGeneration.TaxonomyDigest, "fixtureDigest": newGeneration.FixtureDigest,
		"newGenerationSealDigest": newGeneration.SealDigest, "newWorkspaceSealDigest": workspaceSeal.SealDigest,
		"workspaceAggregateDigest": workspaceSeal.WorkspaceAggregateDigest,
		"draft":                    draft, "draftCanonicalPayload": canonical(draft), "draftRowDigest": digestJSON(draft),
		"authorityBindings": authorityBindings, "authorityBindingCount": len(authorityBindings),
		"providerScopes": providerScopes, "providerScopeCount": len(providerScopes),
		"providerTargets": providerTargets, "providerTargetCount": len(providerTargets),
	})
	if err != nil {
		return Generation{}, ResetTombstone{}, err
	}
	store.runtimeMu.Lock()
	store.runtimeBaseReady = false
	store.runtimeCurrentReady = false
	store.runtimeBaseWorkspace = LoadedWorkspace{}
	store.runtimeCurrentDraft = aga.Draft{}
	store.runtimeMu.Unlock()
	return newGeneration, tombstone, nil
}

func agaNewDraft(result aga.ClassificationResult, generation string) (aga.Draft, error) {
	return aga.NewDraftFromClassification(result, generation)
}

func canonical(value any) string  { data, _ := json.Marshal(value); return string(data) }
func digestJSON(value any) string { return digestValue("AGA-DEMO-WORKSPACE-ROW-V1", value) }
func nullableJSON(value any) string {
	if value == nil {
		return "null"
	}
	return canonical(value)
}
func insertJSON(ctx context.Context, tx pgx.Tx, query string, args ...any) error {
	_, err := tx.Exec(ctx, query, args...)
	return err
}
