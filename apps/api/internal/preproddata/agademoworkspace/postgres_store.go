package agademoworkspace

import (
	"context"
	"encoding/json"
	"fmt"
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
		classificationItem := ClassificationItem{
			QuestionKey: item.Identity.Key(), Identity: item.Identity, Projection: item.Projection,
			AgreementConfidence: item.AgreementConfidence, RecommendationState: item.RecommendationState,
			Governance: item.GovernanceState, ItemSemanticDigest: item.ItemSemanticDigest,
			CandidateDigest: item.PassOneResultDigest, ChallengeDigest: item.PassTwoResultDigest,
		}
		if draftItem, ok := draftItemsByKey[item.Identity.Key()]; ok {
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
	store.rememberRuntimeClassification(workspace.Run.Result)
	return workspace, nil
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
	store.runtimeMu.Unlock()
	if classificationID == "" {
		workspace, err := store.Snapshot(ctx)
		if err != nil {
			return LoadedWorkspace{}, err
		}
		classification = workspace.Run.Result
		classificationID = workspace.Run.RunID
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
	if store.runtimeCurrentReady && store.runtimeCurrentGeneration == generation.GenerationID && store.runtimeCurrentRevision == meta.Draft.Revision && store.runtimeCurrentDigest == meta.Draft.ContentDigest && store.runtimeClassificationID == classificationID {
		current := store.runtimeCurrentDraft
		store.runtimeMu.Unlock()
		return LoadedWorkspace{Generation: generation, Run: ClassificationRun{RunID: classification.ClassificationRunID, State: classification.State, TaxonomyVersion: classification.TaxonomyVersion, TaxonomyDigest: classification.TaxonomyDigest, InputDigest: classification.InputDigest, AggregateDigest: classification.AggregateDigest, ClassificationRunDigest: classification.ClassificationRunDigest, Result: classification}, Draft: DraftRecord{Draft: current}}, nil
	}
	store.runtimeMu.Unlock()
	if generation.ClassificationRunID != classificationID || generation.ClassificationRunDigest != classification.ClassificationRunDigest {
		workspace, err := store.Snapshot(ctx)
		if err != nil {
			return LoadedWorkspace{}, err
		}
		classification = workspace.Run.Result
		classificationID = workspace.Run.RunID
		generation = workspace.Generation
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
	return LoadedWorkspace{
		Generation: state.Generation,
		Run:        ClassificationRun{RunID: classification.ClassificationRunID, State: classification.State, TaxonomyVersion: classification.TaxonomyVersion, TaxonomyDigest: classification.TaxonomyDigest, InputDigest: classification.InputDigest, AggregateDigest: classification.AggregateDigest, ClassificationRunDigest: classification.ClassificationRunDigest, Result: classification},
		Draft:      state.Draft,
		Seal:       state.Seal,
	}, nil
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
		bindingDigest := binding.BindingDigest
		if !validDigest(bindingDigest) {
			bindingDigest = digestValue("AGA-DEMO-WORKSPACE-AUTHORITY-BINDING-V1", map[string]any{
				"bindingId": binding.BindingID, "subjectSlot": binding.SubjectSlot, "membershipSlot": binding.MembershipSlot,
				"organizationId": organizationID, "departmentId": binding.DepartmentID,
				"organizationalUnitId": binding.OrganizationalUnitID, "operationRoles": operationRoles,
				"subjectId": account.SubjectID, "membershipId": account.MembershipID,
				"membershipVersion": account.MembershipVersion, "membershipDigest": account.MembershipDigest,
				"roles": account.Roles, "active": binding.Active,
			})
		}
		payload := map[string]any{
			"bindingId": binding.BindingID, "subjectSlot": binding.SubjectSlot, "membershipSlot": binding.MembershipSlot,
			"organizationId": organizationID, "departmentId": binding.DepartmentID,
			"organizationalUnitId": binding.OrganizationalUnitID, "operationRoles": operationRoles,
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
	var basePayload json.RawMessage
	if err := queryWorkspaceJSON(ctx, store.pool, map[string]any{"operation": "GET_DRAFT_BASE", "generationId": current.GenerationID}, &basePayload); err != nil {
		return Generation{}, ResetTombstone{}, err
	}
	var draft aga.Draft
	if err := json.Unmarshal(basePayload, &draft); err != nil || draft.DraftID == "" {
		return Generation{}, ResetTombstone{}, ErrWorkspaceNotSealed
	}
	newGenerationID := fmt.Sprintf("aga-ws-generation-reset-%d", input.Now.UnixNano())
	draft.GenerationID = newGenerationID
	draft.GenerationState = GenerationActive
	draft.Revision = 1
	draft.State = aga.DraftWorking
	draft.ReadinessEvents = nil
	draft.CurrentReadinessEventID = ""
	draft.ContentDigest = aga.ComputeDraftContentDigest(draft)
	workspaceSeal := WorkspaceSealReceipt{
		GenerationID: newGenerationID, ClassificationRunDigest: current.ClassificationRunDigest,
		FixtureDigest: current.FixtureDigest, WorkspaceAggregateDigest: workspace.Seal.WorkspaceAggregateDigest,
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
	err = callWorkspaceReset(ctx, store.pool, map[string]any{
		"tombstoneId": tombstone.TombstoneID, "fromGenerationId": tombstone.FromGenerationID,
		"toGenerationId": tombstone.ToGenerationID, "expectedGenerationId": tombstone.ExpectedGenerationID,
		"expectedRevision": tombstone.ExpectedRevision, "expectedSealDigest": tombstone.ExpectedSealDigest,
		"reasonCode": tombstone.ReasonCode, "actorSubjectId": tombstone.ActorSubjectID, "createdAt": tombstone.CreatedAt,
		"tombstoneDigest": tombstone.TombstoneDigest, "classificationRunId": newGeneration.ClassificationRunID,
		"classificationRunDigest": newGeneration.ClassificationRunDigest, "taxonomyVersion": newGeneration.TaxonomyVersion,
		"taxonomyDigest": newGeneration.TaxonomyDigest, "fixtureDigest": newGeneration.FixtureDigest,
		"newGenerationSealDigest": newGeneration.SealDigest, "newWorkspaceSealDigest": workspaceSeal.SealDigest,
		"draft": draft, "draftCanonicalPayload": canonical(draft), "draftRowDigest": digestJSON(draft),
	})
	if err != nil {
		return Generation{}, ResetTombstone{}, err
	}
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
