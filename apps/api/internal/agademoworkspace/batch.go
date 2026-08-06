package agademoworkspace

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

const selectionBatchPreviewLifetime = 10 * time.Minute

func normalizeBatchFilter(filter BatchFilter) (BatchFilter, error) {
	filter.Search = strings.TrimSpace(filter.Search)
	filter.DomainCode = strings.TrimSpace(filter.DomainCode)
	filter.TopicCode = strings.TrimSpace(filter.TopicCode)
	filter.Confidence = strings.ToUpper(strings.TrimSpace(filter.Confidence))
	filter.Blocker = strings.ToLower(strings.TrimSpace(filter.Blocker))
	filter.SourceGap = strings.ToLower(strings.TrimSpace(filter.SourceGap))
	filter.ExternalInvolvement = strings.ToLower(strings.TrimSpace(filter.ExternalInvolvement))
	filter.FormCode = strings.TrimSpace(filter.FormCode)
	filter.Disposition = strings.ToUpper(strings.TrimSpace(filter.Disposition))
	if len(filter.Search) > 200 || !validBooleanFilter(filter.Blocker) || !validBooleanFilter(filter.SourceGap) || !validBooleanFilter(filter.ExternalInvolvement) {
		return BatchFilter{}, ErrMalformedCommand
	}
	if filter.Confidence != "" && filter.Confidence != string(aga.ConfidenceHigh) && filter.Confidence != string(aga.ConfidenceMedium) && filter.Confidence != string(aga.ConfidenceLow) {
		return BatchFilter{}, ErrMalformedCommand
	}
	if filter.Disposition != "" && filter.Disposition != aga.DispositionInclude && filter.Disposition != aga.DispositionExclude && filter.Disposition != aga.DispositionDefer && filter.Disposition != "UNSET" {
		return BatchFilter{}, ErrMalformedCommand
	}
	return filter, nil
}

func validBooleanFilter(value string) bool {
	return value == "" || value == "all" || value == "true" || value == "false"
}

func batchFilterRequest(filter BatchFilter) QueryRequest {
	return QueryRequest{OperationID: OperationSearchItems, Search: filter.Search, DomainCode: filter.DomainCode, TopicCode: filter.TopicCode, Confidence: filter.Confidence, Blocker: filter.Blocker, SourceGap: filter.SourceGap, ExternalInvolvement: filter.ExternalInvolvement, FormCode: filter.FormCode, Disposition: filter.Disposition}
}

func batchFilterJSON(filter BatchFilter) (string, string, error) {
	encoded, err := json.Marshal(filter)
	if err != nil {
		return "", "", err
	}
	digest, err := aga.DigestExcludingJSONFields("AGA-DEMO-BATCH-FILTER-V1", filter)
	if err != nil || digest == "" {
		return "", "", ErrBatchPreviewConflict
	}
	return string(encoded), digest, nil
}

func batchAction(value BatchAction) (aga.DraftAction, error) {
	switch value {
	case BatchInclude:
		return aga.DraftInclude, nil
	case BatchExclude:
		return aga.DraftExclude, nil
	case BatchDefer:
		return aga.DraftDefer, nil
	default:
		return "", ErrMalformedCommand
	}
}

func batchIdentityDigest(item preprod.ClassificationItem) string {
	digest, _ := aga.DigestExcludingJSONFields("AGA-DEMO-BATCH-QUESTION-IDENTITY-V1", item.Identity)
	return digest
}

func batchQuestionKey(item preprod.ClassificationItem) string {
	return aga.BaseQuestionReference(item.Identity).Key()
}

func batchItemsDigest(items []preprod.ClassificationItem) string {
	digests := make([]string, 0, len(items))
	for _, item := range items {
		digests = append(digests, batchIdentityDigest(item))
	}
	sort.Strings(digests)
	digest, _ := aga.DigestValue("AGA-DEMO-BATCH-AFFECTED-IDENTITIES-V1", digests)
	return digest
}

func (service *Service) filteredBatchItems(ctx context.Context, workspace preprod.LoadedWorkspace, filter BatchFilter) ([]preprod.ClassificationItem, error) {
	metadataFilter := batchFilterRequest(filter)
	if filter.Search != "" && service.questionTextSearch != nil {
		// Search is resolved against sealed text below. Applying the body
		// fragment to metadata first would silently remove every valid match.
		metadataFilter.Search = ""
	}
	items := filterClassificationItems(workspace.Items, metadataFilter)
	if filter.Search == "" {
		return items, nil
	}
	if service.questionTextSearch == nil {
		return nil, ErrQuestionBodyResolverUnavailable
	}
	search, err := normalizeBodySearch(filter.Search)
	if err != nil {
		return nil, err
	}
	identities, err := service.questionTextSearch.Search(ctx, search)
	if err != nil {
		return nil, fmt.Errorf("%w: body search: %v", ErrQuestionBodyResolverUnavailable, err)
	}
	allowed := make(map[string]struct{}, len(identities))
	for _, identity := range identities {
		if err := identity.Validate(); err != nil {
			return nil, ErrQuestionBodyIdentityMismatch
		}
		allowed[identity.Key()] = struct{}{}
	}
	return filterItemsByBaseIdentity(items, allowed), nil
}

