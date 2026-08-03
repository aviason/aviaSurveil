package agaapplicability

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func applyDraftTestCommand(draft Draft, command DraftCommand, allocator IDAllocator) (Draft, error) {
	command.OperationID = "test-operation"
	command.IdempotencyKey = "test-idempotency"
	command.ExpectedGenerationID = draft.GenerationID
	return ApplyDraftCommand(draft, command, allocator)
}

func TestDraftCommandsCreateImmutableSuccessors(t *testing.T) {
	reasonCases := []struct {
		name   string
		action DraftAction
		reason string
		want   error
	}{
		{name: "retain without reason", action: DraftRetain},
		{name: "ordinary include without reason", action: DraftInclude},
		{name: "exclude reason", action: DraftExclude, reason: "MANAGER_SCOPE_DECISION"},
		{name: "defer reason", action: DraftDefer, reason: "MANAGER_SCOPE_DECISION"},
		{name: "reclassify reason", action: DraftReclassifyMainDomain, reason: "CLASSIFICATION_EXPERT_REVIEW"},
		{name: "add topic reason", action: DraftAddTopic, reason: "CLASSIFICATION_EXPERT_REVIEW"},
		{name: "remove topic reason", action: DraftRemoveTopic, reason: "CLASSIFICATION_EXPERT_REVIEW"},
		{name: "resolution reason", action: DraftResolveClassificationProposals, reason: "MANAGER_EXACT_RESOLUTION"},
		{name: "add candidate reason", action: DraftAddCandidate, reason: "SYNTHETIC_CANDIDATE_ADDED"},
		{name: "reword reason", action: DraftRewordCandidate, reason: "SYNTHETIC_CANDIDATE_REWORDED"},
		{name: "ready reason", action: DraftMarkReady, reason: "MANAGER_SCOPE_DECISION"},
		{name: "required reason missing", action: DraftAddTopic, want: ErrReasonRequired},
		{name: "known reason on wrong action", action: DraftExclude, reason: "CLASSIFICATION_EXPERT_REVIEW", want: ErrInvalidReason},
		{name: "unknown reason", action: DraftExclude, reason: "MODEL_REASON", want: ErrInvalidReason},
	}
	for _, tc := range reasonCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateDraftReason(tc.action, tc.reason); !errors.Is(err, tc.want) {
				t.Fatalf("validateDraftReason(%s, %q) = %v, want %v", tc.action, tc.reason, err, tc.want)
			}
		})
	}

	draft := draftFixture(t, 3)
	if _, err := ApplyDraftCommand(draft, DraftCommand{
		Action: DraftRetain, TargetQuestionKey: currentDraftItems(draft)[0].QuestionRef.Key(),
		ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
	}, NewSequentialIDAllocator("missing-envelope")); !errors.Is(err, ErrCommandEnvelope) {
		t.Fatalf("missing generic command envelope error = %v", err)
	}
	original := cloneDraft(draft)
	item := currentDraftItems(draft)[0]
	allocator := NewSequentialIDAllocator("test")
	next, err := applyDraftTestCommand(draft, DraftCommand{
		Action: DraftInclude, TargetQuestionKey: item.QuestionRef.Key(),
		ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
		ReasonCode: "MANAGER_SCOPE_DECISION",
	}, allocator)
	if err != nil {
		t.Fatalf("applyDraftTestCommand(INCLUDE) error = %v", err)
	}
	if next.Revision != draft.Revision+1 || next.ContentDigest == draft.ContentDigest {
		t.Fatalf("successor revision/digest = %d/%s", next.Revision, next.ContentDigest)
	}
	if !draftEqual(draft, original) {
		t.Fatal("predecessor Draft mutated")
	}
	if got := currentDraftItems(next)[0].Disposition; got == nil || *got != DispositionInclude {
		t.Fatal("successor disposition invariant failed")
	}
	_, err = applyDraftTestCommand(next, DraftCommand{
		Action: DraftExclude, TargetQuestionKey: item.QuestionRef.Key(),
		ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
		ReasonCode: "MANAGER_SCOPE_DECISION",
	}, allocator)
	if !errors.Is(err, ErrDraftConflict) {
		t.Fatalf("stale CAS error = %v", err)
	}
	_, err = applyDraftTestCommand(next, DraftCommand{
		Action: DraftExclude, TargetQuestionKey: item.QuestionRef.Key(),
		ExpectedRevision: next.Revision, ExpectedContentDigest: next.ContentDigest,
	}, allocator)
	if !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("missing reason error = %v", err)
	}

	sourceGap := draftFixture(t, 1)
	gapItem := currentDraftItems(sourceGap)[0]
	gapItem.QuestionSourceProposalGap = true
	gapItem.sealedGovernance.QuestionSourceProposalGap = true
	gapItem.RecommendationState = RecommendationBlockedSourceGap
	gapItem.sealedRecommendationState = RecommendationBlockedSourceGap
	gapItem.ReviewState = ReviewPendingManager
	gapItem.Disposition = nil
	replaceCurrentDraftItem(&sourceGap, gapItem)
	sourceGap.ContentDigest = ComputeDraftContentDigest(sourceGap)
	if _, err := applyDraftTestCommand(sourceGap, DraftCommand{
		Action: DraftInclude, TargetQuestionKey: gapItem.QuestionRef.Key(),
		ExpectedRevision: sourceGap.Revision, ExpectedContentDigest: sourceGap.ContentDigest,
	}, allocator); !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("source-gap INCLUDE without reason error = %v", err)
	}
	if _, err := applyDraftTestCommand(sourceGap, DraftCommand{
		Action: DraftInclude, TargetQuestionKey: gapItem.QuestionRef.Key(),
		ExpectedRevision: sourceGap.Revision, ExpectedContentDigest: sourceGap.ContentDigest,
		ReasonCode: "MANAGER_SCOPE_DECISION",
	}, allocator); !errors.Is(err, ErrInvalidReason) {
		t.Fatalf("source-gap INCLUDE without override reason error = %v", err)
	}

	notComplete := draftFixture(t, 1)
	if _, err := applyDraftTestCommand(notComplete, DraftCommand{
		Action: DraftMarkReady, ExpectedRevision: notComplete.Revision,
		ExpectedContentDigest: notComplete.ContentDigest, ReasonCode: "MANAGER_SCOPE_DECISION",
		ActorSubjectID: "manager-1", CreatedAt: time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC),
		ReadinessEventID: "aga-ws-readiness-small-0001", ProviderScopeProfileDigest: digestHex("scope-profile"),
	}, allocator); !errors.Is(err, ErrDraftNotReady) {
		t.Fatalf("non-1310 readiness error = %v", err)
	}

	ready := draftFixture(t, 1)
	ready.State = DraftReadyForDemoSimulation
	ready.ContentDigest = ComputeDraftContentDigest(ready)
	readyItem := currentDraftItems(ready)[0]
	edited, err := applyDraftTestCommand(ready, DraftCommand{
		Action: DraftRetain, TargetQuestionKey: readyItem.QuestionRef.Key(),
		ExpectedRevision: ready.Revision, ExpectedContentDigest: ready.ContentDigest,
	}, allocator)
	if err != nil {
		t.Fatalf("ready successor edit error = %v", err)
	}
	if edited.State != DraftWorking {
		t.Fatalf("edited ready Draft state = %q", edited.State)
	}
}

