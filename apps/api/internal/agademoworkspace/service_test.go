package agademoworkspace

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

type serviceTestStore struct {
	workspace  preprod.LoadedWorkspace
	applyCalls int
	getCalls   int
	responses  map[preprod.StoredResponseKey]preprod.IdempotencyResponse
	applyErr   error
}

func (store *serviceTestStore) LoadAndSeal(context.Context, preprod.LoadInput) (preprod.WorkspaceSealReceipt, error) {
	return preprod.WorkspaceSealReceipt{}, nil
}
func (store *serviceTestStore) Snapshot(context.Context) (preprod.LoadedWorkspace, error) {
	return store.workspace, nil
}
func (store *serviceTestStore) ApplyDraftCommand(_ context.Context, command aga.DraftCommand) (aga.Draft, error) {
	store.applyCalls++
	if store.applyErr != nil {
		return aga.Draft{}, store.applyErr
	}
	draft := store.workspace.Draft.Draft
	draft.Revision++
	draft.ContentDigest = digestForTest()
	return draft, nil
}
func (store *serviceTestStore) AppendQuestionVersion(context.Context, preprod.AppendQuestionVersionInput) (preprod.WorkspaceQuestionVersion, error) {
	return preprod.WorkspaceQuestionVersion{}, nil
}
func (store *serviceTestStore) PutIdempotencyResponse(_ context.Context, response preprod.IdempotencyResponse) (preprod.IdempotencyResponse, bool, error) {
	if store.responses == nil {
		store.responses = map[preprod.StoredResponseKey]preprod.IdempotencyResponse{}
	}
	key := preprod.StoredResponseKey{GenerationID: response.GenerationID, ActorSubjectID: response.ActorSubjectID, OperationID: response.OperationID, IdempotencyKey: response.IdempotencyKey}
	if existing, ok := store.responses[key]; ok {
		return existing, true, nil
	}
	store.responses[key] = response
	return response, false, nil
}
func (store *serviceTestStore) GetIdempotencyResponse(_ context.Context, key preprod.StoredResponseKey) (preprod.IdempotencyResponse, bool, error) {
	store.getCalls++
	response, ok := store.responses[key]
	return response, ok, nil
}
func (store *serviceTestStore) ResetGeneration(context.Context, preprod.ResetInput) (preprod.Generation, preprod.ResetTombstone, error) {
	generation := store.workspace.Generation
	generation.GenerationID = "aga-ws-reset-00000001"
	return generation, preprod.ResetTombstone{}, nil
}

func testWorkspace() preprod.LoadedWorkspace {
	return preprod.LoadedWorkspace{
		Generation: preprod.Generation{GenerationID: "aga-ws-generation-0001", State: preprod.GenerationActive, Revision: 1, ClassificationRunDigest: digestForTest(), TaxonomyDigest: digestForTest(), FixtureDigest: digestForTest(), SealDigest: digestForTest(), CreatedAt: time.Unix(1, 0).UTC()},
		Draft:      preprod.DraftRecord{Draft: aga.Draft{DraftID: "aga-ws-draft-0001", GenerationID: "aga-ws-generation-0001", GenerationState: preprod.GenerationActive, Revision: 1, ContentDigest: digestForTest(), State: aga.DraftWorking}},
	}
}

func digestForTest() string {
	return "sha256:0000000000000000000000000000000000000000000000000000000000000000"
}

func TestWorkspaceDirectIDDenialIsNeutral(t *testing.T) {
	store := &serviceTestStore{workspace: testWorkspace(), applyErr: aga.ErrNonCurrentQuestion}
	service := NewService(ServiceConfig{Store: store, Resolver: StaticBindingResolver{Bindings: map[string]preprod.AuthorityBinding{"manager": bindingFor("manager", "ORG", "manager")}}})
	command := CommandEnvelope{OperationID: OperationInclude, IdempotencyKey: "key", ExpectedGenerationID: store.workspace.Generation.GenerationID, ExpectedDraftRevision: 1, ExpectedDraftContentDigest: store.workspace.Draft.Draft.ContentDigest, TargetQuestionKey: "workspace\x1fguess\x1fguess\x1fguess", ReasonCode: "CLASSIFICATION_EXPERT_REVIEW"}
	_, err := service.Command(context.Background(), principalFor("manager", "ORG", identity.RoleDepartmentManager), FamilyClassificationCommand, command)
	if !errors.Is(err, ErrNeutralDenied) {
		t.Fatalf("direct identity denial = %v, want neutral denial", err)
	}
}

