package agaapplicability

import (
	"errors"
	"testing"
	"time"
)

func TestRecommendRequiresCurrentIncludedLeaf(t *testing.T) {
	draft := draftFixture(t, 1310)
	selectionStart := FrozenExternalUnresolvedCount
	baseToReword := currentDraftItems(draft)[selectionStart]
	supersededBaseKey := baseToReword.QuestionRef.Key()
	reworded, err := applyDraftTestCommand(draft, DraftCommand{
		Action: DraftRewordCandidate, TargetQuestionKey: baseToReword.QuestionRef.Key(),
		ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
		ReasonCode: "SYNTHETIC_CANDIDATE_REWORDED", ActorSubjectID: "manager-1",
		WorkspaceBody:       "Synthetic superseding wording for recommendation selection.",
		WorkspaceBodyDigest: ComputeWorkspaceBodyDigest("Synthetic superseding wording for recommendation selection."),
		CreatedAt:           time.Date(2026, 8, 3, 14, 0, 0, 0, time.UTC),
	}, NewSequentialIDAllocator("recommendation-reword"))
	if err != nil {
		t.Fatalf("reword before recommendation error = %v", err)
	}
	rewordedLeaf := currentDraftItems(reworded)[selectionStart]
	projection := completeProjection()
	draft, err = applyDraftTestCommand(reworded, DraftCommand{
		Action: DraftResolveClassificationProposals, TargetQuestionKey: rewordedLeaf.QuestionRef.Key(),
		ExpectedRevision: reworded.Revision, ExpectedContentDigest: reworded.ContentDigest,
		ReasonCode: "MANAGER_EXACT_RESOLUTION", ResolutionMode: ResolutionSetExact, ExactProjection: &projection,
	}, NewSequentialIDAllocator("recommendation-resolution"))
	if err != nil {
		t.Fatalf("reword resolution error = %v", err)
	}
	current := currentDraftItems(draft)
	include := DispositionInclude
	exclude := DispositionExclude
	deferDisposition := DispositionDefer
	for index := range current {
		current[index].Disposition = &exclude
		current[index].ReviewState = ReviewManagerDisposed
	}
	current[selectionStart].Disposition = &include
	current[selectionStart+1].Disposition = &include
	current[selectionStart+2].Disposition = &exclude
	current[selectionStart+3].Disposition = &deferDisposition
	current[selectionStart+4].Disposition = &exclude
	current[selectionStart+5].CurrentProjection.InspectionTypeCodes = []string{"FOLLOW_UP"}
	setExactProposalResolution(&current[selectionStart+5])
	demoteSemanticEdit(&current[selectionStart+5])
	current[selectionStart+5].Disposition = &include
	current[selectionStart+5].ReviewState = ReviewManagerDisposed
	current[selectionStart+6].CurrentProjection.ApplicabilityDisposition = "NOT_APPLICABLE_WITH_REASON"
	setExactProposalResolution(&current[selectionStart+6])
	demoteSemanticEdit(&current[selectionStart+6])
	current[selectionStart+6].Disposition = &include
	current[selectionStart+6].ReviewState = ReviewManagerDisposed
	current[selectionStart+7].CurrentProjection.ApplicabilityDisposition = "REQUIRES_EXPERT_DETERMINATION"
	setExactProposalResolution(&current[selectionStart+7])
	demoteSemanticEdit(&current[selectionStart+7])
	current[selectionStart+7].Disposition = &include
	current[selectionStart+7].ReviewState = ReviewManagerDisposed
	for _, item := range current {
		replaceCurrentDraftItem(&draft, item)
	}
	draft.ContentDigest = ComputeDraftContentDigest(draft)
	effectiveAt := time.Date(2026, 8, 3, 15, 0, 0, 0, time.UTC)
	ready, err := applyDraftTestCommand(draft, DraftCommand{
		Action: DraftMarkReady, ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
		ReasonCode: "SIMULATION_SOURCE_GAP_OVERRIDE", ActorSubjectID: "manager-1", CreatedAt: effectiveAt.Add(-time.Minute),
		ReadinessEventID: "aga-ws-readiness-event-0001", ProviderScopeProfileDigest: digestHex("scope-profile"),
	}, NewSequentialIDAllocator("ready"))
	if err != nil {
		t.Fatalf("mark ready error = %v", err)
	}
	draft = ready
	readiness, ok := resolveReadinessEvent(draft, draft.CurrentReadinessEventID)
	if !ok {
		t.Fatal("readiness event missing")
	}

	scope := ProviderScopeFact{
		GenerationID: draft.GenerationID, ProfileDigest: digestHex("scope-profile"), OrganizationID: "org-aga", ProviderScopeRootID: "scope-root",
		ProviderScopeID: "scope-v2", ProviderScopeVersion: 2, ProviderTypeID: "provider-type-aerodrome-operator",
		ProviderTypeCode: "AERODROME_OPERATOR", Status: "ACTIVE", EffectiveFrom: effectiveAt.Add(-time.Hour), DepartmentID: "AGA",
		OrganizationalUnitID: "AERODROME_INSPECTORATE",
		Targets:              []TypedTarget{{ID: "target-1", Kind: "SYSTEM", ProfileCode: "AERODROME_MANAGEMENT_SYSTEM"}},
		OperationQualifiers:  []Qualifier{{Key: "OPERATION_STATUS", Value: "ACTIVE"}},
		ActivityQualifiers:   []Qualifier{{Key: "ACTIVITY_TYPE", Value: "MAINTENANCE"}},
	}
	scope.ProfileDigest = ComputeProviderScopeProfileDigest(scope)
	draft.ReadinessEvents[0].ProviderScopeProfileDigest = scope.ProfileDigest
	draft.ReadinessEvents[0].ReadinessEventDigest = digestExcludingJSONFields("AGA-DEMO-READINESS-EVENT-V1", draft.ReadinessEvents[0], "readinessEventDigest")
	readiness = draft.ReadinessEvents[0]
	request := RecommendationRequest{
		OperationID: "create-recommendation-1", IdempotencyKey: "idem-recommendation-1",
		ExpectedGenerationID: draft.GenerationID, OrganizationID: scope.OrganizationID,
		ProviderScopeRootID: scope.ProviderScopeRootID, ProviderScopeID: scope.ProviderScopeID,
		ProviderScopeVersion: scope.ProviderScopeVersion, ProviderTypeID: scope.ProviderTypeID,
		DepartmentID: scope.DepartmentID, OrganizationalUnitID: scope.OrganizationalUnitID,
		TargetID: "target-1", CanonicalTargetKind: "SYSTEM", TargetProfileCode: "AERODROME_MANAGEMENT_SYSTEM",
		InspectionProfileCode: "AERODROME_MANAGEMENT_SYSTEM", InspectionTypeCode: "PERIODIC_SURVEILLANCE",
		OperationQualifiers: []Qualifier{{Key: "OPERATION_STATUS", Value: "ACTIVE"}},
		ActivityQualifiers:  []Qualifier{{Key: "ACTIVITY_TYPE", Value: "MAINTENANCE"}}, EffectiveAt: effectiveAt,
		TaxonomyVersion: draft.TaxonomyVersion, TaxonomyDigest: draft.TaxonomyDigest,
		ClassificationRunID: draft.ClassificationRunID, ClassificationRunDigest: draft.ClassificationRunDigest,
		DraftID: draft.DraftID, DraftRevision: draft.Revision, DraftContentDigest: draft.ContentDigest,
		ExpectedDraftRevision: draft.Revision, ReadinessEventID: readiness.ReadinessEventID,
		ReadinessEventDigest: readiness.ReadinessEventDigest,
	}
	recommendation, err := BuildRecommendation(draft, scope, request)
	if err != nil {
		t.Fatalf("BuildRecommendation() error = %v", err)
	}
	if len(recommendation.Items) != 2 {
		t.Fatalf("recommended item count = %d, want 2", len(recommendation.Items))
	}
	if recommendation.Items[0].RootSequence >= recommendation.Items[1].RootSequence {
		t.Fatal("recommendation order invariant failed")
	}
	for _, item := range recommendation.Items {
		if !item.Current || item.DraftDisposition != DispositionInclude {
			t.Fatal("recommended item state invariant failed")
		}
		if item.QuestionRef.Key() == supersededBaseKey {
			t.Fatal("superseded base leaf entered recommendation")
		}
	}
	if recommendation.Items[0].QuestionRef.Workspace == nil {
		t.Fatal("current reworded workspace leaf was not selected")
	}
	if recommendation.OperationID != request.OperationID || recommendation.IdempotencyKey != request.IdempotencyKey || recommendation.ProviderScopeRootID != scope.ProviderScopeRootID || recommendation.ProviderScopeVersion != scope.ProviderScopeVersion || recommendation.ProviderScopeProfileDigest != scope.ProfileDigest || recommendation.ProviderTypeCode != scope.ProviderTypeCode || recommendation.ProviderTypeID != scope.ProviderTypeID || recommendation.DraftID != draft.DraftID || recommendation.DraftRevision != draft.Revision || recommendation.TaxonomyDigest != draft.TaxonomyDigest || recommendation.CanonicalTargetKind != request.CanonicalTargetKind || recommendation.TargetProfileCode != request.TargetProfileCode || recommendation.DepartmentID != scope.DepartmentID || recommendation.OrganizationalUnitID != scope.OrganizationalUnitID || recommendation.ReadinessEventDigest != readiness.ReadinessEventDigest || recommendation.Digest == "" {
		t.Fatal("recommendation pin invariant failed")
	}

	cases := []struct {
		name          string
		mutateDraft   func(*Draft, *RecommendationRequest)
		mutateScope   func(*ProviderScopeFact)
		mutateRequest func(*RecommendationRequest)
		want          error
	}{
		{name: "revision mismatch", mutateRequest: func(value *RecommendationRequest) { value.DraftRevision-- }, want: ErrDraftConflict},
		{name: "expected revision mismatch", mutateRequest: func(value *RecommendationRequest) { value.ExpectedDraftRevision-- }, want: ErrDraftConflict},
		{name: "content mismatch", mutateRequest: func(value *RecommendationRequest) { value.DraftContentDigest = digestHex("stale") }, want: ErrDraftConflict},
		{name: "not ready", mutateDraft: func(value *Draft, req *RecommendationRequest) {
			value.State = DraftWorking
			value.ContentDigest = ComputeDraftContentDigest(*value)
			req.DraftContentDigest = value.ContentDigest
		}, want: ErrDraftNotReady},
		{name: "pending in ready", mutateDraft: func(value *Draft, req *RecommendationRequest) {
			item := currentDraftItems(*value)[0]
			item.ReviewState = ReviewPendingManager
			item.Disposition = nil
			replaceCurrentDraftItem(value, item)
			value.ContentDigest = ComputeDraftContentDigest(*value)
			req.DraftContentDigest = value.ContentDigest
		}, want: ErrDraftNotReady},
		{name: "generation mismatch", mutateRequest: func(value *RecommendationRequest) { value.ExpectedGenerationID = "aga-ws-generation-other" }, want: ErrReadinessPinMismatch},
		{name: "run mismatch", mutateRequest: func(value *RecommendationRequest) { value.ClassificationRunDigest = digestHex("other") }, want: ErrReadinessPinMismatch},
		{name: "taxonomy mismatch", mutateRequest: func(value *RecommendationRequest) { value.TaxonomyDigest = digestHex("other") }, want: ErrReadinessPinMismatch},
		{name: "readiness mismatch", mutateRequest: func(value *RecommendationRequest) { value.ReadinessEventDigest = digestHex("other") }, want: ErrReadinessPinMismatch},
		{name: "scope profile mismatch", mutateScope: func(value *ProviderScopeFact) { value.ProfileDigest = digestHex("other-profile") }, want: ErrReadinessPinMismatch},
		{name: "provider mismatch", mutateRequest: func(value *RecommendationRequest) { value.ProviderTypeID = "other" }, want: ErrProviderScopeMismatch},
		{name: "provider code not inspected scope", mutateScope: func(value *ProviderScopeFact) { value.ProviderTypeCode = "ANSP" }, want: ErrProviderScopeMismatch},
		{name: "inactive scope", mutateScope: func(value *ProviderScopeFact) { value.Status = "EXPIRED" }, want: ErrProviderScopeNotApplicable},
		{name: "organization mismatch", mutateRequest: func(value *RecommendationRequest) { value.OrganizationID = "other" }, want: ErrProviderScopeMismatch},
		{name: "target mismatch", mutateRequest: func(value *RecommendationRequest) { value.TargetID = "other" }, want: ErrTargetMismatch},
		{name: "ambiguous target", mutateScope: func(value *ProviderScopeFact) { value.Targets = append(value.Targets, value.Targets[0]) }, want: ErrTargetMismatch},
		{name: "zero eligible leaves", mutateScope: func(value *ProviderScopeFact) {
			value.Targets = append(value.Targets, TypedTarget{ID: "target-organization", Kind: "ORGANIZATION", ProfileCode: "AERODROME_MANAGEMENT_SYSTEM"})
		}, mutateRequest: func(value *RecommendationRequest) {
			value.TargetID = "target-organization"
			value.CanonicalTargetKind = "ORGANIZATION"
		}, want: ErrNoEligibleRecommendation},
		{name: "missing operation qualifier", mutateRequest: func(value *RecommendationRequest) { value.OperationQualifiers = nil }, want: ErrQualifierMismatch},
		{name: "extra activity qualifier", mutateRequest: func(value *RecommendationRequest) {
			value.ActivityQualifiers = append(value.ActivityQualifiers, Qualifier{Key: "ACTIVITY_TYPE", Value: "RISK_ASSESSMENT"})
		}, want: ErrQualifierMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			draftCopy := cloneDraft(draft)
			scopeCopy := cloneJSON(scope)
			requestCopy := cloneJSON(request)
			if tc.mutateDraft != nil {
				tc.mutateDraft(&draftCopy, &requestCopy)
			}
			if tc.mutateScope != nil {
				tc.mutateScope(&scopeCopy)
				if tc.name != "scope profile mismatch" {
					scopeCopy.ProfileDigest = ComputeProviderScopeProfileDigest(scopeCopy)
					draftCopy.ReadinessEvents[0].ProviderScopeProfileDigest = scopeCopy.ProfileDigest
					draftCopy.ReadinessEvents[0].ReadinessEventDigest = digestExcludingJSONFields("AGA-DEMO-READINESS-EVENT-V1", draftCopy.ReadinessEvents[0], "readinessEventDigest")
					requestCopy.ReadinessEventDigest = draftCopy.ReadinessEvents[0].ReadinessEventDigest
				}
			}
			if tc.mutateRequest != nil {
				tc.mutateRequest(&requestCopy)
			}
			if _, err := BuildRecommendation(draftCopy, scopeCopy, requestCopy); !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
}