func TestSealedBaseStateBindsImmutableProjectionAndGlobalOrder(t *testing.T) {
	draft := draftFixture(t, 2)
	item := currentDraftItems(draft)[0]
	item.CurrentProjection.MainDomainCode = "SAFETY_MANAGEMENT_RISK_ASSESSMENT"
	draft.Items[0] = item
	draft.ContentDigest = ComputeDraftContentDigest(draft)
	if err := validateDraftQuestionGraph(draft); !errors.Is(err, ErrInvalidResolution) {
		t.Fatalf("altered unedited base projection error = %v", err)
	}
	draft = draftFixture(t, 2)
	draft.Items[0].QuestionRef.RootSequence, draft.Items[1].QuestionRef.RootSequence = 2, 1
	draft.ContentDigest = ComputeDraftContentDigest(draft)
	if err := validateDraftQuestionGraph(draft); !errors.Is(err, ErrInvalidResolution) {
		t.Fatalf("base global order mutation error = %v", err)
	}
}

func TestDraftBatchRejectsInactiveResetSupersededAndUnsealed(t *testing.T) {
	draft := draftFixture(t, 2)
	now := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	for _, mutate := range []func(*Draft){
		func(value *Draft) { value.GenerationState = "RESET" },
		func(value *Draft) { value.State = "SUPERSEDED" },
		func(value *Draft) { value.ClassificationRunState = "LOADING" },
	} {
		copy := cloneDraft(draft)
		mutate(&copy)
		copy.ContentDigest = ComputeDraftContentDigest(copy)
		if _, err := PreviewDraftBatch(copy, DraftBatchFilter{}, DraftBatchAction{Action: DraftRetain}, now, NewSequentialIDAllocator("blocked")); !errors.Is(err, ErrDraftNotReady) {
			t.Fatalf("blocked preview error = %v", err)
		}
	}
}

func TestDraftSemanticEditDemotesAutoPreselection(t *testing.T) {
	draft := draftFixture(t, 1)
	item := currentDraftItems(draft)[0]
	high := ConfidenceHigh
	item.DraftAgreementConfidence = &high
	item.SealedAgreementConfidence = &high
	item.RecommendationState = RecommendationAutoProposed
	item.ReviewState = ReviewAutoPreselected
	include := DispositionInclude
	item.Disposition = &include
	replaceCurrentDraftItem(&draft, item)
	draft.ContentDigest = ComputeDraftContentDigest(draft)
	next, err := applyDraftTestCommand(draft, DraftCommand{
		Action: DraftReclassifyMainDomain, TargetQuestionKey: item.QuestionRef.Key(),
		ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
		ReasonCode: "CLASSIFICATION_EXPERT_REVIEW", MainDomainCode: "SAFETY_MANAGEMENT_RISK_ASSESSMENT",
	}, NewSequentialIDAllocator("test"))
	if err != nil {
		t.Fatalf("semantic edit error = %v", err)
	}
	got := currentDraftItems(next)[0]
	if got.SealedAgreementConfidence == nil || *got.SealedAgreementConfidence != ConfidenceHigh || got.DraftAgreementConfidence != nil {
		t.Fatal("sealed/draft confidence invariant failed")
	}
	if got.ReviewState != ReviewPendingManager || got.Disposition != nil || got.RecommendationState != RecommendationManagerReview {
		t.Fatal("semantic edit governance invariant failed")
	}
	if got.CurrentProjection.MainDomainCode != "SAFETY_MANAGEMENT_RISK_ASSESSMENT" {
		t.Fatalf("main domain = %q", got.CurrentProjection.MainDomainCode)
	}

	oneTopic := draftFixture(t, 1)
	oneTopicItem := currentDraftItems(oneTopic)[0]
	oneTopic, err = applyDraftTestCommand(oneTopic, DraftCommand{
		Action: DraftRemoveTopic, TargetQuestionKey: oneTopicItem.QuestionRef.Key(),
		ExpectedRevision: oneTopic.Revision, ExpectedContentDigest: oneTopic.ContentDigest,
		ReasonCode: "CLASSIFICATION_EXPERT_REVIEW", TopicCode: "STAFFING_AND_COMPETENCE",
	}, NewSequentialIDAllocator("remove-first-topic"))
	if err != nil {
		t.Fatalf("remove first optional topic error = %v", err)
	}
	oneTopicItem = currentDraftItems(oneTopic)[0]
	removed, err := applyDraftTestCommand(oneTopic, DraftCommand{
		Action: DraftRemoveTopic, TargetQuestionKey: oneTopicItem.QuestionRef.Key(),
		ExpectedRevision: oneTopic.Revision, ExpectedContentDigest: oneTopic.ContentDigest,
		ReasonCode: "CLASSIFICATION_EXPERT_REVIEW", TopicCode: "QUALITY_MANAGEMENT_SYSTEM",
	}, NewSequentialIDAllocator("remove-final-topic"))
	if err != nil {
		t.Fatalf("remove optional final topic error = %v", err)
	}
	if topics := currentDraftItems(removed)[0].CurrentProjection.TopicCodes; len(topics) != 0 || topics == nil {
		t.Fatal("optional topic removal invariant failed")
	}
}