func dispositionCounts(items []preprod.ClassificationItem) BatchDispositionCounts {
	counts := BatchDispositionCounts{}
	for _, item := range items {
		if item.DraftDisposition == nil {
			counts.Unset++
			continue
		}
		switch *item.DraftDisposition {
		case aga.DispositionInclude:
			counts.Include++
		case aga.DispositionExclude:
			counts.Exclude++
		case aga.DispositionDefer:
			counts.Defer++
		default:
			counts.Unset++
		}
	}
	return counts
}

func eligibleForSimulation(item preprod.ClassificationItem, setup SimulationSetupProjection) bool {
	projection := item.Projection
	if projection.CanonicalTargetKind != setup.CanonicalTargetKind || projection.TargetProfileCode != setup.TargetProfileCode || !containsString(projection.InspectionProfileCodes, setup.InspectionProfileCode) || !containsString(projection.InspectionTypeCodes, setup.InspectionTypeCode) || !qualifiersEqual(projection.OperationQualifiers, setup.OperationQualifiers) || !qualifiersEqual(projection.ActivityQualifiers, setup.ActivityQualifiers) {
		return false
	}
	switch projection.ApplicabilityDisposition {
	case "APPLICABLE", "CONDITIONAL_ON_CONFIGURATION", "CONDITIONAL_ON_FACILITY", "CONDITIONAL_ON_OPERATION":
		return true
	default:
		return false
	}
}

func qualifiersEqual(left, right []aga.Qualifier) bool {
	leftCopy := append([]aga.Qualifier(nil), left...)
	rightCopy := append([]aga.Qualifier(nil), right...)
	sort.Slice(leftCopy, func(i, j int) bool {
		return leftCopy[i].Key+"\x00"+leftCopy[i].Value < leftCopy[j].Key+"\x00"+leftCopy[j].Value
	})
	sort.Slice(rightCopy, func(i, j int) bool {
		return rightCopy[i].Key+"\x00"+rightCopy[i].Value < rightCopy[j].Key+"\x00"+rightCopy[j].Value
	})
	if len(leftCopy) != len(rightCopy) {
		return false
	}
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
	}
	return true
}

func previewProjection(record preprod.SelectionBatchPreviewRecord, items []preprod.ClassificationItem, setup SimulationSetupProjection, filter BatchFilter) BatchPreviewProjection {
	eligible, ineligible, blockers, sourceGaps := 0, 0, 0, 0
	for _, item := range items {
		if eligibleForSimulation(item, setup) {
			eligible++
		} else {
			ineligible++
		}
		if len(item.Governance.BlockerCodes) > 0 {
			blockers++
		}
		if item.Governance.QuestionSourceProposalGap {
			sourceGaps++
		}
	}
	projection := BatchPreviewProjection{
		PreviewID: record.PreviewID, GenerationID: record.GenerationID, DraftID: record.DraftID,
		DraftRevision: record.DraftRevision, DraftContentDigest: record.DraftContentDigest,
		ClassificationRunDigest: record.ClassificationRunDigest, Filter: filter, FilterDigest: record.FilterDigest,
		AffectedIdentityDigest: record.AffectedIdentityDigest, Action: BatchAction(record.Action), ReasonCode: record.ReasonCode,
		Count: len(items), CurrentDisposition: dispositionCounts(items), EligibleCount: eligible, IneligibleCount: ineligible,
		BlockerCount: blockers, SourceGapCount: sourceGaps, ExpiresAt: record.ExpiresAt.UTC().Format(time.RFC3339Nano), PreviewDigest: record.PreviewDigest,
	}
	if record.ConsumedAt != nil {
		value := record.ConsumedAt.UTC().Format(time.RFC3339Nano)
		projection.ConsumedAt = &value
	}
	return projection
}

func (service *Service) resolveBatchSetup(ctx context.Context, principal identity.Principal, workspace preprod.LoadedWorkspace, digest string) (SimulationSetupProjection, error) {
	if strings.TrimSpace(digest) == "" {
		return SimulationSetupProjection{}, ErrRecommendationFactsUnavailable
	}
	setup, err := service.simulationSetup(ctx, principal, workspace)
	if err != nil || setup.SimulationSetupDigest != digest {
		return SimulationSetupProjection{}, ErrRecommendationFactsUnavailable
	}
	return setup, nil
}

