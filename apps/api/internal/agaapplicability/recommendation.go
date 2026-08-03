package agaapplicability

import (
	"reflect"
	"sort"
)

func BuildRecommendation(draft Draft, scope ProviderScopeFact, request RecommendationRequest) (Recommendation, error) {
	if request.DraftRevision != draft.Revision || request.ExpectedDraftRevision != draft.Revision || request.DraftContentDigest != draft.ContentDigest || request.DraftID != draft.DraftID {
		return Recommendation{}, ErrDraftConflict
	}
	if draft.State != DraftReadyForDemoSimulation {
		return Recommendation{}, ErrDraftNotReady
	}
	if ComputeDraftContentDigest(draft) != draft.ContentDigest || validateDraftQuestionGraph(draft) != nil || validateReadinessDraft(draft, readinessReason(draft)) != nil {
		return Recommendation{}, ErrDraftNotReady
	}
	readiness, ok := resolveReadinessEvent(draft, request.ReadinessEventID)
	if !ok || request.ReadinessEventDigest != readiness.ReadinessEventDigest || readiness.ReadinessEventDigest != digestExcludingJSONFields("AGA-DEMO-READINESS-EVENT-V1", readiness, "readinessEventDigest") ||
		readiness.GenerationID != draft.GenerationID || readiness.ClassificationRunID != draft.ClassificationRunID || readiness.ClassificationRunDigest != draft.ClassificationRunDigest ||
		readiness.TaxonomyVersion != draft.TaxonomyVersion || readiness.TaxonomyDigest != draft.TaxonomyDigest || readiness.DraftID != draft.DraftID || readiness.DraftRevision != draft.Revision || readiness.DraftContentDigest != draft.ContentDigest {
		return Recommendation{}, ErrReadinessPinMismatch
	}
	if request.OperationID == "" || request.IdempotencyKey == "" || request.ExpectedGenerationID != draft.GenerationID || request.TaxonomyVersion != draft.TaxonomyVersion || request.TaxonomyDigest != draft.TaxonomyDigest || request.ClassificationRunID != draft.ClassificationRunID || request.ClassificationRunDigest != draft.ClassificationRunDigest {
		return Recommendation{}, ErrReadinessPinMismatch
	}
	computedScopeDigest := ComputeProviderScopeProfileDigest(scope)
	if !validDigest(scope.ProfileDigest) || computedScopeDigest == "" || scope.ProfileDigest != computedScopeDigest || computedScopeDigest != readiness.ProviderScopeProfileDigest {
		return Recommendation{}, ErrReadinessPinMismatch
	}
	if scope.GenerationID != draft.GenerationID || request.OrganizationID == "" || request.OrganizationID != scope.OrganizationID || request.ProviderScopeRootID == "" || request.ProviderScopeRootID != scope.ProviderScopeRootID || request.ProviderScopeID == "" || request.ProviderScopeID != scope.ProviderScopeID || request.ProviderScopeVersion < 1 || request.ProviderScopeVersion != scope.ProviderScopeVersion || request.ProviderTypeID == "" || request.ProviderTypeID != scope.ProviderTypeID || request.DepartmentID == "" || request.DepartmentID != scope.DepartmentID || request.OrganizationalUnitID == "" || request.OrganizationalUnitID != scope.OrganizationalUnitID || scope.ProviderTypeCode != "AERODROME_OPERATOR" {
		return Recommendation{}, ErrProviderScopeMismatch
	}
	if scope.Status != "ACTIVE" || request.EffectiveAt.IsZero() || request.EffectiveAt.Before(scope.EffectiveFrom) || (scope.EffectiveTo != nil && !request.EffectiveAt.Before(*scope.EffectiveTo)) {
		return Recommendation{}, ErrProviderScopeNotApplicable
	}
	if !qualifiersEqual(scope.OperationQualifiers, request.OperationQualifiers) || !qualifiersEqual(scope.ActivityQualifiers, request.ActivityQualifiers) {
		return Recommendation{}, ErrQualifierMismatch
	}
	var target *TypedTarget
	for index := range scope.Targets {
		if scope.Targets[index].ID == request.TargetID {
			if target != nil {
				return Recommendation{}, ErrTargetMismatch
			}
			copy := scope.Targets[index]
			target = &copy
		}
	}
	if target == nil || target.Kind != request.CanonicalTargetKind || target.ProfileCode != request.TargetProfileCode {
		return Recommendation{}, ErrTargetMismatch
	}
	taxonomy := FrozenTaxonomy()
	profile, exists := taxonomy.InspectionProfiles[request.InspectionProfileCode]
	if !exists || !contains(profile.AllowedTargetKinds, request.CanonicalTargetKind) || !contains(profile.AllowedTargetProfileCodes, request.TargetProfileCode) || !contains(profile.AllowedInspectionTypeCodes, request.InspectionTypeCode) {
		return Recommendation{}, ErrTargetMismatch
	}
	if !qualifierKeysExact(request.OperationQualifiers, profile.RequiredOperationQualifierKeys) || !qualifierKeysExact(request.ActivityQualifiers, profile.RequiredActivityQualifierKeys) {
		return Recommendation{}, ErrQualifierMismatch
	}
	normalizedOperationQualifiers, err := normalizeQualifiers(request.OperationQualifiers, taxonomy.OperationQualifierValues, "operationQualifiers")
	if err != nil {
		return Recommendation{}, ErrQualifierMismatch
	}
	normalizedActivityQualifiers, err := normalizeQualifiers(request.ActivityQualifiers, taxonomy.ActivityQualifierValues, "activityQualifiers")
	if err != nil {
		return Recommendation{}, ErrQualifierMismatch
	}

	recommendation := Recommendation{
		OperationID: request.OperationID, IdempotencyKey: request.IdempotencyKey,
		GenerationID: draft.GenerationID, DraftID: draft.DraftID, DraftRevision: draft.Revision, DraftContentDigest: draft.ContentDigest,
		TaxonomyVersion: draft.TaxonomyVersion, TaxonomyDigest: draft.TaxonomyDigest,
		ClassificationRunID: draft.ClassificationRunID, ClassificationRunDigest: draft.ClassificationRunDigest,
		AggregateDigest: draft.AggregateDigest, OrganizationID: request.OrganizationID,
		ProviderScopeRootID: scope.ProviderScopeRootID, ProviderScopeID: scope.ProviderScopeID,
		ProviderScopeVersion: scope.ProviderScopeVersion, ProviderScopeProfileDigest: scope.ProfileDigest,
		ProviderTypeID: scope.ProviderTypeID, ProviderTypeCode: scope.ProviderTypeCode,
		DepartmentID: scope.DepartmentID, OrganizationalUnitID: scope.OrganizationalUnitID,
		TargetID: request.TargetID, CanonicalTargetKind: request.CanonicalTargetKind, TargetProfileCode: request.TargetProfileCode,
		InspectionProfileCode: request.InspectionProfileCode,
		InspectionTypeCode:    request.InspectionTypeCode, ReadinessEventID: readiness.ReadinessEventID,
		OperationQualifiers: normalizedOperationQualifiers, ActivityQualifiers: normalizedActivityQualifiers,
		EffectiveAt: request.EffectiveAt.UTC(), ReadinessEventDigest: readiness.ReadinessEventDigest, Items: []RecommendationItem{},
	}
	for _, item := range currentDraftItems(draft) {
		if item.Disposition == nil || *item.Disposition != DispositionInclude {
			continue
		}
		projection := item.CurrentProjection
		if projection.CanonicalTargetKind != request.CanonicalTargetKind || projection.TargetProfileCode != request.TargetProfileCode ||
			!contains(projection.InspectionProfileCodes, request.InspectionProfileCode) ||
			!contains(projection.InspectionTypeCodes, request.InspectionTypeCode) ||
			!qualifiersEqual(projection.OperationQualifiers, request.OperationQualifiers) ||
			!qualifiersEqual(projection.ActivityQualifiers, request.ActivityQualifiers) {
			continue
		}
		if !recommendationApplicabilityEligible(projection.ApplicabilityDisposition) {
			continue
		}
		recommendation.Items = append(recommendation.Items, RecommendationItem{
			QuestionRef: cloneQuestionRef(item.QuestionRef), RootSequence: item.QuestionRef.RootSequence,
			Current: true, DraftDisposition: *item.Disposition, Projection: cloneJSON(projection),
		})
	}
	sort.Slice(recommendation.Items, func(i, j int) bool {
		if recommendation.Items[i].RootSequence == recommendation.Items[j].RootSequence {
			return recommendation.Items[i].QuestionRef.Key() < recommendation.Items[j].QuestionRef.Key()
		}
		return recommendation.Items[i].RootSequence < recommendation.Items[j].RootSequence
	})
	if len(recommendation.Items) == 0 {
		return Recommendation{}, ErrNoEligibleRecommendation
	}
	recommendation.Digest = digestExcludingJSONFields("AGA-DETERMINISTIC-RECOMMENDATION-V1", recommendation, "digest")
	return recommendation, nil
}

