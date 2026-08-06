package agademoworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func NewService(config ServiceConfig) *Service {
	clock := config.Clock
	if clock == nil {
		clock = time.Now
	}
	commandStore := config.CommandStore
	if commandStore == nil {
		commandStore = config.Store
	}
	readerStore := config.ReaderStore
	if readerStore == nil {
		readerStore = config.Store
	}
	return &Service{store: config.Store, reader: readerStore, command: commandStore, resolver: config.Resolver, questionBodies: config.QuestionBodies, questionTextSearch: config.QuestionTextSearch, recommendationScopes: config.RecommendationScopes, simulationSetupResolver: config.SimulationSetup, lifecycleBindings: config.LifecycleBindings, clock: clock}
}

type readerDraftCache interface {
	UpdateCachedReaderDraft(string, aga.Draft)
	InvalidateCachedReaderSnapshot()
}

func (service *Service) updateReaderDraftCache(generationID string, draft *aga.Draft) {
	if draft == nil {
		return
	}
	if cache, ok := service.reader.(readerDraftCache); ok {
		cache.UpdateCachedReaderDraft(generationID, *draft)
	}
}

func (service *Service) invalidateReaderSnapshotCache() {
	if cache, ok := service.reader.(readerDraftCache); ok {
		cache.InvalidateCachedReaderSnapshot()
	}
}

func (service *Service) Capability(ctx context.Context, principal identity.Principal) (Capability, error) {
	if !service.HasBroadAuthority(ctx, principal) {
		return Capability{}, ErrNeutralDenied
	}
	if principal.HasRole(identity.RoleAdmin) && isCAAOrganization(principal.OrganizationID) {
		return Capability{Available: true, Projection: "CAA_ADMIN", ClassificationEnabled: true, RecommendationEnabled: true, LifecycleEnabled: true, ResetEnabled: true}, nil
	}
	binding, found, err := service.ResolveBinding(ctx, principal)
	if err != nil || !found {
		return Capability{}, ErrNeutralDenied
	}
	projection := "WORKSPACE_SCOPED"
	if principal.HasRole(identity.RoleAuditee) {
		projection = "AUDITEE_ORGANIZATION_SCOPED"
	} else if principal.HasRole(identity.RoleDepartmentManager) {
		projection = "DEPARTMENT_MANAGER_SCOPED"
	} else if principal.HasRole(identity.RoleLeadInspector) {
		projection = "LEAD_INSPECTOR_SCOPED"
	} else if principal.HasRole(identity.RoleInspector) {
		projection = "INSPECTOR_SCOPED"
	}
	_ = binding
	lifecycleEnabled := principal.HasRole(identity.RoleAuditee, identity.RoleInspector, identity.RoleLeadInspector, identity.RoleDepartmentManager) && bindingHasWorkspaceRole(binding, principal, "LIFECYCLE_READ")
	return Capability{Available: true, Projection: projection, ClassificationEnabled: principal.HasRole(identity.RoleDepartmentManager), RecommendationEnabled: principal.HasRole(identity.RoleDepartmentManager), LifecycleEnabled: lifecycleEnabled, ResetEnabled: false}, nil
}