func previewRecordDigest(record preprod.SelectionBatchPreviewRecord) string {
	digest, _ := aga.DigestExcludingJSONFields("AGA-DEMO-BATCH-PREVIEW-V1", record, "previewDigest", "consumedAt")
	return digest
}

func previewIDFor(record preprod.SelectionBatchPreviewRecord) string {
	digest, _ := aga.DigestExcludingJSONFields("AGA-DEMO-BATCH-PREVIEW-ID-V1", struct {
		GenerationID string
		DraftID      string
		DraftDigest  string
		FilterDigest string
		Action       string
		ReasonCode   string
		Affected     string
	}{record.GenerationID, record.DraftID, record.DraftContentDigest, record.FilterDigest, record.Action, record.ReasonCode, record.AffectedIdentityDigest})
	if len(digest) > len("sha256:")+24 {
		digest = digest[len("sha256:") : len("sha256:")+24]
	}
	return "aga-ws-preview-" + digest
}

func (service *Service) previewSelectionBatch(ctx context.Context, principal identity.Principal, workspace preprod.LoadedWorkspace, command CommandEnvelope) (BatchPreviewProjection, error) {
	if command.BatchFilter == nil {
		return BatchPreviewProjection{}, ErrMalformedCommand
	}
	filter, err := normalizeBatchFilter(*command.BatchFilter)
	if err != nil {
		return BatchPreviewProjection{}, err
	}
	if _, err := batchAction(command.BatchAction); err != nil || strings.TrimSpace(command.ReasonCode) == "" {
		return BatchPreviewProjection{}, ErrMalformedCommand
	}
	setup, err := service.resolveBatchSetup(ctx, principal, workspace, command.SetupDigest)
	if err != nil {
		return BatchPreviewProjection{}, err
	}
	items, err := service.filteredBatchItems(ctx, workspace, filter)
	if err != nil {
		return BatchPreviewProjection{}, err
	}
	if len(items) > MaxBatchPreviewSize {
		return BatchPreviewProjection{}, ErrBatchPreviewTooLarge
	}
	filterJSON, filterDigest, err := batchFilterJSON(filter)
	if err != nil {
		return BatchPreviewProjection{}, err
	}
	record := preprod.SelectionBatchPreviewRecord{GenerationID: workspace.Generation.GenerationID, DraftID: workspace.Draft.Draft.DraftID, DraftRevision: workspace.Draft.Draft.Revision, DraftContentDigest: workspace.Draft.Draft.ContentDigest, ClassificationRunDigest: workspace.Run.ClassificationRunDigest, FilterJSON: filterJSON, FilterDigest: filterDigest, AffectedIdentityDigest: batchItemsDigest(items), Action: string(command.BatchAction), ReasonCode: strings.TrimSpace(command.ReasonCode), ExpiresAt: service.clock().UTC().Add(selectionBatchPreviewLifetime)}
	record.Items = make([]preprod.SelectionBatchPreviewItem, 0, len(items))
	for _, item := range items {
		record.Items = append(record.Items, preprod.SelectionBatchPreviewItem{QuestionKey: batchQuestionKey(item), IdentityDigest: batchIdentityDigest(item)})
	}
	record.PreviewID = previewIDFor(record)
	record.PreviewDigest = previewRecordDigest(record)
	store, ok := service.command.(preprod.BatchPreviewStore)
	if !ok {
		return BatchPreviewProjection{}, ErrCapabilityUnavailable
	}
	stored, _, err := store.PutSelectionBatchPreview(ctx, record)
	if err != nil {
		return BatchPreviewProjection{}, err
	}
	return previewProjection(stored, items, setup, filter), nil
}

func decodePreviewFilter(record preprod.SelectionBatchPreviewRecord) (BatchFilter, error) {
	var filter BatchFilter
	if err := json.Unmarshal([]byte(record.FilterJSON), &filter); err != nil {
		return BatchFilter{}, ErrBatchPreviewConflict
	}
	normalized, err := normalizeBatchFilter(filter)
	if err != nil {
		return BatchFilter{}, ErrBatchPreviewConflict
	}
	_, digest, err := batchFilterJSON(normalized)
	if err != nil || digest != record.FilterDigest {
		return BatchFilter{}, ErrBatchPreviewConflict
	}
	return normalized, nil
}

func sameBatchItems(record preprod.SelectionBatchPreviewRecord, items []preprod.ClassificationItem) bool {
	if len(record.Items) != len(items) || record.AffectedIdentityDigest != batchItemsDigest(items) {
		return false
	}
	for index, item := range items {
		if record.Items[index].QuestionKey != batchQuestionKey(item) || record.Items[index].IdentityDigest != batchIdentityDigest(item) {
			return false
		}
	}
	return true
}

