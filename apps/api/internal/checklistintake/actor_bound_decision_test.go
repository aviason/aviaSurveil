package checklistintake

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aviason/aviaSurveil/internal/identity"
)

func actorBoundPrincipal() identity.Principal {
	return identity.Principal{
		SubjectID:      "admin-subject",
		OrganizationID: "CAA",
		SessionID:      "session-1",
		Roles:          []identity.Role{identity.RoleAdmin},
	}
}

func actorBoundScopeCommand() ActorBoundCandidateScopeCommand {
	return ActorBoundCandidateScopeCommand{
		MembershipID:              "membership-1",
		OperationID:               "operation-1",
		IdempotencyKey:            "idempotency-1",
		ReviewedPackageSHA256:     "sha256:30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2",
		ScopeDigest:               "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		FormCodes:                 []string{"FSS-AGA-FORM-048", "FSS-AGA-FORM-001"},
		MaxFormsPerBatch:          2,
		MaxQuestionProposalsBatch: 250,
		Reason:                    "candidate-only scope review",
	}
}

type actorBoundFixtureResolver struct {
	actor                    ActorBoundActorContext
	scope                    ActorBoundScopeContext
	actorErr                 error
	scopeErr                 error
	ignoreMembershipSelector bool
}

func (resolver actorBoundFixtureResolver) ResolveActor(_ context.Context, principal identity.Principal, membershipID string) (ActorBoundActorContext, error) {
	if resolver.actorErr != nil {
		return ActorBoundActorContext{}, resolver.actorErr
	}
	if principal.SubjectID != resolver.actor.Principal.SubjectID || principal.OrganizationID != resolver.actor.Principal.OrganizationID || principal.SessionID != resolver.actor.Principal.SessionID || !sameRoles(principal.Roles, resolver.actor.Principal.Roles) || (!resolver.ignoreMembershipSelector && membershipID != resolver.actor.MembershipID) {
		return ActorBoundActorContext{}, errors.New("actor context does not match active membership and session")
	}
	return resolver.actor, nil
}

func (resolver actorBoundFixtureResolver) ResolveScope(_ context.Context, packageDigest, scopeDigest string) (ActorBoundScopeContext, error) {
	if resolver.scopeErr != nil {
		return ActorBoundScopeContext{}, resolver.scopeErr
	}
	if packageDigest != resolver.scope.ReviewedPackageSHA256 || scopeDigest != resolver.scope.ScopeDigest {
		return ActorBoundScopeContext{}, errors.New("reviewed package or scope is not registered")
	}
	return resolver.scope, nil
}

func sameRoles(left, right []identity.Role) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func newActorBoundLedger() *ActorBoundCandidateScopeLedger {
	command := actorBoundScopeCommand()
	return NewActorBoundCandidateScopeLedger(actorBoundFixtureResolverForCommand(command))
}

func actorBoundFixtureResolverForCommand(command ActorBoundCandidateScopeCommand) actorBoundFixtureResolver {
	return actorBoundFixtureResolver{
		actor: ActorBoundActorContext{Principal: actorBoundPrincipal(), MembershipID: command.MembershipID, MembershipActive: true, SessionExpiresAt: time.Now().UTC().Add(time.Hour)},
		scope: ActorBoundScopeContext{ReviewedPackageSHA256: command.ReviewedPackageSHA256, ScopeDigest: command.ScopeDigest, FormCodes: append([]string(nil), command.FormCodes...)},
	}
}

func appendActorBound(ledger *ActorBoundCandidateScopeLedger, command ActorBoundCandidateScopeCommand) (ActorBoundCandidateScopeDecision, error) {
	return ledger.Append(context.Background(), actorBoundPrincipal(), command)
}

func TestActorBoundCandidateScopeRequiresFreshAdminSessionAndMembership(t *testing.T) {
	ledger := newActorBoundLedger()
	command := actorBoundScopeCommand()
	if _, err := appendActorBound(ledger, command); err != nil {
		t.Fatalf("valid actor-bound scope should append: %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(*identity.Principal, *ActorBoundCandidateScopeCommand)
		message string
	}{
		{name: "missing subject", mutate: func(principal *identity.Principal, _ *ActorBoundCandidateScopeCommand) { principal.SubjectID = "" }, message: "actor"},
		{name: "missing session", mutate: func(principal *identity.Principal, _ *ActorBoundCandidateScopeCommand) { principal.SessionID = "" }, message: "session"},
		{name: "missing membership", mutate: func(_ *identity.Principal, command *ActorBoundCandidateScopeCommand) { command.MembershipID = "" }, message: "membership"},
		{name: "manager is not candidate intake admin", mutate: func(principal *identity.Principal, _ *ActorBoundCandidateScopeCommand) {
			principal.Roles = []identity.Role{identity.RoleDepartmentManager}
		}, message: "Admin"},
		{name: "non CAA principal", mutate: func(principal *identity.Principal, _ *ActorBoundCandidateScopeCommand) {
			principal.OrganizationID = "ORG-OTHER"
		}, message: "authority"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			principal := actorBoundPrincipal()
			candidate := actorBoundScopeCommand()
			testCase.mutate(&principal, &candidate)
			candidate.IdempotencyKey = "idempotency-" + strings.ReplaceAll(testCase.name, " ", "-")
			if _, err := ledger.Append(context.Background(), principal, candidate); err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(testCase.message)) {
				t.Fatalf("expected actor-bound %s rejection, got %v", testCase.message, err)
			}
		})
	}
}