func (service *Service) Query(ctx context.Context, principal identity.Principal, request QueryRequest) (QueryResponse, error) {
	if err := request.Validate(); err != nil {
		return QueryResponse{}, err
	}
	if _, err := service.Authorize(ctx, principal, request.OperationID); err != nil {
		return QueryResponse{}, err
	}
	if isRecommendationQuery(request.OperationID) {
		return service.recommendationQuery(ctx, principal, request)
	}
	if isLifecycleQuery(request.OperationID) {
		return service.lifecycleQuery(ctx, principal, request)
	}
	if !isClassificationQuery(request.OperationID) {
		return QueryResponse{}, ErrCapabilityUnavailable
	}
	if service == nil || service.reader == nil {
		return QueryResponse{}, ErrWorkspaceStore
	}
	workspace, err := service.reader.Snapshot(ctx)
	if err != nil {
		return QueryResponse{}, fmt.Errorf("%w: %v", ErrWorkspaceStore, err)
	}
	response := QueryResponse{Operation: request.OperationID, Generation: workspace.Generation, ItemCount: len(workspace.Items), LifecycleAvailable: false}
	switch request.OperationID {
	case OperationGetSummary:
		response.BaseQuestionCount = workspace.Draft.Draft.BaseQuestionCount
		response.DraftIncludedCount, response.DraftExcludedCount, response.DraftDeferredCount = draftDispositionCounts(workspace.Draft.Draft)
	case OperationGetTaxonomy:
		taxonomy := workspace.Taxonomy
		response.Taxonomy = &taxonomy
	case OperationGetProviderConfiguration:
		// Provider configuration is a sealed fixture projection. The fixture
		// digest is exposed through Generation; no canonical provider lookup is
		// performed here.
		response.ProviderConfiguration = append([]preprod.ProviderConfigurationEntry(nil), workspace.Fixture.ProviderConfiguration...)
	case OperationSearchItems:
		rows, rowsErr := service.classificationReviewRows(ctx, workspace)
		if rowsErr != nil {
			return QueryResponse{}, rowsErr
		}
		metadataRequest := request
		// A non-empty Search is a sealed-body search when the tagged resolver is
		// present. Do not ask the metadata-only filter to match the body fragment
		// before the resolver has returned exact identities.
		if request.Search != "" && service.questionTextSearch != nil {
			metadataRequest.Search = ""
		}
		filteredRows := filterClassificationReviewRows(rows, metadataRequest)
		if request.Search != "" && service.questionTextSearch != nil {
			search, searchErr := normalizeBodySearch(request.Search)
			if searchErr != nil {
				return QueryResponse{}, searchErr
			}
			identities, resolveErr := service.questionTextSearch.Search(ctx, search)
			if resolveErr != nil {
				return QueryResponse{}, fmt.Errorf("%w: body search: %v", ErrQuestionBodyResolverUnavailable, resolveErr)
			}
			allowed := make(map[string]struct{}, len(identities))
			for _, identity := range identities {
				allowed[identity.Key()] = struct{}{}
			}
			filteredRows = filterReviewRowsByBodyIdentity(filteredRows, allowed, search)
		}
		pageSize := request.PageSize
		if pageSize == 0 {
			pageSize = MaxQuestionTextPage
		}
		if pageSize > MaxQuestionTextPage {
			return QueryResponse{}, ErrMalformedCommand
		}
		start := request.Page * pageSize
		if start > len(filteredRows) {
			start = len(filteredRows)
		}
		end := start + pageSize
		if end > len(filteredRows) {
			end = len(filteredRows)
		}
		pageRows := filteredRows[start:end]
		response.Items = make([]ClassificationReviewItem, 0, len(pageRows))
		for _, row := range pageRows {
			response.Items = append(response.Items, reviewMetadataItem(row))
		}
		decision, decisionErr := service.Authorize(ctx, principal, request.OperationID)
		if decisionErr != nil {
			return QueryResponse{}, decisionErr
		}
		if canReceiveQuestionText(principal, decision.Binding) {
			identities := make([]aga.BaseIdentity, 0, len(pageRows))
			for _, row := range pageRows {
				if row.baseIdentity != nil {
					identities = append(identities, *row.baseIdentity)
				}
			}
			bodies, bodyErr := composeReviewPage(ctx, service.questionBodies, identities, "")
			if len(identities) > 0 && bodyErr != nil {
				return QueryResponse{}, bodyErr
			}
			byKey := bodyMap(bodies)
			for index := range response.Items {
				row := pageRows[index]
				var text, digest, origin string
				if row.baseIdentity != nil {
					body, found := byKey[row.baseIdentity.Key()]
					if !found {
						return QueryResponse{}, ErrQuestionBodyIncomplete
					}
					text, digest, origin = body.Text, body.TextDigest, "SEALED_BASE"
				} else if row.workspaceVersion != nil && row.workspaceVersion.BodyDigest == aga.ComputeWorkspaceBodyDigest(row.workspaceVersion.Body) && row.workspaceVersion.BodyDigest == row.ref.Workspace.BodyDigest {
					text, digest, origin = row.workspaceVersion.Body, row.workspaceVersion.BodyDigest, "WORKSPACE_AUTHORED"
				} else {
					return QueryResponse{}, ErrQuestionBodyIncomplete
				}
				response.Items[index].QuestionText = &text
				response.Items[index].QuestionTextDigest = &digest
				response.Items[index].TextOrigin = origin
			}
		}
		response.ItemCount = len(filteredRows)
		response.Page = request.Page
		response.PageSize = pageSize
		if end < len(filteredRows) {
			nextPage := request.Page + 1
			response.NextPage = &nextPage
		}
	case OperationGetDraft:
		draft := workspace.Draft.Draft
		response.Draft = &draft
	case OperationGetHistory:
		response.History = []preprod.Generation{workspace.Generation}
	case OperationGetSimulationSetup:
		setup, setupErr := service.simulationSetup(ctx, principal, workspace)
		if setupErr != nil {
			return QueryResponse{}, setupErr
		}
		response.SimulationSetup = &setup
	}
	return response, nil
}