func TestDraftResolvesEveryProposalFamily(t *testing.T) {
	candidate := completeProjection()
	exact := exactProjection()
	cases := []struct {
		name  string
		mode  ResolutionMode
		exact *ProposalProjection
		want  ProposalProjection
	}{
		{name: "candidate", mode: ResolutionCandidate, want: candidate},
		{name: "challenge", mode: ResolutionChallenge, want: candidate},
		{name: "exact", mode: ResolutionSetExact, exact: &exact, want: exact},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			draft := draftFixture(t, 1)
			item := currentDraftItems(draft)[0]
			item.CurrentProjection = candidate
			replaceCurrentDraftItem(&draft, item)
			draft.ContentDigest = ComputeDraftContentDigest(draft)
			if err := validateDraftQuestionGraph(draft); err != nil {
				t.Fatalf("pre-resolution graph error = %v", err)
			}
			resolvedProbe, resolutionProbe, err := resolveProjection(FrozenTaxonomy(), draft, item, DraftCommand{ResolutionMode: tc.mode, ExactProjection: tc.exact})
			if err != nil {
				t.Fatalf("direct resolution error = %v", err)
			}
			probe := cloneDraftItem(item)
			probe.CurrentProjection = resolvedProbe
			probe.ProposalResolution = resolutionProbe
			demoteSemanticEdit(&probe)
			if tc.mode == ResolutionChallenge {
				if !validStoredPass(probe.challengePassRecord, probe.challengePassResultDigest, draft.ClassificationRunID, probe.QuestionRef, PassChallenge) {
					t.Fatal("resolved challenge pass pin invalid")
				}
				if !ProjectionFieldSetsEqual(probe.CurrentProjection, probe.challengePassRecord.ProposalProjection) {
					t.Fatal("resolved challenge projection mismatch")
				}
			}
			if err := validateDraftItemState(draft, probe); err != nil {
				t.Fatalf("resolved item state error = %v", err)
			}
			next, err := applyDraftTestCommand(draft, DraftCommand{
				Action: DraftResolveClassificationProposals, TargetQuestionKey: item.QuestionRef.Key(),
				ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
				ReasonCode: "MANAGER_EXACT_RESOLUTION", ResolutionMode: tc.mode, ExactProjection: tc.exact,
			}, NewSequentialIDAllocator("test"))
			if err != nil {
				t.Fatalf("resolve proposals error = %v", err)
			}
			got := currentDraftItems(next)[0]
			for _, field := range FrozenTaxonomy().ProposalFields {
				if !ProjectionFieldEqual(got.CurrentProjection, tc.want, field) {
					t.Fatalf("field %s did not use the complete %s projection", field, tc.mode)
				}
			}
			if got.ReviewState != ReviewPendingManager || got.Disposition != nil {
				t.Fatal("resolution disposition invariant failed")
			}
		})
	}
	for _, tc := range []struct {
		name string
		mode ResolutionMode
	}{
		{name: "candidate", mode: ResolutionCandidate},
		{name: "challenge", mode: ResolutionChallenge},
	} {
		t.Run("rejects replaced "+tc.name+" pass and local digest", func(t *testing.T) {
			draft := draftFixture(t, 1)
			item := currentDraftItems(draft)[0]
			if tc.mode == ResolutionCandidate {
				replaced := cloneJSON(*item.candidatePassRecord)
				replaced.ProposalProjection = exactProjection()
				replaced.PassResultDigest = ComputePassResultDigest(replaced)
				item.candidatePassRecord = &replaced
				item.candidatePassResultDigest = replaced.PassResultDigest
			} else {
				replaced := cloneJSON(*item.challengePassRecord)
				replaced.ProposalProjection = exactProjection()
				replaced.PassResultDigest = ComputePassResultDigest(replaced)
				item.challengePassRecord = &replaced
				item.challengePassResultDigest = replaced.PassResultDigest
			}
			replaceCurrentDraftItem(&draft, item)
			draft.ContentDigest = ComputeDraftContentDigest(draft)
			if _, err := applyDraftTestCommand(draft, DraftCommand{
				Action: DraftResolveClassificationProposals, TargetQuestionKey: item.QuestionRef.Key(),
				ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
				ReasonCode: "MANAGER_EXACT_RESOLUTION", ResolutionMode: tc.mode,
			}, NewSequentialIDAllocator("replaced-pass")); !errors.Is(err, ErrPassBijection) {
				t.Fatalf("replaced sealed pass error = %v", err)
			}
		})
	}
	draft := draftFixture(t, 1)
	item := currentDraftItems(draft)[0]
	if _, err := applyDraftTestCommand(draft, DraftCommand{
		Action: DraftResolveClassificationProposals, TargetQuestionKey: item.QuestionRef.Key(),
		ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
		ReasonCode: "MANAGER_EXACT_RESOLUTION", ResolutionMode: ResolutionCandidate, ExactProjection: &exact,
	}, NewSequentialIDAllocator("mixed")); !errors.Is(err, ErrInvalidResolution) {
		t.Fatalf("mixed resolution payload error = %v", err)
	}
	if _, err := applyDraftTestCommand(draft, DraftCommand{
		Action: DraftResolveClassificationProposals, TargetQuestionKey: item.QuestionRef.Key(),
		ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
		ReasonCode: "MANAGER_EXACT_RESOLUTION", ResolutionMode: ResolutionCandidate,
		TopicCode: "QUALITY_MANAGEMENT_SYSTEM",
	}, NewSequentialIDAllocator("mixed-field")); !errors.Is(err, ErrInvalidResolution) {
		t.Fatalf("irrelevant mixed resolution field error = %v", err)
	}
}