func TestActorBoundCandidateScopeRequiresResolverBoundFacts(t *testing.T) {
	command := actorBoundScopeCommand()
	if _, err := NewActorBoundCandidateScopeLedger(nil).Append(context.Background(), actorBoundPrincipal(), command); !errors.Is(err, ErrActorBoundContextUnavailable) {
		t.Fatalf("missing resolver must fail closed, got %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(*actorBoundFixtureResolver)
		message string
	}{
		{name: "revoked session", mutate: func(resolver *actorBoundFixtureResolver) { resolver.actor.SessionRevoked = true }, message: "session"},
		{name: "inactive membership", mutate: func(resolver *actorBoundFixtureResolver) { resolver.actor.MembershipActive = false }, message: "membership"},
		{name: "expired session", mutate: func(resolver *actorBoundFixtureResolver) {
			resolver.actor.SessionExpiresAt = time.Now().UTC().Add(-time.Minute)
		}, message: "session"},
		{name: "unknown reviewed scope", mutate: func(resolver *actorBoundFixtureResolver) {
			resolver.scopeErr = errors.New("reviewed scope is not registered")
		}, message: "scope"},
		{name: "resolver returns different forms", mutate: func(resolver *actorBoundFixtureResolver) { resolver.scope.FormCodes = []string{"FSS-AGA-FORM-999"} }, message: "scope"},
		{name: "resolver returns different active membership", mutate: func(resolver *actorBoundFixtureResolver) {
			resolver.ignoreMembershipSelector = true
			resolver.actor.MembershipID = "membership-2"
		}, message: "membership"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			resolver := actorBoundFixtureResolverForCommand(command)
			testCase.mutate(&resolver)
			if _, err := NewActorBoundCandidateScopeLedger(resolver).Append(context.Background(), actorBoundPrincipal(), command); err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(testCase.message)) {
				t.Fatalf("expected resolver-bound %s rejection, got %v", testCase.message, err)
			}
		})
	}
	for name, mutate := range map[string]func(*identity.Principal, *ActorBoundCandidateScopeCommand){
		"valid but different Admin context": func(principal *identity.Principal, _ *ActorBoundCandidateScopeCommand) {
			principal.SubjectID = "other-admin"
		},
		"valid but unknown package digest": func(_ *identity.Principal, command *ActorBoundCandidateScopeCommand) {
			command.ReviewedPackageSHA256 = "sha256:" + strings.Repeat("b", 64)
		},
		"valid but unknown scope digest": func(_ *identity.Principal, command *ActorBoundCandidateScopeCommand) {
			command.ScopeDigest = "sha256:" + strings.Repeat("b", 64)
		},
	} {
		t.Run(name, func(t *testing.T) {
			principal := actorBoundPrincipal()
			candidate := actorBoundScopeCommand()
			mutate(&principal, &candidate)
			candidate.IdempotencyKey = "idempotency-" + strings.ReplaceAll(name, " ", "-")
			if _, err := NewActorBoundCandidateScopeLedger(actorBoundFixtureResolverForCommand(command)).Append(context.Background(), principal, candidate); err == nil || !errors.Is(err, ErrActorBoundContextUnavailable) {
				t.Fatalf("expected valid-shaped unknown or mismatched fact to fail closed, got %v", err)
			}
		})
	}
}