func TestWorkspaceIncludeFailsClosedWhenSimulationScopeIsUnavailable(t *testing.T) {
	workspace := testWorkspace()
	baseIdentity := aga.BaseIdentity{PackageVersion: "AGA_PACKAGE", PackageJSONSHA256: digestForTest(), FormCode: "FSS-AGA-FORM-001", ProposalID: "proposal-001", Ordinal: 1, TextDigest: digestForTest()}
	workspace.Items = []preprod.ClassificationItem{{Identity: baseIdentity, QuestionKey: "server-returned-question-key"}}
	store := &serviceTestStore{workspace: workspace, responses: map[preprod.StoredResponseKey]preprod.IdempotencyResponse{}}
	service := NewService(ServiceConfig{Store: store, Resolver: StaticBindingResolver{Bindings: map[string]preprod.AuthorityBinding{"manager": bindingFor("manager", "ORG", "manager")}}})
	command := CommandEnvelope{OperationID: OperationInclude, IdempotencyKey: "scope-unavailable", ExpectedGenerationID: workspace.Generation.GenerationID, ExpectedDraftRevision: 1, ExpectedDraftContentDigest: workspace.Draft.Draft.ContentDigest, TargetQuestionKey: "server-returned-question-key", ReasonCode: "CLASSIFICATION_EXPERT_REVIEW"}
	if _, err := service.Command(context.Background(), principalFor("manager", "ORG", identity.RoleDepartmentManager), FamilyClassificationCommand, command); !errors.Is(err, ErrIncludedQuestionIneligible) {
		t.Fatalf("include without a server simulation scope = %v, want %v", err, ErrIncludedQuestionIneligible)
	}
	if store.applyCalls != 0 {
		t.Fatalf("ineligible include reached Draft write path %d time(s)", store.applyCalls)
	}
}

func TestWorkspaceIdempotencyLookupPrecedesDomainSnapshot(t *testing.T) {
	store := &serviceTestStore{workspace: testWorkspace(), responses: map[preprod.StoredResponseKey]preprod.IdempotencyResponse{}}
	service := NewService(ServiceConfig{Store: store, Resolver: StaticBindingResolver{Bindings: map[string]preprod.AuthorityBinding{"manager": bindingFor("manager", "ORG", "manager")}}})
	command := CommandEnvelope{OperationID: OperationInclude, IdempotencyKey: "key", ExpectedGenerationID: store.workspace.Generation.GenerationID, ExpectedDraftRevision: 1, ExpectedDraftContentDigest: store.workspace.Draft.Draft.ContentDigest, TargetQuestionKey: "base\x1fguess", ReasonCode: "CLASSIFICATION_EXPERT_REVIEW"}
	_, _ = service.Command(context.Background(), principalFor("manager", "ORG", identity.RoleDepartmentManager), FamilyClassificationCommand, command)
	if store.getCalls != 1 {
		t.Fatalf("idempotency calls = %d, want one", store.getCalls)
	}
}

func TestWorkspaceCommandResponseIsStoredAndReplayed(t *testing.T) {
	store := &serviceTestStore{workspace: testWorkspace(), responses: map[preprod.StoredResponseKey]preprod.IdempotencyResponse{}}
	service := NewService(ServiceConfig{Store: store, Resolver: StaticBindingResolver{Bindings: map[string]preprod.AuthorityBinding{"manager": bindingFor("manager", "ORG", "manager")}}})
	command := CommandEnvelope{OperationID: OperationInclude, IdempotencyKey: "key", ExpectedGenerationID: store.workspace.Generation.GenerationID, ExpectedDraftRevision: 1, ExpectedDraftContentDigest: store.workspace.Draft.Draft.ContentDigest, TargetQuestionKey: "base\x1fguess", ReasonCode: "CLASSIFICATION_EXPERT_REVIEW"}
	bytes, _ := json.Marshal(CommandResponse{OperationID: OperationInclude})
	store.responses[preprod.StoredResponseKey{GenerationID: command.ExpectedGenerationID, ActorSubjectID: "manager", OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey}] = preprod.IdempotencyResponse{GenerationID: command.ExpectedGenerationID, ActorSubjectID: "manager", OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey, CommandHash: commandDigest(command), AuthorizationScopeDigest: AuthorizationScopeDigest(principalFor("manager", "ORG", identity.RoleDepartmentManager), bindingFor("manager", "ORG", "manager"), command.OperationID), Response: bytes}
	replayed, err := service.Command(context.Background(), principalFor("manager", "ORG", identity.RoleDepartmentManager), FamilyClassificationCommand, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || store.applyCalls != 0 {
		t.Fatalf("replay = %+v, apply calls = %d", replayed, store.applyCalls)
	}
}

func TestQueryRequestRejectsUnboundedPageBeforePagination(t *testing.T) {
	request := QueryRequest{OperationID: OperationSearchItems, Page: MaxWorkspacePage + 1, PageSize: 25}
	if !errors.Is(request.Validate(), ErrMalformedCommand) {
		t.Fatalf("large page validation = %v, want malformed command", request.Validate())
	}

	start, end := boundedPageWindow(math.MaxInt, 25, 1)
	if start != 1 || end != 1 {
		t.Fatalf("bounded page window = (%d, %d), want empty tail", start, end)
	}
}