type classificationReviewRow struct {
	item             preprod.ClassificationItem
	ref              aga.QuestionRef
	baseIdentity     *aga.BaseIdentity
	workspaceVersion *preprod.WorkspaceQuestionVersion
}

func (service *Service) classificationReviewRows(ctx context.Context, workspace preprod.LoadedWorkspace) ([]classificationReviewRow, error) {
	rows := make([]classificationReviewRow, 0, len(workspace.Items)+len(workspace.Draft.Draft.Items))
	for _, item := range workspace.Items {
		identity := item.Identity
		rows = append(rows, classificationReviewRow{item: item, ref: aga.BaseQuestionReference(identity), baseIdentity: &identity})
	}
	versions, versionErr := service.listCurrentWorkspaceQuestionVersions(ctx, workspace.Generation.GenerationID)
	if versionErr != nil && hasCurrentWorkspaceDraftItems(workspace.Draft.Draft) {
		return nil, fmt.Errorf("%w: workspace candidate projection: %v", ErrWorkspaceStore, versionErr)
	}
	versionsByID := make(map[string]preprod.WorkspaceQuestionVersion, len(versions))
	for _, version := range versions {
		versionsByID[version.VersionID] = version
	}
	for _, draftItem := range workspace.Draft.Draft.Items {
		if !draftItem.Current || draftItem.QuestionRef.Workspace == nil {
			continue
		}
		version, found := versionsByID[draftItem.QuestionRef.Workspace.VersionID]
		if !found || version.Reference().Key() != draftItem.QuestionRef.Key() || version.BodyDigest != draftItem.QuestionRef.Workspace.BodyDigest || version.BodyDigest != aga.ComputeWorkspaceBodyDigest(version.Body) {
			return nil, ErrQuestionBodyIdentityMismatch
		}
		metadata, found := service.workspaceDraftMetadata(ctx, workspace.Generation.GenerationID, draftItem)
		if !found {
			return nil, ErrQuestionBodyIdentityMismatch
		}
		rows = append(rows, classificationReviewRow{item: metadata, ref: draftItem.QuestionRef, workspaceVersion: &version})
	}
	sort.SliceStable(rows, func(left, right int) bool {
		leftSequence, rightSequence := rows[left].ref.RootSequence, rows[right].ref.RootSequence
		if leftSequence != rightSequence {
			return leftSequence < rightSequence
		}
		return rows[left].ref.Key() < rows[right].ref.Key()
	})
	return rows, nil
}

func hasCurrentWorkspaceDraftItems(draft aga.Draft) bool {
	for _, item := range draft.Items {
		if item.Current && item.QuestionRef.Workspace != nil {
			return true
		}
	}
	return false
}

func reviewMetadataItem(row classificationReviewRow) ClassificationReviewItem {
	return ClassificationReviewItem{ClassificationItem: row.item, QuestionRef: row.ref, QuestionOrigin: string(row.ref.Origin)}
}