func TestActorBoundCandidateScopeGeneratesAppendOnlyServerLeaf(t *testing.T) {
	ledger := newActorBoundLedger()
	first, err := appendActorBound(ledger, actorBoundScopeCommand())
	if err != nil {
		t.Fatal(err)
	}
	if first.DecisionID == "" || first.DecisionRootID == "" || first.DecisionID == "AGA-P2-AUTH-2026-08-01-0001" {
		t.Fatalf("server must generate a non-user-supplied decision ID: %+v", first)
	}
	if first.ActorSubjectID != "admin-subject" || first.ActorMembershipID != "membership-1" || first.ActorSessionID != "session-1" || first.ActorRole != identity.RoleAdmin {
		t.Fatalf("actor binding was not persisted: %+v", first)
	}
	if !first.CandidateOnly || first.SourceMappingAuthorized || first.SourceAuthorityAttested || first.ApplicabilityAuthorized || first.RiskDecisionAuthorized || first.FunctionalAssignmentAuthorized || first.TechnicalApprovalAuthorized || first.PublicationAuthorized || first.ReleaseAuthorized || first.ProductionAuthorized {
		t.Fatalf("candidate-only authority boundary was not fail-closed: %+v", first)
	}
	if !strings.HasPrefix(first.SemanticPayloadDigest, "sha256:") {
		t.Fatalf("decision digest is missing: %+v", first)
	}

	replay, err := appendActorBound(ledger, actorBoundScopeCommand())
	if err != nil || replay.DecisionID != first.DecisionID || replay.SemanticPayloadDigest != first.SemanticPayloadDigest {
		t.Fatalf("identical actor-bound replay must be stable: first=%+v replay=%+v err=%v", first, replay, err)
	}

	divergent := actorBoundScopeCommand()
	divergent.Reason = "divergent decision"
	if _, err := appendActorBound(ledger, divergent); err != ErrActorBoundIdempotencyConflict {
		t.Fatalf("divergent idempotency replay must conflict, got %v", err)
	}

	successorCommand := actorBoundScopeCommand()
	successorCommand.OperationID = "operation-2"
	successorCommand.IdempotencyKey = "idempotency-2"
	successorCommand.ExpectedPriorLeafID = first.DecisionID
	successorCommand.ExpectedPriorDigest = first.SemanticPayloadDigest
	successorCommand.Reason = "bounded successor decision"
	successor, err := appendActorBound(ledger, successorCommand)
	if err != nil {
		t.Fatal(err)
	}
	if successor.Revision != 2 || successor.SupersedesDecisionID != first.DecisionID || successor.DecisionRootID != first.DecisionRootID {
		t.Fatalf("successor must be append-only and CAS-bound: first=%+v successor=%+v", first, successor)
	}
	current, ok := ledger.Current(first.DecisionRootID)
	if !ok || current.DecisionID != successor.DecisionID || current.Revision != 2 {
		t.Fatalf("current leaf must be the append-only successor: current=%+v ok=%v", current, ok)
	}
	history, ok := ledger.History(first.DecisionRootID)
	if !ok || len(history) != 2 || history[0].DecisionID != first.DecisionID || history[1].DecisionID != successor.DecisionID {
		t.Fatalf("append-only history must retain both immutable leaves: history=%+v ok=%v", history, ok)
	}
	history[0].FormCodes[0] = "tampered"
	reloadedHistory, ok := ledger.History(first.DecisionRootID)
	if !ok || reloadedHistory[0].FormCodes[0] == "tampered" {
		t.Fatalf("history reads must not expose mutable ledger state: history=%+v ok=%v", reloadedHistory, ok)
	}
	if _, err := appendActorBound(ledger, func() ActorBoundCandidateScopeCommand {
		stale := successorCommand
		stale.OperationID = "operation-3"
		stale.IdempotencyKey = "idempotency-3"
		stale.ExpectedPriorLeafID = first.DecisionID
		stale.ExpectedPriorDigest = first.SemanticPayloadDigest
		return stale
	}()); err != ErrActorBoundStaleRevision {
		t.Fatalf("stale successor must be rejected, got %v", err)
	}
}

func TestActorBoundCandidateScopeKeepsBatchAndScopeBounds(t *testing.T) {
	ledger := newActorBoundLedger()
	tooManyForms := actorBoundScopeCommand()
	tooManyForms.MaxFormsPerBatch = MaxActorBoundBatchForms + 1
	if _, err := appendActorBound(ledger, tooManyForms); err == nil {
		t.Fatal("batch form limit must be enforced")
	}
	tooManyQuestions := actorBoundScopeCommand()
	tooManyQuestions.MaxQuestionProposalsBatch = MaxActorBoundBatchQuestionProposals + 1
	tooManyQuestions.IdempotencyKey = "idempotency-too-many-questions"
	if _, err := appendActorBound(ledger, tooManyQuestions); err == nil {
		t.Fatal("batch question limit must be enforced")
	}
	invalidDigest := actorBoundScopeCommand()
	invalidDigest.ScopeDigest = "not-a-sha"
	invalidDigest.IdempotencyKey = "idempotency-invalid-digest"
	if _, err := appendActorBound(ledger, invalidDigest); err == nil {
		t.Fatal("scope digest must be immutable SHA-256")
	}
	for name, malformed := range map[string]string{
		"leading whitespace": " " + actorBoundScopeCommand().ScopeDigest,
		"uppercase hex":      "sha256:" + strings.Repeat("A", 64),
	} {
		candidate := actorBoundScopeCommand()
		candidate.ScopeDigest = malformed
		candidate.IdempotencyKey = "idempotency-invalid-" + strings.ReplaceAll(name, " ", "-")
		if _, err := appendActorBound(ledger, candidate); err == nil {
			t.Fatalf("%s digest must be rejected", name)
		}
	}
}