func TestQuestionReferenceUnionIsClosed(t *testing.T) {
	if _, err := parseCanonicalUTCTimestamp("2026-08-03T10:00:00.0Z"); err != nil {
		t.Fatalf("one-digit UTC fraction error = %v", err)
	}
	base := baseIdentity(1, digestHex("base"))
	baseRef := BaseQuestionReference(base)
	if err := ValidateQuestionRef(baseRef); err != nil {
		t.Fatalf("base ref error = %v", err)
	}
	workspace := WorkspaceQuestionRef{
		GenerationID: "aga-ws-generation-0001", RootID: "aga-ws-root-manual-0001", VersionID: "aga-ws-version-manual-0001",
		ProposalID: "aga-ws-proposal-manual-0001", RootSequence: 1311, BodyDigest: ComputeWorkspaceBodyDigest("body"),
		ParentQuestionKey: &ParentQuestionKey{Base: &base}, ActorSubjectID: "actor-1",
		CreatedAt: time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC), ReasonCode: "SYNTHETIC_CANDIDATE_REWORDED",
	}
	workspaceRef := WorkspaceQuestionReference(workspace)
	if err := ValidateQuestionRef(workspaceRef); err != nil {
		t.Fatalf("workspace ref error = %v", err)
	}
	encoded, err := json.Marshal(workspaceRef)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"workspace"`) || strings.Contains(string(encoded), `"base"`) || !strings.Contains(string(encoded), `"questionRootId"`) || !strings.Contains(string(encoded), `"createdBySubjectId"`) {
		t.Fatal("workspace questionRef flat union invariant failed")
	}
	var roundTrip QuestionRef
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatalf("workspace questionRef round trip error = %v", err)
	}
	if roundTrip.Key() != workspaceRef.Key() || roundTrip.Workspace == nil || !roundTrip.Workspace.CreatedAt.Equal(workspace.CreatedAt) || roundTrip.Workspace.ParentQuestionKey == nil || roundTrip.Workspace.ParentQuestionKey.Base == nil {
		t.Fatal("workspace questionRef did not round trip exactly")
	}
	baseEncoded, err := json.Marshal(baseRef)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(baseEncoded, &roundTrip); err != nil || roundTrip.Key() != baseRef.Key() {
		t.Fatalf("base questionRef round trip error = %v", err)
	}
	withExtra := strings.TrimSuffix(string(encoded), "}") + `,"unexpected":true}`
	if err := json.Unmarshal([]byte(withExtra), &roundTrip); !errors.Is(err, ErrQuestionReferenceUnion) {
		t.Fatalf("questionRef extra-property error = %v", err)
	}
	nonCanonicalUTC := strings.Replace(string(encoded), `"createdAt":"2026-08-03T10:00:00Z"`, `"createdAt":"2026-08-03T10:00:00+00:00"`, 1)
	if err := json.Unmarshal([]byte(nonCanonicalUTC), &roundTrip); !errors.Is(err, ErrQuestionReferenceUnion) {
		t.Fatalf("questionRef non-canonical timestamp error = %v", err)
	}
	if err := ValidateQuestionRef(QuestionRef{Origin: QuestionOriginBase}); !errors.Is(err, ErrQuestionReferenceUnion) {
		t.Fatalf("missing base union error = %v", err)
	}
	if err := ValidateQuestionRef(QuestionRef{Origin: QuestionOriginBase, Base: &base, Workspace: &workspace}); !errors.Is(err, ErrQuestionReferenceUnion) {
		t.Fatalf("dual union error = %v", err)
	}
	workspace.ParentQuestionKey = nil
	if err := ValidateQuestionRef(WorkspaceQuestionReference(workspace)); !errors.Is(err, ErrParentQuestionKey) {
		t.Fatalf("reword without parent error = %v", err)
	}
	workspace.ParentQuestionKey = &ParentQuestionKey{Base: &base}
	workspace.ReasonCode = "SYNTHETIC_CANDIDATE_ADDED"
	if err := ValidateQuestionRef(WorkspaceQuestionReference(workspace)); !errors.Is(err, ErrParentQuestionKey) {
		t.Fatalf("add with parent error = %v", err)
	}

	draftItem := currentDraftItems(draftFixture(t, 1))[0]
	draftItemEncoded, err := json.Marshal(draftItem)
	if err != nil {
		t.Fatal(err)
	}
	var draftItemObject map[string]json.RawMessage
	if err := json.Unmarshal(draftItemEncoded, &draftItemObject); err != nil {
		t.Fatal(err)
	}
	wantDraftItemKeys := []string{"currentLeaf", "draftAgreementConfidence", "draftDisposition", "draftItemOrigin", "draftRecommendationState", "draftReviewState", "proposalProjection", "proposalResolution", "questionRef", "questionSourceProposalGap"}
	gotDraftItemKeys := make([]string, 0, len(draftItemObject))
	for key := range draftItemObject {
		gotDraftItemKeys = append(gotDraftItemKeys, key)
	}
	slices.Sort(gotDraftItemKeys)
	if !slices.Equal(gotDraftItemKeys, wantDraftItemKeys) {
		t.Fatalf("draft item schema keys = %v", gotDraftItemKeys)
	}
	if strings.Contains(string(draftItemEncoded), "passRecord") || strings.Contains(string(draftItemEncoded), "workspaceBody") || strings.Contains(string(draftItemEncoded), "sealedAgreementConfidence") {
		t.Fatal("draft item serialized private provenance or wording")
	}
}

func TestAddAllocatesFreshWorkspaceRootVersionAndProposal(t *testing.T) {
	draft := draftFixture(t, 2)
	allocator := NewSequentialIDAllocator("add")
	next, err := applyDraftTestCommand(draft, DraftCommand{
		Action: DraftAddCandidate, ExpectedRevision: draft.Revision,
		ExpectedContentDigest: draft.ContentDigest, ReasonCode: "SYNTHETIC_CANDIDATE_ADDED",
		ActorSubjectID: "manager-1", WorkspaceBody: "Synthetic candidate wording.",
		WorkspaceBodyDigest: ComputeWorkspaceBodyDigest("Synthetic candidate wording."), ExactProjection: ptrProjection(completeProjection()),
		CreatedAt: time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC),
	}, allocator)
	if err != nil {
		t.Fatalf("add candidate error = %v", err)
	}
	if next.BaseQuestionCount != 2 || len(currentDraftItems(next)) != 3 {
		t.Fatalf("base/current counts = %d/%d", next.BaseQuestionCount, len(currentDraftItems(next)))
	}
	added := currentDraftItems(next)[2]
	if added.QuestionRef.Workspace == nil || added.QuestionRef.Workspace.RootID == "" || added.QuestionRef.Workspace.VersionID == "" || added.QuestionRef.Workspace.ProposalID == "" {
		t.Fatal("server IDs missing")
	}
	if added.QuestionRef.Workspace.ParentQuestionKey != nil || added.QuestionRef.Workspace.RootSequence != 3 {
		t.Fatal("add parent/sequence invariant failed")
	}
	if !added.QuestionSourceProposalGap || added.SourceMappingState != SourceMappingRequired || added.SourceAuthorityState != SourceAuthorityNotAttested || added.RiskClassificationState != RiskExpertReviewRequired {
		t.Fatal("new wording governance invariant failed")
	}
	if _, err := applyDraftTestCommand(draft, DraftCommand{
		Action: DraftAddCandidate, ExpectedRevision: draft.Revision,
		ExpectedContentDigest: draft.ContentDigest, ReasonCode: "SYNTHETIC_CANDIDATE_ADDED",
		ActorSubjectID: "manager-1", WorkspaceBody: "Synthetic candidate wording.",
		WorkspaceBodyDigest: digestHex("not-the-body"), ExactProjection: ptrProjection(completeProjection()),
		CreatedAt: time.Date(2026, 8, 3, 11, 0, 0, 0, time.UTC),
	}, NewSequentialIDAllocator("bad-body")); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("workspace body digest mismatch error = %v", err)
	}
}

func TestDraftRewordReplacesCurrentLeaf(t *testing.T) {
	draft := draftFixture(t, 1)
	base := currentDraftItems(draft)[0]
	allocator := NewSequentialIDAllocator("reword")
	first, err := applyDraftTestCommand(draft, DraftCommand{
		Action: DraftRewordCandidate, TargetQuestionKey: base.QuestionRef.Key(),
		ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
		ReasonCode: "SYNTHETIC_CANDIDATE_REWORDED", ActorSubjectID: "manager-1",
		WorkspaceBody: "First synthetic reword.", WorkspaceBodyDigest: ComputeWorkspaceBodyDigest("First synthetic reword."),
		CreatedAt: time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC),
	}, allocator)
	if err != nil {
		t.Fatalf("base reword error = %v", err)
	}
	leaf := currentDraftItems(first)[0]
	if leaf.QuestionRef.Workspace == nil || leaf.QuestionRef.Workspace.ParentQuestionKey == nil || leaf.QuestionRef.Workspace.ParentQuestionKey.Base == nil {
		t.Fatal("base parent key invariant failed")
	}
	if _, err := applyDraftTestCommand(first, DraftCommand{
		Action: DraftInclude, TargetQuestionKey: leaf.QuestionRef.Key(),
		ExpectedRevision: first.Revision, ExpectedContentDigest: first.ContentDigest,
		ReasonCode: "SIMULATION_SOURCE_GAP_OVERRIDE",
	}, NewSequentialIDAllocator("unresolved-reword")); !errors.Is(err, ErrInvalidResolution) {
		t.Fatalf("unresolved reword inclusion error = %v", err)
	}
	rootID := leaf.QuestionRef.Workspace.RootID
	versionID := leaf.QuestionRef.Workspace.VersionID
	second, err := applyDraftTestCommand(first, DraftCommand{
		Action: DraftRewordCandidate, TargetQuestionKey: leaf.QuestionRef.Key(),
		ExpectedRevision: first.Revision, ExpectedContentDigest: first.ContentDigest,
		ReasonCode: "SYNTHETIC_CANDIDATE_REWORDED", ActorSubjectID: "manager-1",
		WorkspaceBody: "Second synthetic reword.", WorkspaceBodyDigest: ComputeWorkspaceBodyDigest("Second synthetic reword."),
		CreatedAt: time.Date(2026, 8, 3, 13, 0, 0, 0, time.UTC),
	}, allocator)
	if err != nil {
		t.Fatalf("workspace reword error = %v", err)
	}
	leaf2 := currentDraftItems(second)[0]
	if leaf2.QuestionRef.Workspace.RootID != rootID || leaf2.QuestionRef.Workspace.VersionID == versionID {
		t.Fatal("root/version successor invariant failed")
	}
	if leaf2.QuestionRef.Workspace.ParentQuestionKey == nil || leaf2.QuestionRef.Workspace.ParentQuestionKey.WorkspaceVersionID != versionID {
		t.Fatal("workspace parent invariant failed")
	}
	_, err = applyDraftTestCommand(second, DraftCommand{
		Action: DraftRewordCandidate, TargetQuestionKey: leaf2.QuestionRef.Key(),
		ExpectedRevision: second.Revision, ExpectedContentDigest: second.ContentDigest,
		ReasonCode: "SYNTHETIC_CANDIDATE_REWORDED", ActorSubjectID: "manager-1",
		WorkspaceBody: "Second synthetic reword.", WorkspaceBodyDigest: leaf2.QuestionRef.Workspace.BodyDigest,
		CreatedAt: time.Date(2026, 8, 3, 14, 0, 0, 0, time.UTC),
	}, allocator)
	if !errors.Is(err, ErrByteIdenticalReword) {
		t.Fatalf("byte-identical reword error = %v", err)
	}
}

func TestWorkspaceQuestionIdentityRejectsAliases(t *testing.T) {
	draft := draftFixture(t, 1)
	allocator := NewSequentialIDAllocator("alias")
	added, err := applyDraftTestCommand(draft, DraftCommand{
		Action: DraftAddCandidate, ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
		ReasonCode: "SYNTHETIC_CANDIDATE_ADDED", ActorSubjectID: "manager",
		WorkspaceBody: "A", WorkspaceBodyDigest: ComputeWorkspaceBodyDigest("A"), ExactProjection: ptrProjection(completeProjection()),
		CreatedAt: time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC),
	}, allocator)
	if err != nil {
		t.Fatalf("add error = %v", err)
	}
	added, err = applyDraftTestCommand(added, DraftCommand{
		Action: DraftAddCandidate, ExpectedRevision: added.Revision, ExpectedContentDigest: added.ContentDigest,
		ReasonCode: "SYNTHETIC_CANDIDATE_ADDED", ActorSubjectID: "manager",
		WorkspaceBody: "A", WorkspaceBodyDigest: ComputeWorkspaceBodyDigest("A"), ExactProjection: ptrProjection(completeProjection()),
		CreatedAt: time.Date(2026, 8, 3, 10, 1, 0, 0, time.UTC),
	}, allocator)
	if err != nil {
		t.Fatalf("repeated body on distinct root error = %v", err)
	}
	refs := make([]WorkspaceQuestionRef, 0, 2)
	for _, item := range currentDraftItems(added) {
		if item.QuestionRef.Workspace != nil {
			refs = append(refs, *item.QuestionRef.Workspace)
		}
	}
	if len(refs) != 2 || refs[0].RootID == refs[1].RootID {
		t.Fatal("distinct roots invariant failed")
	}
	duplicate := slices.Clone(refs)
	duplicate[1].VersionID = duplicate[0].VersionID
	if err := ValidateWorkspaceQuestionRefs(duplicate, added.GenerationID); !errors.Is(err, ErrWorkspaceIdentityAlias) {
		t.Fatalf("duplicate version alias error = %v", err)
	}
	crossGeneration := slices.Clone(refs)
	crossGeneration[1].GenerationID = "aga-ws-generation-other"
	if err := ValidateWorkspaceQuestionRefs(crossGeneration, added.GenerationID); !errors.Is(err, ErrCrossGenerationParent) {
		t.Fatalf("cross-generation error = %v", err)
	}
	cycle := refs[0]
	cycle.ParentQuestionKey = &ParentQuestionKey{WorkspaceGenerationID: cycle.GenerationID, WorkspaceRootID: cycle.RootID, WorkspaceVersionID: cycle.VersionID}
	if err := ValidateWorkspaceQuestionRefs([]WorkspaceQuestionRef{cycle}, added.GenerationID); !errors.Is(err, ErrCyclicParent) {
		t.Fatalf("cycle error = %v", err)
	}
	duplicateProposal := slices.Clone(refs)
	duplicateProposal[1].ProposalID = duplicateProposal[0].ProposalID
	if err := ValidateWorkspaceQuestionRefs(duplicateProposal, added.GenerationID); !errors.Is(err, ErrWorkspaceIdentityAlias) {
		t.Fatalf("proposal alias error = %v", err)
	}
	duplicateRoot := slices.Clone(refs)
	duplicateRoot[1].RootID = duplicateRoot[0].RootID
	if err := ValidateWorkspaceQuestionRefs(duplicateRoot, added.GenerationID); !errors.Is(err, ErrWorkspaceIdentityAlias) {
		t.Fatalf("independent root alias error = %v", err)
	}
	missingParent := refs[0]
	missingParent.ParentQuestionKey = &ParentQuestionKey{WorkspaceGenerationID: added.GenerationID, WorkspaceRootID: missingParent.RootID, WorkspaceVersionID: "aga-ws-version-missing-0001", WorkspaceProposalID: "aga-ws-proposal-missing-0001", WorkspaceRootSequence: missingParent.RootSequence, WorkspaceBodyDigest: digestHex("missing")}
	if err := ValidateWorkspaceQuestionRefs([]WorkspaceQuestionRef{missingParent}, added.GenerationID); !errors.Is(err, ErrMissingParent) {
		t.Fatalf("missing parent error = %v", err)
	}
	crossRoot := refs[1]
	crossRoot.ParentQuestionKey = parentKeyForQuestion(WorkspaceQuestionReference(refs[0]))
	if err := ValidateWorkspaceQuestionRefs([]WorkspaceQuestionRef{refs[0], crossRoot}, added.GenerationID); !errors.Is(err, ErrCrossRootParent) {
		t.Fatalf("cross-root parent error = %v", err)
	}
	childOne := refs[0]
	childOne.VersionID = "aga-ws-version-child-0001"
	childOne.ProposalID = "aga-ws-proposal-child-0001"
	childOne.BodyDigest = digestHex("child-one")
	childOne.ParentQuestionKey = parentKeyForQuestion(WorkspaceQuestionReference(refs[0]))
	childTwo := childOne
	childTwo.VersionID = "aga-ws-version-child-0002"
	childTwo.ProposalID = "aga-ws-proposal-child-0002"
	childTwo.BodyDigest = digestHex("child-two")
	if err := ValidateWorkspaceQuestionRefs([]WorkspaceQuestionRef{refs[0], childOne, childTwo}, added.GenerationID); !errors.Is(err, ErrWorkspaceIdentityAlias) {
		t.Fatalf("branching lineage error = %v", err)
	}
}

func TestQuestionSnapshotReconstructsExactLeaves(t *testing.T) {
	draft := draftFixture(t, 3)
	allocator := NewSequentialIDAllocator("snapshot")
	base := currentDraftItems(draft)[1]
	reworded, err := applyDraftTestCommand(draft, DraftCommand{
		Action: DraftRewordCandidate, TargetQuestionKey: base.QuestionRef.Key(),
		ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
		ReasonCode: "SYNTHETIC_CANDIDATE_REWORDED", ActorSubjectID: "manager",
		WorkspaceBody: "Replacement", WorkspaceBodyDigest: ComputeWorkspaceBodyDigest("Replacement"),
		CreatedAt: time.Date(2026, 8, 3, 10, 2, 0, 0, time.UTC),
	}, allocator)
	if err != nil {
		t.Fatalf("reword error = %v", err)
	}
	snapshot, err := QuestionSnapshot(reworded)
	if err != nil {
		t.Fatalf("QuestionSnapshot() error = %v", err)
	}
	if len(snapshot) != 3 || snapshot[1].Workspace == nil || snapshot[1].RootSequence != 2 {
		t.Fatal("snapshot count invariant failed")
	}
	if snapshot[0].RootSequence != 1 || snapshot[2].RootSequence != 3 {
		t.Fatal("snapshot order invariant failed")
	}
	wrongBasePosition := cloneDraft(reworded)
	for index := range wrongBasePosition.Items {
		if wrongBasePosition.Items[index].QuestionRef.Workspace != nil {
			wrongBasePosition.Items[index].QuestionRef.RootSequence = 99
			wrongBasePosition.Items[index].QuestionRef.Workspace.RootSequence = 99
		}
	}
	if _, err := QuestionSnapshot(wrongBasePosition); !errors.Is(err, ErrWorkspaceIdentityAlias) {
		t.Fatalf("base reword rootSequence position error = %v", err)
	}
	for _, history := range reworded.Items {
		if history.QuestionRef.Key() == base.QuestionRef.Key() && history.Current {
			t.Fatal("superseded Base remained current")
		}
	}
	invalidCurrent := cloneDraft(reworded)
	for index := range invalidCurrent.Items {
		if invalidCurrent.Items[index].QuestionRef.Key() == base.QuestionRef.Key() {
			invalidCurrent.Items[index].Current = true
		}
	}
	if _, err := QuestionSnapshot(invalidCurrent); !errors.Is(err, ErrNonCurrentQuestion) {
		t.Fatalf("current parent error = %v", err)
	}
}

func TestDraftBatchPreviewIsAtomic(t *testing.T) {
	draft := draftFixture(t, 500)
	now := time.Date(2026, 8, 3, 14, 0, 0, 0, time.UTC)
	filter := DraftBatchFilter{MainDomainCodes: []string{"QUALITY_MANAGEMENT"}}
	preview, err := PreviewDraftBatch(draft, filter, DraftBatchAction{Action: DraftRetain}, now, NewSequentialIDAllocator("batch"))
	if err != nil {
		t.Fatalf("PreviewDraftBatch() error = %v", err)
	}
	if preview.Count != 500 || preview.PreviewID == "" || preview.FilterDigest == "" || preview.OrderedIdentityDigest == "" || preview.PreviewDigest == "" {
		t.Fatal("preview invariant failed")
	}
	execution := DraftBatchExecution{
		OperationID: "batch-operation-1", IdempotencyKey: "batch-idempotency-1",
		ExpectedGenerationID: draft.GenerationID, ExpectedDraftRevision: draft.Revision,
		ExpectedDraftContentDigest: draft.ContentDigest, PreviewID: preview.PreviewID, PreviewDigest: preview.PreviewDigest,
	}
	original := cloneDraft(draft)
	next, err := ExecuteDraftBatch(draft, preview, execution, filter, DraftBatchAction{Action: DraftRetain}, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("ExecuteDraftBatch() error = %v", err)
	}
	if next.Revision != draft.Revision+1 || !draftEqual(draft, original) {
		t.Fatal("batch execution was not immutable/atomic")
	}
	stale := preview
	stale.DraftContentDigest = digestHex("stale")
	if _, err := ExecuteDraftBatch(draft, stale, execution, filter, DraftBatchAction{Action: DraftRetain}, now); !errors.Is(err, ErrDraftConflict) {
		t.Fatalf("stale preview error = %v", err)
	}
	if _, err := ExecuteDraftBatch(draft, preview, execution, filter, DraftBatchAction{Action: DraftReclassifyMainDomain, MainDomainCode: "QUALITY_MANAGEMENT"}, now); !errors.Is(err, ErrPreviewMismatch) {
		t.Fatalf("preview action substitution error = %v", err)
	}
	tamperedID := execution
	tamperedID.PreviewID = "aga-ws-preview-other-0001"
	if _, err := ExecuteDraftBatch(draft, preview, tamperedID, filter, DraftBatchAction{Action: DraftRetain}, now); !errors.Is(err, ErrPreviewMismatch) {
		t.Fatalf("preview ID substitution error = %v", err)
	}
	if _, err := ExecuteDraftBatch(draft, preview, execution, filter, DraftBatchAction{Action: DraftRetain}, preview.ExpiresAt); !errors.Is(err, ErrPreviewExpired) {
		t.Fatalf("preview equality-boundary expiry error = %v", err)
	}
	tooMany := draftFixture(t, 501)
	if _, err := PreviewDraftBatch(tooMany, filter, DraftBatchAction{Action: DraftRetain}, now, NewSequentialIDAllocator("too-many")); !errors.Is(err, ErrBatchLimit) {
		t.Fatalf("501-item preview error = %v", err)
	}
	if _, err := PreviewDraftBatch(draft, filter, DraftBatchAction{Action: DraftAddTopic, TopicCode: "QUALITY_MANAGEMENT_SYSTEM"}, now, NewSequentialIDAllocator("reason")); !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("batch semantic action without reason error = %v", err)
	}
}

var (
	exactDraftOnce  sync.Once
	exactDraftValue Draft
	exactDraftError error
)

func draftFixture(t *testing.T, count int) Draft {
	t.Helper()
	if count < 1 || count > FrozenBaseQuestionCount {
		t.Fatalf("invalid draft fixture count %d", count)
	}
	exactDraftOnce.Do(func() {
		_, classification := exactClassificationFixture(t)
		exactDraftValue, exactDraftError = NewDraftFromClassification(classification, "aga-ws-generation-0001")
	})
	if exactDraftError != nil {
		t.Fatalf("NewDraftFromClassification() error = %v", exactDraftError)
	}
	draft := cloneDraft(exactDraftValue)
	if count == FrozenBaseQuestionCount {
		return draft
	}
	start := FrozenExternalUnresolvedCount
	draft.Items = make([]DraftItem, count)
	for index := 0; index < count; index++ {
		draft.Items[index] = cloneDraftItem(exactDraftValue.Items[start+index])
		draft.Items[index].QuestionRef.RootSequence = index + 1
		draft.Items[index].sealedBaseRootSequence = index + 1
	}
	draft.BaseQuestionCount = count
	draft.ClassificationItemCount = count
	draft.ContentDigest = ComputeDraftContentDigest(draft)
	return draft
}

func TestDraftRequiresResolvableSealedPasses(t *testing.T) {
	_, exact := exactClassificationFixture(t)
	short := exact
	short.Items = slices.Clone(exact.Items[:FrozenBaseQuestionCount-1])
	short.CandidateRecords = slices.Clone(exact.CandidateRecords[:FrozenBaseQuestionCount-1])
	short.ChallengeRecords = slices.Clone(exact.ChallengeRecords[:FrozenBaseQuestionCount-1])
	if _, err := NewDraftFromClassification(short, "aga-ws-generation-short"); !errors.Is(err, ErrPassBijection) {
		t.Fatalf("short sealed run error = %v", err)
	}

	blankRecommendation := exact
	blankRecommendation.Items = slices.Clone(exact.Items)
	blankRecommendation.Items[0].RecommendationState = ""
	blankRecommendation.Items[0].ItemSemanticDigest = ComputeItemSemanticDigest(blankRecommendation.Items[0])
	blankRecommendation.Aggregate = buildClassificationAggregate(FrozenTaxonomy(), blankRecommendation.Items, FrozenPassProposalRecordCount)
	blankRecommendation.AggregateDigest = blankRecommendation.Aggregate.AggregateDigest
	blankRecommendation.RunReceipt.AggregateDigest = blankRecommendation.AggregateDigest
	blankRecommendation.RunReceipt.ClassificationRunDigest = digestExcludingJSONFields("AGA-CLASSIFICATION-RUN-V1", blankRecommendation.RunReceipt, "classificationRunDigest")
	blankRecommendation.ClassificationRunDigest = blankRecommendation.RunReceipt.ClassificationRunDigest
	for index := range blankRecommendation.Items {
		blankRecommendation.Items[index].AggregateDigest = blankRecommendation.AggregateDigest
		blankRecommendation.Items[index].ClassificationRunDigest = blankRecommendation.ClassificationRunDigest
	}
	if _, err := NewDraftFromClassification(blankRecommendation, "aga-ws-generation-blank-state"); !errors.Is(err, ErrUnknownCode) {
		t.Fatalf("blank recommendation state error = %v", err)
	}

	taxonomy := FrozenTaxonomy()
	result := ClassificationResult{
		ClassificationRunID: "aga-classification-run-missing-passes", State: ClassificationRunSealed,
		TaxonomyVersion: taxonomy.Version, TaxonomyDigest: taxonomy.Digest,
		InputDigest: digestHex("run-input"), AggregateDigest: digestHex("aggregate"),
		ClassificationRunDigest: digestHex("run"), Items: []SealedClassificationItem{{
			Identity: packageIdentity(0), Projection: completeProjection(),
			AgreementConfidence: ConfidenceHigh, RecommendationState: RecommendationAutoProposed,
			GovernanceState: governanceState(), PassOneResultDigest: digestHex("missing-one"),
			PassTwoResultDigest: digestHex("missing-two"), ItemSemanticDigest: digestHex("item"),
		}},
	}
	if _, err := NewDraftFromClassification(result, "aga-ws-generation-missing"); !errors.Is(err, ErrPassBijection) {
		t.Fatalf("missing sealed pass records error = %v", err)
	}
}

func TestDraftReadinessExhaustsExactBaseAndPreservesUnchangedSourceGapConfidence(t *testing.T) {
	draft := draftFixture(t, FrozenBaseQuestionCount)
	current := currentDraftItems(draft)
	include := DispositionInclude
	sourceGapCount := 0
	for index := range current {
		if current[index].QuestionSourceProposalGap {
			sourceGapCount++
			if current[index].DraftAgreementConfidence == nil || current[index].SealedAgreementConfidence == nil {
				t.Fatal("unchanged sealed source-gap confidence was not copied")
			}
		}
		if current[index].ReviewState == ReviewPendingManager {
			current[index].ReviewState = ReviewManagerDisposed
			current[index].Disposition = &include
		}
		replaceCurrentDraftItem(&draft, current[index])
	}
	if sourceGapCount != FrozenSourceGapCount {
		t.Fatalf("source-gap count = %d", sourceGapCount)
	}
	draft.ContentDigest = ComputeDraftContentDigest(draft)
	ready, err := applyDraftTestCommand(draft, DraftCommand{
		Action: DraftMarkReady, ExpectedRevision: draft.Revision, ExpectedContentDigest: draft.ContentDigest,
		ReasonCode: "SIMULATION_SOURCE_GAP_OVERRIDE", ActorSubjectID: "manager-1",
		CreatedAt: time.Date(2026, 8, 3, 16, 0, 0, 0, time.UTC), ReadinessEventID: "aga-ws-readiness-source-gap-0001",
		ProviderScopeProfileDigest: digestHex("source-gap-scope-profile"),
	}, NewSequentialIDAllocator("source-gap-ready"))
	if err != nil {
		t.Fatalf("exact source-gap readiness error = %v", err)
	}
	if ready.State != DraftReadyForDemoSimulation || len(ready.ReadinessEvents) != 1 {
		t.Fatal("exact source-gap draft did not become ready")
	}
	first := currentDraftItems(ready)[0]
	working, err := applyDraftTestCommand(ready, DraftCommand{
		Action: DraftRetain, TargetQuestionKey: first.QuestionRef.Key(),
		ExpectedRevision: ready.Revision, ExpectedContentDigest: ready.ContentDigest,
	}, NewSequentialIDAllocator("source-gap-retain"))
	if err != nil {
		t.Fatalf("ready successor error = %v", err)
	}
	if _, err := applyDraftTestCommand(working, DraftCommand{
		Action: DraftMarkReady, ExpectedRevision: working.Revision, ExpectedContentDigest: working.ContentDigest,
		ReasonCode: "SIMULATION_SOURCE_GAP_OVERRIDE", ActorSubjectID: "manager-1",
		CreatedAt: time.Date(2026, 8, 3, 16, 1, 0, 0, time.UTC), ReadinessEventID: "aga-ws-readiness-source-gap-0001",
		ProviderScopeProfileDigest: digestHex("source-gap-scope-profile"),
	}, NewSequentialIDAllocator("source-gap-duplicate")); !errors.Is(err, ErrDraftNotReady) {
		t.Fatalf("duplicate readiness event error = %v", err)
	}
}

func alternateProjection() ProposalProjection {
	projection := completeProjection()
	projection.MainDomainCode = "SAFETY_MANAGEMENT_RISK_ASSESSMENT"
	projection.TopicCodes = []string{"SAFETY_MANAGEMENT_SYSTEM"}
	projection.InspectionProfileCodes = []string{"RUNWAY_SAFETY"}
	projection.InspectionTypeCodes = []string{"ON_SITE_INSPECTION"}
	projection.CanonicalTargetKind = "LOCATION"
	projection.TargetProfileCode = "RUNWAY_SYSTEM"
	projection.OperationQualifiers = []Qualifier{{Key: "OPERATION_STATUS", Value: "TEMPORARILY_RESTRICTED"}}
	projection.OperationQualifiers = append(projection.OperationQualifiers, Qualifier{Key: "RUNWAY_USE", Value: "MIXED"})
	projection.ActivityQualifiers = []Qualifier{{Key: "ACTIVITY_TYPE", Value: "RISK_ASSESSMENT"}}
	projection.ApplicabilityDisposition = "CONDITIONAL_ON_OPERATION"
	projection.EvidenceExpectationCodes = []string{"RISK_ASSESSMENT"}
	projection.ExternalInvolvements = []ExternalInvolvement{externalEdge("ANSP", "COORDINATION", "ANSP_COORDINATION_REQUIRED")}
	return projection
}

func exactProjection() ProposalProjection {
	projection := completeProjection()
	projection.MainDomainCode = "AERODROME_DATA_INFORMATION_PUBLICATION"
	projection.TopicCodes = []string{"AERODROME_DATA_QUALITY"}
	projection.InspectionProfileCodes = []string{"AERODROME_DATA_QUALITY"}
	projection.InspectionTypeCodes = []string{"SPECIAL_PURPOSE"}
	projection.CanonicalTargetKind = "SYSTEM"
	projection.TargetProfileCode = "AERODROME_DATA_SYSTEM"
	projection.OperationQualifiers = []Qualifier{{Key: "OPERATION_STATUS", Value: "ACTIVE"}}
	projection.ActivityQualifiers = []Qualifier{{Key: "ACTIVITY_TYPE", Value: "MAINTENANCE"}}
	projection.ApplicabilityDisposition = "APPLICABLE"
	projection.EvidenceExpectationCodes = []string{"SOURCE_REFERENCE"}
	projection.ExternalInvolvements = nil
	return projection
}

func ptrProjection(value ProposalProjection) *ProposalProjection { return &value }