func filterClassificationReviewRows(rows []classificationReviewRow, request QueryRequest) []classificationReviewRow {
	filtered := make([]classificationReviewRow, 0, len(rows))
	search := strings.ToLower(strings.TrimSpace(request.Search))
	for _, row := range rows {
		projection := row.item.Projection
		governance := row.item.Governance
		formCode := ""
		proposalID := ""
		ordinal := row.ref.RootSequence
		textDigest := ""
		if row.baseIdentity != nil {
			formCode, proposalID, ordinal, textDigest = row.baseIdentity.FormCode, row.baseIdentity.ProposalID, row.baseIdentity.Ordinal, row.baseIdentity.TextDigest
		}
		if request.DomainCode != "" && projection.MainDomainCode != request.DomainCode || request.TopicCode != "" && !containsString(projection.TopicCodes, request.TopicCode) || request.Confidence != "" && string(row.item.AgreementConfidence) != request.Confidence || !matchesBooleanFilter(request.Blocker, len(governance.BlockerCodes) > 0) || !matchesBooleanFilter(request.SourceGap, governance.QuestionSourceProposalGap) || !matchesBooleanFilter(request.ExternalInvolvement, governance.ExternalApplicabilityUnresolved) {
			continue
		}
		if request.FormCode != "" && formCode != request.FormCode {
			continue
		}
		if request.Disposition != "" {
			if request.Disposition == "UNSET" {
				if row.item.DraftDisposition != nil {
					continue
				}
			} else if row.item.DraftDisposition == nil || *row.item.DraftDisposition != request.Disposition {
				continue
			}
		}
		if search != "" {
			fields := []string{formCode, proposalID, fmt.Sprint(ordinal), textDigest, row.item.QuestionKey, row.ref.Key(), projection.MainDomainCode, projection.TargetProfileCode, row.item.RecommendationState, string(row.item.AgreementConfidence)}
			matched := false
			for _, field := range fields {
				if strings.Contains(strings.ToLower(field), search) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func filterReviewRowsByBodyIdentity(rows []classificationReviewRow, allowed map[string]struct{}, search string) []classificationReviewRow {
	filtered := make([]classificationReviewRow, 0, len(rows))
	for _, row := range rows {
		if row.baseIdentity != nil {
			if _, ok := allowed[row.baseIdentity.Key()]; ok {
				filtered = append(filtered, row)
			}
			continue
		}
		if row.workspaceVersion != nil && strings.Contains(strings.ToLower(row.workspaceVersion.Body), strings.ToLower(search)) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func filterClassificationItems(items []preprod.ClassificationItem, request QueryRequest) []preprod.ClassificationItem {
	filtered := make([]preprod.ClassificationItem, 0, len(items))
	search := strings.ToLower(strings.TrimSpace(request.Search))
	for _, item := range items {
		identity := item.Identity
		projection := item.Projection
		governance := item.Governance
		if request.DomainCode != "" && projection.MainDomainCode != request.DomainCode {
			continue
		}
		if request.TopicCode != "" && !containsString(projection.TopicCodes, request.TopicCode) {
			continue
		}
		if request.Confidence != "" && string(item.AgreementConfidence) != request.Confidence {
			continue
		}
		if !matchesBooleanFilter(request.Blocker, len(governance.BlockerCodes) > 0) ||
			!matchesBooleanFilter(request.SourceGap, governance.QuestionSourceProposalGap) ||
			!matchesBooleanFilter(request.ExternalInvolvement, governance.ExternalApplicabilityUnresolved) {
			continue
		}
		if request.FormCode != "" && identity.FormCode != request.FormCode {
			continue
		}
		if request.Disposition != "" {
			if request.Disposition == "UNSET" {
				if item.DraftDisposition != nil {
					continue
				}
			} else if item.DraftDisposition == nil || stringValue(*item.DraftDisposition) != request.Disposition {
				continue
			}
		}
		if search != "" {
			fields := []string{
				identity.FormCode,
				identity.ProposalID,
				identity.TextDigest,
				projection.MainDomainCode,
				projection.TargetProfileCode,
				item.RecommendationState,
				string(item.AgreementConfidence),
			}
			matched := false
			for _, field := range fields {
				if strings.Contains(strings.ToLower(field), search) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	sort.SliceStable(filtered, func(left, right int) bool {
		if filtered[left].Identity.Ordinal == filtered[right].Identity.Ordinal {
			return filtered[left].Identity.Key() < filtered[right].Identity.Key()
		}
		return filtered[left].Identity.Ordinal < filtered[right].Identity.Ordinal
	})
	return filtered
}

func stringValue(value string) string { return value }

func filterItemsByBaseIdentity(items []preprod.ClassificationItem, allowed map[string]struct{}) []preprod.ClassificationItem {
	filtered := make([]preprod.ClassificationItem, 0, len(items))
	for _, item := range items {
		if _, ok := allowed[item.Identity.Key()]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func matchesBooleanFilter(filter string, value bool) bool {
	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "", "all":
		return true
	case "true":
		return value
	case "false":
		return !value
	default:
		return false
	}
}

type runtimeCommandSnapshotter interface {
	SnapshotForRuntimeCommand(context.Context) (preprod.LoadedWorkspace, error)
}

type runtimeDraftCommandApplier interface {
	ApplyDraftCommandFromRuntime(context.Context, aga.Draft, aga.DraftCommand) (aga.Draft, error)
}

func (service *Service) Command(ctx context.Context, principal identity.Principal, family OperationFamily, command CommandEnvelope) (CommandResponse, error) {
	if _, err := service.Authorize(ctx, principal, command.OperationID); err != nil {
		return CommandResponse{}, err
	}
	if err := command.Validate(family); err != nil {
		return CommandResponse{}, err
	}
	if family == FamilyRecommendationCommand {
		if err := validateRecommendationCommand(command); err != nil {
			return CommandResponse{}, err
		}
	}
	if family == FamilyLifecycleCommand {
		if err := validateLifecycleCommand(command); err != nil {
			return CommandResponse{}, err
		}
	}
	if family == FamilyAdminCommand && command.OperationID != OperationResetGeneration {
		return CommandResponse{}, ErrNeutralDenied
	}
	if family == FamilyClassificationCommand && !isClassificationCommand(command.OperationID, command.Action) {
		return CommandResponse{}, ErrNeutralDenied
	}
	if service == nil || service.command == nil {
		return CommandResponse{}, ErrWorkspaceStore
	}
	decision, err := service.Authorize(ctx, principal, command.OperationID)
	if err != nil {
		return CommandResponse{}, err
	}
	clientCommand := command
	commandHash := commandDigest(clientCommand)
	key := preprod.StoredResponseKey{GenerationID: command.ExpectedGenerationID, ActorSubjectID: principal.SubjectID, OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey}
	stored, found, err := service.command.GetIdempotencyResponse(ctx, key)
	if err != nil {
		return CommandResponse{}, fmt.Errorf("%w: idempotency lookup: %v", ErrWorkspaceStore, err)
	}
	if found {
		if stored.CommandHash != commandHash || stored.AuthorizationScopeDigest != decision.ScopeDigest {
			return CommandResponse{}, preprod.ErrWorkspaceIdempotency
		}
		var replay CommandResponse
		if err := json.Unmarshal(stored.Response, &replay); err != nil {
			return CommandResponse{}, fmt.Errorf("%w: stored response: %v", ErrWorkspaceStore, err)
		}
		replay.Replayed = true
		service.updateReaderDraftCache(command.ExpectedGenerationID, replay.Draft)
		return replay, nil
	}
	var workspace preprod.LoadedWorkspace
	// The command store's runtime projection is complete once its immutable
	// sealed workspace base has been loaded. It reuses the sealed items,
	// fixture, and authority projection while reading only the mutable Draft/CAS
	// state, so setup-bound classification commands do not rebuild 1,310 rows
	// for every batch preview and execute.
	if family == FamilyClassificationCommand {
		if runtimeStore, ok := service.command.(runtimeCommandSnapshotter); ok {
			workspace, err = runtimeStore.SnapshotForRuntimeCommand(ctx)
		} else {
			workspace, err = service.command.Snapshot(ctx)
		}
	} else {
		workspace, err = service.command.Snapshot(ctx)
	}
	if err != nil {
		return CommandResponse{}, fmt.Errorf("%w: snapshot: %v", ErrWorkspaceStore, err)
	}
	if family == FamilyRecommendationCommand && workspace.Run.Result.ClassificationRunID != "" {
		hydrated, hydrateErr := aga.HydrateDraftForRuntime(workspace.Draft.Draft, workspace.Run.Result)
		if hydrateErr != nil {
			return CommandResponse{}, fmt.Errorf("%w: hydrate sealed draft context: %v", ErrWorkspaceStore, hydrateErr)
		}
		workspace.Draft.Draft = hydrated
	}
	if workspace.Generation.GenerationID != command.ExpectedGenerationID || workspace.Generation.State != preprod.GenerationActive {
		return CommandResponse{}, preprod.ErrWorkspaceCAS
	}
	if command.SetupDigest != "" {
		command, err = service.hydrateSetupCommand(ctx, principal, workspace, command)
		if err != nil {
			return CommandResponse{}, err
		}
		if family == FamilyRecommendationCommand {
			if err := validateRecommendationCommand(command); err != nil {
				return CommandResponse{}, err
			}
		}
	}
	var response CommandResponse
	switch family {
	case FamilyAdminCommand:
		generation, tombstone, resetErr := service.command.ResetGeneration(ctx, preprod.ResetInput{ExpectedGenerationID: command.ExpectedGenerationID, ExpectedGenerationRevision: command.ExpectedGenerationRevision, ExpectedGenerationSealDigest: command.ExpectedGenerationSealDigest, ReasonCode: command.ReasonCode, ActorSubjectID: principal.SubjectID, Now: service.clock().UTC()})
		if resetErr != nil {
			return CommandResponse{}, resetErr
		}
		response = CommandResponse{OperationID: command.OperationID, Generation: &generation, ResetTombstone: &tombstone}
		service.invalidateReaderSnapshotCache()
	case FamilyClassificationCommand:
		if command.OperationID == OperationPreviewBatch || command.OperationID == OperationExecuteBatch {
			response, batchErr := service.batchCommand(ctx, principal, workspace, command)
			if batchErr != nil {
				return CommandResponse{}, batchErr
			}
			responseBytes, _ := json.Marshal(response)
			storedResponse := preprod.IdempotencyResponse{GenerationID: command.ExpectedGenerationID, ActorSubjectID: principal.SubjectID, OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey, CommandHash: commandHash, AuthorizationScopeDigest: decision.ScopeDigest, StatusCode: 200, Response: responseBytes, CreatedAt: service.clock().UTC()}
			if _, _, err := service.command.PutIdempotencyResponse(ctx, storedResponse); err != nil {
				return CommandResponse{}, fmt.Errorf("%w: idempotency commit: %v", ErrWorkspaceStore, err)
			}
			service.updateReaderDraftCache(command.ExpectedGenerationID, response.Draft)
			return response, nil
		}
		draftCommand := aga.DraftCommand{OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey, ExpectedGenerationID: command.ExpectedGenerationID, Action: command.Action, TargetQuestionKey: command.TargetQuestionKey, ExpectedRevision: command.ExpectedDraftRevision, ExpectedContentDigest: command.ExpectedDraftContentDigest, ReasonCode: command.ReasonCode, ActorSubjectID: principal.SubjectID, CreatedAt: service.clock().UTC(), MainDomainCode: command.MainDomainCode, TopicCode: command.TopicCode, ResolutionMode: command.ResolutionMode, ExactProjection: command.ExactProjection, WorkspaceBody: command.WorkspaceBody, WorkspaceBodyDigest: command.WorkspaceBodyDigest, ReadinessEventID: command.ReadinessEventID, ProviderScopeProfileDigest: command.ProviderScopeProfileDigest}
		if draftCommand.Action == "" {
			draftCommand.Action = aga.DraftAction(command.OperationID)
		}
		var updated aga.Draft
		var applyErr error
		if runtimeStore, ok := service.command.(runtimeDraftCommandApplier); ok {
			updated, applyErr = runtimeStore.ApplyDraftCommandFromRuntime(ctx, workspace.Draft.Draft, draftCommand)
		} else {
			updated, applyErr = service.command.ApplyDraftCommand(ctx, draftCommand)
		}
		if applyErr != nil {
			if errors.Is(applyErr, aga.ErrNonCurrentQuestion) || errors.Is(applyErr, aga.ErrMissingParent) || errors.Is(applyErr, aga.ErrCrossRootParent) || errors.Is(applyErr, aga.ErrCrossGenerationParent) || errors.Is(applyErr, aga.ErrWorkspaceIdentityAlias) {
				return CommandResponse{}, ErrNeutralDenied
			}
			return CommandResponse{}, applyErr
		}
		response = CommandResponse{OperationID: command.OperationID, Draft: &updated}
	case FamilyRecommendationCommand:
		if command.OperationID == OperationCreateRecommendation {
			recommendation, replayed, recommendationErr := service.createRecommendation(ctx, principal, workspace, command)
			if recommendationErr != nil {
				return CommandResponse{}, recommendationErr
			}
			response = CommandResponse{OperationID: command.OperationID, Recommendation: &recommendation, Replayed: replayed}
		} else if command.OperationID == OperationCreateInspection {
			aggregate, inspectionErr := service.createInspection(ctx, principal, workspace, command)
			if inspectionErr != nil {
				return CommandResponse{}, inspectionErr
			}
			projection := ProjectLifecycle(aggregate, principal)
			response = CommandResponse{OperationID: command.OperationID, Lifecycle: &projection}
		} else {
			return CommandResponse{}, ErrCapabilityUnavailable
		}
	case FamilyLifecycleCommand:
		aggregate, lifecycleErr := service.lifecycleCommand(ctx, principal, command)
		if lifecycleErr != nil {
			return CommandResponse{}, lifecycleErr
		}
		projection := ProjectLifecycle(aggregate, principal)
		response = CommandResponse{OperationID: command.OperationID, Lifecycle: &projection}
	default:
		return CommandResponse{}, ErrCapabilityUnavailable
	}
	responseBytes, _ := json.Marshal(response)
	storedResponse := preprod.IdempotencyResponse{GenerationID: command.ExpectedGenerationID, ActorSubjectID: principal.SubjectID, OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey, CommandHash: commandHash, AuthorizationScopeDigest: decision.ScopeDigest, StatusCode: 200, Response: responseBytes, CreatedAt: service.clock().UTC()}
	if _, _, err := service.command.PutIdempotencyResponse(ctx, storedResponse); err != nil {
		return CommandResponse{}, fmt.Errorf("%w: idempotency commit: %v", ErrWorkspaceStore, err)
	}
	service.updateReaderDraftCache(command.ExpectedGenerationID, response.Draft)
	return response, nil
}

func (service *Service) SetNonTerminalLifecycle(value bool) {
	if setter, ok := service.store.(interface{ SetNonTerminalLifecycle(bool) }); ok {
		setter.SetNonTerminalLifecycle(value)
	}
}

func isClassificationQuery(operation string) bool {
	switch operation {
	case OperationGetSummary, OperationGetTaxonomy, OperationGetProviderConfiguration, OperationSearchItems, OperationGetDraft, OperationGetHistory, OperationGetSimulationSetup:
		return true
	default:
		return false
	}
}

func isRecommendationQuery(operation string) bool {
	return operation == OperationGetCurrentRecommendation
}

func isLifecycleQuery(operation string) bool {
	return operation == OperationGetInspection || operation == OperationGetCurrentInspection || operation == OperationGetInspectionQuestionPage || operation == OperationGetFinding || operation == OperationGetCAPEvidence || operation == OperationGetRoleHistory
}

func isClassificationCommand(operation string, action aga.DraftAction) bool {
	switch operation {
	case OperationPreviewBatch, OperationExecuteBatch, OperationRetain, OperationReclassify, OperationAddTopic, OperationRemoveTopic, OperationResolve, OperationInclude, OperationExclude, OperationDefer, OperationAddCandidate, OperationReword, OperationMarkReady:
		return true
	default:
		return action != ""
	}
}

func draftDispositionCounts(draft aga.Draft) (included, excluded, deferred int) {
	for _, item := range draft.Items {
		if !item.Current || item.Disposition == nil {
			continue
		}
		switch *item.Disposition {
		case aga.DispositionInclude:
			included++
		case aga.DispositionExclude:
			excluded++
		case aga.DispositionDefer:
			deferred++
		}
	}
	return included, excluded, deferred
}

func commandDigest(command CommandEnvelope) string {
	data, _ := json.Marshal(command)
	hash := sha256.Sum256(append([]byte("AGA-DEMO-WORKSPACE-COMMAND-V1\n"), data...))
	return "sha256:" + hex.EncodeToString(hash[:])
}