func recommendationApplicabilityEligible(disposition string) bool {
	return contains([]string{
		"APPLICABLE", "CONDITIONAL_ON_CONFIGURATION", "CONDITIONAL_ON_FACILITY", "CONDITIONAL_ON_OPERATION",
	}, disposition)
}

func readinessReason(draft Draft) string {
	for _, event := range draft.ReadinessEvents {
		if event.ReadinessEventID == draft.CurrentReadinessEventID {
			return event.ReasonCode
		}
	}
	return ""
}

func resolveReadinessEvent(draft Draft, eventID string) (ReadinessEvent, bool) {
	if eventID == "" || eventID != draft.CurrentReadinessEventID {
		return ReadinessEvent{}, false
	}
	var result ReadinessEvent
	found := false
	for _, event := range draft.ReadinessEvents {
		if event.ReadinessEventID == eventID {
			if found {
				return ReadinessEvent{}, false
			}
			result, found = event, true
		}
	}
	return result, found
}

func qualifiersEqual(left, right []Qualifier) bool {
	taxonomy := FrozenTaxonomy()
	leftOperation, leftOperationErr := normalizeQualifiers(left, taxonomy.OperationQualifierValues, "qualifiers")
	rightOperation, rightOperationErr := normalizeQualifiers(right, taxonomy.OperationQualifierValues, "qualifiers")
	if leftOperationErr == nil && rightOperationErr == nil {
		return reflect.DeepEqual(leftOperation, rightOperation)
	}
	leftActivity, leftActivityErr := normalizeQualifiers(left, taxonomy.ActivityQualifierValues, "qualifiers")
	rightActivity, rightActivityErr := normalizeQualifiers(right, taxonomy.ActivityQualifierValues, "qualifiers")
	if leftActivityErr != nil || rightActivityErr != nil {
		return false
	}
	return reflect.DeepEqual(leftActivity, rightActivity)
}

func qualifierKeysExact(qualifiers []Qualifier, required []string) bool {
	keys := make([]string, len(qualifiers))
	for index, qualifier := range qualifiers {
		keys[index] = qualifier.Key
	}
	sort.Strings(keys)
	want := append([]string{}, required...)
	sort.Strings(want)
	return reflect.DeepEqual(keys, want)
}
