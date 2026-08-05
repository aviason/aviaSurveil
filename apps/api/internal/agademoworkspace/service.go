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
	return &Service{store: config.Store, reader: readerStore, command: commandStore, resolver: config.Resolver, recommendationScopes: config.RecommendationScopes, lifecycleBindings: config.LifecycleBindings, clock: clock}
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
		filtered := filterClassificationItems(workspace.Items, request)
		pageSize := request.PageSize
		if pageSize == 0 {
			pageSize = 25
		}
		start := request.Page * pageSize
		if start > len(filtered) {
			start = len(filtered)
		}
		end := start + pageSize
		if end > len(filtered) {
			end = len(filtered)
		}
		response.Items = append([]preprod.ClassificationItem(nil), filtered[start:end]...)
		response.ItemCount = len(filtered)
		response.Page = request.Page
		response.PageSize = pageSize
		if end < len(filtered) {
			nextPage := request.Page + 1
			response.NextPage = &nextPage
		}
	case OperationGetDraft:
		draft := workspace.Draft.Draft
		response.Draft = &draft
	case OperationGetHistory:
		response.History = []preprod.Generation{workspace.Generation}
	}
	return response, nil
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
	commandHash := commandDigest(command)
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
		return replay, nil
	}
	var workspace preprod.LoadedWorkspace
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
	var response CommandResponse
	switch family {
	case FamilyAdminCommand:
		generation, tombstone, resetErr := service.command.ResetGeneration(ctx, preprod.ResetInput{ExpectedGenerationID: command.ExpectedGenerationID, ExpectedGenerationRevision: command.ExpectedGenerationRevision, ExpectedGenerationSealDigest: command.ExpectedGenerationSealDigest, ReasonCode: command.ReasonCode, ActorSubjectID: principal.SubjectID, Now: service.clock().UTC()})
		if resetErr != nil {
			return CommandResponse{}, resetErr
		}
		response = CommandResponse{OperationID: command.OperationID, Generation: &generation, ResetTombstone: &tombstone}
	case FamilyClassificationCommand:
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
	return response, nil
}

func (service *Service) SetNonTerminalLifecycle(value bool) {
	if setter, ok := service.store.(interface{ SetNonTerminalLifecycle(bool) }); ok {
		setter.SetNonTerminalLifecycle(value)
	}
}

func isClassificationQuery(operation string) bool {
	switch operation {
	case OperationGetSummary, OperationGetTaxonomy, OperationGetProviderConfiguration, OperationSearchItems, OperationGetDraft, OperationGetHistory:
		return true
	default:
		return false
	}
}

func isLifecycleQuery(operation string) bool {
	return operation == OperationGetInspection || operation == OperationGetFinding || operation == OperationGetCAPEvidence || operation == OperationGetRoleHistory
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