func (service *Service) executeSelectionBatch(ctx context.Context, principal identity.Principal, workspace preprod.LoadedWorkspace, command CommandEnvelope) (BatchPreviewProjection, *aga.Draft, error) {
	if err := validateBatchPreviewConsume(BatchPreviewConsume{PreviewID: command.PreviewID, PreviewDigest: command.PreviewDigest}); err != nil {
		return BatchPreviewProjection{}, nil, err
	}
	store, ok := service.command.(preprod.BatchPreviewStore)
	if !ok {
		return BatchPreviewProjection{}, nil, ErrCapabilityUnavailable
	}
	record, found, err := store.GetSelectionBatchPreview(ctx, workspace.Generation.GenerationID, command.PreviewID)
	if err != nil || !found {
		return BatchPreviewProjection{}, nil, ErrBatchPreviewNotFound
	}
	if record.PreviewDigest != command.PreviewDigest || record.ConsumedAt != nil || !service.clock().UTC().Before(record.ExpiresAt) || record.DraftRevision != workspace.Draft.Draft.Revision || record.DraftContentDigest != workspace.Draft.Draft.ContentDigest || record.DraftID != workspace.Draft.Draft.DraftID {
		return BatchPreviewProjection{}, nil, ErrBatchPreviewConflict
	}
	filter, err := decodePreviewFilter(record)
	if err != nil {
		return BatchPreviewProjection{}, nil, err
	}
	if command.BatchFilter != nil {
		provided, normalizeErr := normalizeBatchFilter(*command.BatchFilter)
		if normalizeErr != nil || provided != filter {
			return BatchPreviewProjection{}, nil, ErrBatchPreviewConflict
		}
	}
	if command.BatchAction != "" && string(command.BatchAction) != record.Action || command.ReasonCode != "" && strings.TrimSpace(command.ReasonCode) != record.ReasonCode {
		return BatchPreviewProjection{}, nil, ErrBatchPreviewConflict
	}
	setup, err := service.resolveBatchSetup(ctx, principal, workspace, command.SetupDigest)
	if err != nil {
		return BatchPreviewProjection{}, nil, err
	}
	items, err := service.filteredBatchItems(ctx, workspace, filter)
	if err != nil || !sameBatchItems(record, items) {
		return BatchPreviewProjection{}, nil, ErrBatchPreviewConflict
	}
	if BatchAction(record.Action) == BatchInclude {
		for _, item := range items {
			if !eligibleForSimulation(item, setup) {
				return BatchPreviewProjection{}, nil, ErrIncludedQuestionIneligible
			}
		}
	}
	action, err := batchAction(BatchAction(record.Action))
	if err != nil {
		return BatchPreviewProjection{}, nil, err
	}
	commands := make([]aga.DraftCommand, 0, len(items))
	for index, item := range items {
		commands = append(commands, aga.DraftCommand{OperationID: OperationExecuteBatch, IdempotencyKey: fmt.Sprintf("%s-batch-%04d", command.IdempotencyKey, index), ExpectedGenerationID: workspace.Generation.GenerationID, ExpectedRevision: workspace.Draft.Draft.Revision, ExpectedContentDigest: workspace.Draft.Draft.ContentDigest, Action: action, TargetQuestionKey: batchQuestionKey(item), ReasonCode: record.ReasonCode, ActorSubjectID: principal.SubjectID, CreatedAt: service.clock().UTC()})
	}
	executor, ok := service.command.(preprod.SelectionBatchExecutor)
	if !ok {
		return BatchPreviewProjection{}, nil, ErrCapabilityUnavailable
	}
	updated, consumed, err := executor.ExecuteSelectionBatch(ctx, record, workspace.Draft.Draft, commands, service.clock().UTC())
	if err != nil {
		return BatchPreviewProjection{}, nil, err
	}
	return previewProjection(consumed, items, setup, filter), &updated, nil
}

func (service *Service) batchCommand(ctx context.Context, principal identity.Principal, workspace preprod.LoadedWorkspace, command CommandEnvelope) (CommandResponse, error) {
	var projection BatchPreviewProjection
	var draft *aga.Draft
	var err error
	if command.OperationID == OperationPreviewBatch {
		projection, err = service.previewSelectionBatch(ctx, principal, workspace, command)
	} else {
		projection, draft, err = service.executeSelectionBatch(ctx, principal, workspace, command)
	}
	if err != nil {
		return CommandResponse{}, err
	}
	return CommandResponse{OperationID: command.OperationID, BatchPreview: &projection, Draft: draft}, nil
}
