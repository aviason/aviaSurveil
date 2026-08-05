//go:build preproddemo

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	workspace "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agademoworkspace"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

const (
	probeOrganization = "AGA-DEMO-CAA"
	probeReason       = "MANAGER_SCOPE_DECISION"
	probeReadiness    = "SIMULATION_SOURCE_GAP_OVERRIDE"
)

type probeAccounts struct {
	admin, manager, inspector, lead, auditee, other identity.Principal
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	if err := run(ctx); err != nil {
		// The harness retains private stderr. Do not expose subjects, question
		// keys, or object IDs on the public command surface.
		fmt.Fprintln(os.Stderr, "ERR_AGA_HYBRID_CONNECTED_LIFECYCLE_PROBE")
		fmt.Fprintf(os.Stderr, "probe diagnostic: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	readerURL := os.Getenv("AVIA_AGA_DEMO_WORKSPACE_READER_DATABASE_URL")
	commandURL := os.Getenv("AVIA_AGA_DEMO_WORKSPACE_COMMAND_DATABASE_URL")
	if readerURL == "" || commandURL == "" {
		return errors.New("workspace database urls are required")
	}
	readerPool, err := database.Open(ctx, readerURL)
	if err != nil {
		return err
	}
	defer readerPool.Close()
	commandPool, err := database.Open(ctx, commandURL)
	if err != nil {
		return err
	}
	defer commandPool.Close()
	readerStore, err := preprod.NewPostgresReader(readerPool)
	if err != nil {
		return err
	}
	commandStore, err := preprod.NewPostgresCommandStore(commandPool)
	if err != nil {
		return err
	}
	bindingResolver, err := workspace.NewPostgresBindingResolver(readerPool)
	if err != nil {
		return err
	}
	scopeResolver := workspace.NewPostgresRecommendationScopeResolver(readerPool)
	service := workspace.NewService(workspace.ServiceConfig{
		ReaderStore: readerStore, CommandStore: commandStore, Resolver: bindingResolver,
		RecommendationScopes: scopeResolver, LifecycleBindings: workspace.NewFixtureLifecycleBindingResolver(),
	})

	current, err := commandStore.Snapshot(ctx)
	if err != nil {
		return err
	}
	accounts, err := accountsFromFixture(current.Fixture)
	if err != nil {
		return err
	}
	draft := current.Draft.Draft
	classificationCount := 0
	for _, item := range append([]aga.DraftItem(nil), draft.Items...) {
		if !item.Current || item.ReviewState != aga.ReviewPendingManager || item.Disposition != nil {
			continue
		}
		reason := probeReason
		if item.QuestionSourceProposalGap {
			reason = probeReadiness
		}
		response, commandErr := service.Command(ctx, accounts.manager, workspace.FamilyClassificationCommand, workspace.CommandEnvelope{
			OperationID: workspace.OperationInclude, IdempotencyKey: fmt.Sprintf("aga-ws-probe-classification-%06d", classificationCount+1),
			ExpectedGenerationID: draft.GenerationID, ExpectedDraftRevision: draft.Revision, ExpectedDraftContentDigest: draft.ContentDigest,
			TargetQuestionKey: item.QuestionRef.Key(), ReasonCode: reason, Action: aga.DraftInclude,
		})
		if commandErr != nil {
			return fmt.Errorf("classification command failed: %w (revision=%d items=%d)", commandErr, draft.Revision, len(draft.Items))
		}
		if response.Draft == nil {
			return errors.New("classification command returned no draft")
		}
		draft = *response.Draft
		classificationCount++
	}
	current.Draft.Draft = draft

	request := workspace.CommandEnvelope{
		OperationID: workspace.OperationCreateRecommendation, IdempotencyKey: "aga-ws-probe-recommendation-0001", ExpectedGenerationID: current.Generation.GenerationID,
		OrganizationID: probeOrganization, ProviderScopeRootID: "aga-ws-scope-root-matching", ProviderScopeID: "aga-ws-scope-matching", ProviderScopeVersion: 1,
		ProviderTypeID: "AERODROME_OPERATOR", DepartmentID: "AGA-DEMO-DEPARTMENT", OrganizationalUnitID: "AGA-DEMO-UNIT", TargetID: "aga-ws-target-matching",
		CanonicalTargetKind: "FACILITY", TargetProfileCode: "RFFS_FUNCTION", InspectionProfileCode: "EMERGENCY_AND_RFFS", InspectionTypeCode: "ON_SITE_INSPECTION",
		OperationQualifiers: []aga.Qualifier{{Key: "OPERATION_STATUS", Value: "ACTIVE"}}, ActivityQualifiers: []aga.Qualifier{{Key: "ACTIVITY_TYPE", Value: "EMERGENCY_RESPONSE"}},
		TaxonomyVersion: draft.TaxonomyVersion, TaxonomyDigest: draft.TaxonomyDigest, ClassificationRunID: draft.ClassificationRunID, ClassificationRunDigest: draft.ClassificationRunDigest,
		DraftID: draft.DraftID, DraftRevision: draft.Revision, DraftContentDigest: draft.ContentDigest, ExpectedDraftRevision: draft.Revision,
	}
	facts, err := scopeResolver(ctx, current, recommendationRequest(request))
	if err != nil || len(facts) != 1 {
		return errors.New("provider scope fact resolution failed")
	}
	scope := facts[0]
	readyResponse, err := service.Command(ctx, accounts.manager, workspace.FamilyClassificationCommand, workspace.CommandEnvelope{
		OperationID: workspace.OperationMarkReady, IdempotencyKey: "aga-ws-probe-readiness-0001", ExpectedGenerationID: draft.GenerationID,
		ExpectedDraftRevision: draft.Revision, ExpectedDraftContentDigest: draft.ContentDigest, Action: aga.DraftMarkReady, ReasonCode: probeReadiness,
		ReadinessEventID: "aga-ws-readiness-probe-0001", ProviderScopeProfileDigest: scope.ProfileDigest,
	})
	if err != nil {
		return fmt.Errorf("readiness command failed: %w (revision=%d items=%d contentDigestPresent=%t)", err, draft.Revision, len(draft.Items), draft.ContentDigest != "")
	}
	if readyResponse.Draft == nil {
		return errors.New("readiness command returned no draft")
	}
	draft = *readyResponse.Draft
	current.Draft.Draft = draft
	readiness := draft.ReadinessEvents[len(draft.ReadinessEvents)-1]
	request.DraftRevision, request.ExpectedDraftRevision, request.DraftContentDigest = draft.Revision, draft.Revision, draft.ContentDigest
	request.ReadinessEventID, request.ReadinessEventDigest, request.EffectiveAt = readiness.ReadinessEventID, readiness.ReadinessEventDigest, scope.EffectiveFrom.Add(time.Minute)
	recommendationResponse, err := service.Command(ctx, accounts.manager, workspace.FamilyRecommendationCommand, request)
	if err != nil {
		return fmt.Errorf("recommendation command failed: %w", err)
	}
	if recommendationResponse.Recommendation == nil {
		return errors.New("recommendation command failed")
	}
	recommendation := recommendationResponse.Recommendation.Recommendation
	inspector, lead, err := lifecycleFacts(ctx, current, recommendation)
	if err != nil {
		return err
	}
	inspectionResponse, err := service.Command(ctx, accounts.manager, workspace.FamilyRecommendationCommand, workspace.CommandEnvelope{
		OperationID: workspace.OperationCreateInspection, IdempotencyKey: "aga-ws-probe-inspection-0001", ExpectedGenerationID: recommendation.GenerationID,
		RecommendationID: recommendation.RecommendationID, RecommendationDigest: recommendation.Digest, ExpectedRecommendationRevision: recommendation.Revision,
		InspectorBindingID: inspector.BindingID, InspectorBindingRevision: inspector.BindingRevision, LeadBindingID: lead.BindingID, LeadBindingRevision: lead.BindingRevision,
	})
	if err != nil {
		return fmt.Errorf("inspection creation failed: %w", err)
	}
	if inspectionResponse.Lifecycle == nil {
		return errors.New("inspection creation failed: lifecycle response missing")
	}
	lifecycle := *inspectionResponse.Lifecycle
	if len(lifecycle.Questions) == 0 {
		return errors.New("inspection question snapshot is empty")
	}
	questionKey := lifecycle.Questions[0].QuestionKey
	var lastResponse workspace.CommandResponse
	issue := func(principal identity.Principal, operation, key string, update func(*workspace.CommandEnvelope)) error {
		envelope := workspace.CommandEnvelope{OperationID: operation, IdempotencyKey: key, ExpectedGenerationID: lifecycle.GenerationID,
			ExpectedLifecycleRevision: lifecycle.Revision, ExpectedLifecycleDigest: lifecycle.Digest, InspectionID: lifecycle.InspectionID}
		if update != nil {
			update(&envelope)
		}
		var issueErr error
		lastResponse, issueErr = service.Command(ctx, principal, workspace.FamilyLifecycleCommand, envelope)
		if issueErr != nil {
			return fmt.Errorf("lifecycle command failed operation=%s: %w", operation, issueErr)
		}
		if lastResponse.Lifecycle == nil {
			return fmt.Errorf("lifecycle command failed operation=%s: lifecycle response missing", operation)
		}
		lifecycle = *lastResponse.Lifecycle
		return nil
	}
	if err := issue(accounts.inspector, workspace.OperationStartInspection, "aga-ws-probe-lifecycle-0001", nil); err != nil {
		return err
	}
	if err := issue(accounts.inspector, workspace.OperationRecordResponse, "aga-ws-probe-lifecycle-0002", func(command *workspace.CommandEnvelope) {
		command.TargetQuestionKey, command.Answer = questionKey, string(workspace.AnswerNonCompliant)
		command.CommentToAuditee = "Synthetic checklist response requires corrective action."
	}); err != nil {
		return err
	}
	if err := issue(accounts.inspector, workspace.OperationCreateFinding, "aga-ws-probe-lifecycle-0003", func(command *workspace.CommandEnvelope) {
		command.TargetQuestionKey, command.Answer = questionKey, string(workspace.AnswerNonCompliant)
		command.CommentToAuditee = "Synthetic potential finding created from the recorded response."
	}); err != nil {
		return err
	}
	if err := issue(accounts.inspector, workspace.OperationSubmitChecklist, "aga-ws-probe-lifecycle-0004", nil); err != nil {
		return err
	}
	potential := latestPendingPotential(lifecycle)
	if potential == nil {
		return errors.New("potential finding was not created")
	}
	if err := issue(accounts.lead, workspace.OperationConvertFinding, "aga-ws-probe-lifecycle-0005", func(command *workspace.CommandEnvelope) {
		command.PotentialFindingID, command.ReasonCode, command.Severity = potential.PotentialFindingID, "LEAD_REVIEW_CONVERT", "MAJOR"
		command.CapRequired, command.EvidenceRequired, command.DueDateRequired = true, true, false
	}); err != nil {
		return err
	}
	finding := latestFinding(lifecycle)
	if finding == nil {
		return errors.New("finding was not converted")
	}
	if err := issue(accounts.auditee, workspace.OperationSubmitCAP, "aga-ws-probe-lifecycle-0006", func(command *workspace.CommandEnvelope) {
		command.FindingID = finding.FindingID
		command.RootCause, command.CorrectiveAction, command.PreventiveAction, command.ResponsiblePerson = "Synthetic root cause.", "Synthetic corrective action.", "Synthetic preventive action.", "Synthetic accountable provider role"
		command.CommentToAuditee = "Synthetic CAP submitted for CAA review."
	}); err != nil {
		return err
	}
	if err := issue(accounts.lead, workspace.OperationReviewCAP, "aga-ws-probe-lifecycle-0007", func(command *workspace.CommandEnvelope) {
		command.FindingID, command.Outcome = finding.FindingID, "ACCEPT"
		command.CommentToAuditee, command.InternalCAANote = "CAP accepted after connected CAA review.", "Internal CAA review note remains private."
	}); err != nil {
		return err
	}
	if err := issue(accounts.auditee, workspace.OperationSubmitEvidence, "aga-ws-probe-lifecycle-0008", func(command *workspace.CommandEnvelope) {
		command.FindingID, command.EvidenceFileName, command.CommentToAuditee = finding.FindingID, "synthetic-evidence.pdf", "Synthetic evidence submitted for connected verification."
	}); err != nil {
		return err
	}
	staleRevision, staleDigest := lifecycle.Revision, lifecycle.Digest
	verifyEnvelope := workspace.CommandEnvelope{OperationID: workspace.OperationVerifyEvidence, IdempotencyKey: "aga-ws-probe-lifecycle-0009", ExpectedGenerationID: lifecycle.GenerationID,
		ExpectedLifecycleRevision: staleRevision, ExpectedLifecycleDigest: staleDigest, InspectionID: lifecycle.InspectionID, FindingID: finding.FindingID,
		Outcome: string(workspace.EvidenceClose), CommentToAuditee: "Evidence accepted and finding closed after verification.", InternalCAANote: "Internal verification note remains private."}
	lastResponse, err = service.Command(ctx, accounts.lead, workspace.FamilyLifecycleCommand, verifyEnvelope)
	if err != nil || lastResponse.Lifecycle == nil {
		return errors.New("evidence verification failed")
	}
	lifecycle = *lastResponse.Lifecycle
	replay, err := service.Command(ctx, accounts.lead, workspace.FamilyLifecycleCommand, verifyEnvelope)
	if err != nil || !replay.Replayed {
		return errors.New("lifecycle idempotency replay failed")
	}
	if _, err = service.Command(ctx, accounts.inspector, workspace.FamilyLifecycleCommand, workspace.CommandEnvelope{OperationID: workspace.OperationAuthorizedClose, IdempotencyKey: "aga-ws-probe-deny-role-0001", ExpectedGenerationID: lifecycle.GenerationID, ExpectedLifecycleRevision: lifecycle.Revision, ExpectedLifecycleDigest: lifecycle.Digest, InspectionID: lifecycle.InspectionID, FindingID: finding.FindingID, ReasonCode: "AUTHORIZED_CLOSE_NEGATIVE_TEST"}); !errors.Is(err, workspace.ErrNeutralDenied) {
		return errors.New("role denial was not enforced")
	}
	if _, err = service.Query(ctx, accounts.other, workspace.QueryRequest{OperationID: workspace.OperationGetInspection, InspectionID: lifecycle.InspectionID}); !errors.Is(err, workspace.ErrNeutralDenied) {
		return errors.New("organization denial was not enforced")
	}
	if _, err = service.Command(ctx, accounts.manager, workspace.FamilyLifecycleCommand, workspace.CommandEnvelope{OperationID: workspace.OperationAuthorizedClose, IdempotencyKey: "aga-ws-probe-deny-cas-0001", ExpectedGenerationID: lifecycle.GenerationID, ExpectedLifecycleRevision: staleRevision, ExpectedLifecycleDigest: staleDigest, InspectionID: lifecycle.InspectionID, FindingID: finding.FindingID, ReasonCode: "AUTHORIZED_CLOSE_CAS_NEGATIVE_TEST"}); !(errors.Is(err, preprod.ErrWorkspaceCAS) || errors.Is(err, workspace.ErrLifecycleConflict)) {
		return errors.New("stale lifecycle CAS was not rejected")
	}

	beforeReset, err := commandStore.Snapshot(ctx)
	if err != nil {
		return err
	}
	resetCommand := workspace.CommandEnvelope{OperationID: workspace.OperationResetGeneration, IdempotencyKey: "aga-ws-probe-reset-0001", ExpectedGenerationID: beforeReset.Generation.GenerationID, ExpectedGenerationRevision: beforeReset.Generation.Revision, ExpectedGenerationSealDigest: beforeReset.Generation.SealDigest, ReasonCode: "RESET_AFTER_TERMINAL_LIFECYCLE"}
	resetResponse, err := service.Command(ctx, accounts.admin, workspace.FamilyAdminCommand, resetCommand)
	if err != nil {
		return fmt.Errorf("terminal workspace reset failed: %w", err)
	}
	if resetResponse.Generation == nil {
		return errors.New("terminal workspace reset failed: generation response missing")
	}
	if resetResponse.Generation.State != preprod.GenerationActive {
		return fmt.Errorf("terminal workspace reset failed: unexpected state=%s", resetResponse.Generation.State)
	}
	resetReplay, err := service.Command(ctx, accounts.admin, workspace.FamilyAdminCommand, resetCommand)
	if err != nil || !resetReplay.Replayed {
		return errors.New("reset idempotency replay failed")
	}
	if _, err = service.Query(ctx, accounts.inspector, workspace.QueryRequest{OperationID: workspace.OperationGetInspection, InspectionID: lifecycle.InspectionID}); !errors.Is(err, workspace.ErrNeutralDenied) {
		return errors.New("old generation was not isolated after reset")
	}
	findingState, capState, evidenceState, closureBasis, separated := lifecycleTerminalFacts(lifecycle)
	if findingState != string(workspace.FindingClosed) || capState != string(workspace.CAPAccepted) || evidenceState != string(workspace.EvidenceAccepted) || closureBasis != "EVIDENCE_VERIFIED" || !separated {
		return errors.New("connected lifecycle did not reach the required terminal projection")
	}
	fmt.Printf(`{"schemaVersion":"aga-hybrid-connected-lifecycle-probe/v1","sourceKind":"connected-postgres","classificationCommandCount":%d,"lifecycleCommandCount":10,"findingState":"%s","capState":"%s","evidenceState":"%s","closureBasis":"%s","commentInternalSeparated":%t,"replayed":true,"casConflictRejected":true,"roleDenied":true,"organizationDenied":true,"resetSucceeded":true,"resetReplay":true,"oldGenerationDenied":true,"finalState":"%s"}
`, classificationCount, findingState, capState, evidenceState, closureBasis, separated, lifecycle.State)
	return nil
}

func recommendationRequest(command workspace.CommandEnvelope) aga.RecommendationRequest {
	return aga.RecommendationRequest{OperationID: command.OperationID, IdempotencyKey: command.IdempotencyKey, ExpectedGenerationID: command.ExpectedGenerationID, OrganizationID: command.OrganizationID, ProviderScopeRootID: command.ProviderScopeRootID, ProviderScopeID: command.ProviderScopeID, ProviderScopeVersion: command.ProviderScopeVersion, ProviderTypeID: command.ProviderTypeID, DepartmentID: command.DepartmentID, OrganizationalUnitID: command.OrganizationalUnitID, TargetID: command.TargetID, CanonicalTargetKind: command.CanonicalTargetKind, TargetProfileCode: command.TargetProfileCode, InspectionProfileCode: command.InspectionProfileCode, InspectionTypeCode: command.InspectionTypeCode, OperationQualifiers: command.OperationQualifiers, ActivityQualifiers: command.ActivityQualifiers, EffectiveAt: command.EffectiveAt, TaxonomyVersion: command.TaxonomyVersion, TaxonomyDigest: command.TaxonomyDigest, ClassificationRunID: command.ClassificationRunID, ClassificationRunDigest: command.ClassificationRunDigest, DraftID: command.DraftID, DraftRevision: command.DraftRevision, DraftContentDigest: command.DraftContentDigest, ExpectedDraftRevision: command.ExpectedDraftRevision, ReadinessEventID: command.ReadinessEventID, ReadinessEventDigest: command.ReadinessEventDigest}
}

func accountsFromFixture(fixture preprod.FixtureManifest) (probeAccounts, error) {
	bySlot := make(map[string]preprod.FixtureAccount, len(fixture.Accounts))
	for _, account := range fixture.Accounts {
		bySlot[account.Slot] = account
	}
	get := func(slot string) (preprod.FixtureAccount, error) {
		account, ok := bySlot[slot]
		if !ok || account.SubjectID == "" || account.OrganizationID == "" {
			return preprod.FixtureAccount{}, errors.New("fixture account missing")
		}
		return account, nil
	}
	admin, err := get("CAA_ADMIN")
	if err != nil {
		return probeAccounts{}, err
	}
	manager, err := get("DEPARTMENT_MANAGER")
	if err != nil {
		return probeAccounts{}, err
	}
	inspector, err := get("INSPECTOR")
	if err != nil {
		return probeAccounts{}, err
	}
	lead, err := get("LEAD_INSPECTOR")
	if err != nil {
		return probeAccounts{}, err
	}
	auditee, err := get("AUDITEE_MATCHING")
	if err != nil {
		return probeAccounts{}, err
	}
	other, err := get("AUDITEE_OTHER_ORGANIZATION")
	if err != nil {
		return probeAccounts{}, err
	}
	return probeAccounts{
		admin:     identity.Principal{SubjectID: admin.SubjectID, OrganizationID: admin.OrganizationID, Roles: []identity.Role{identity.RoleAdmin}},
		manager:   identity.Principal{SubjectID: manager.SubjectID, OrganizationID: manager.OrganizationID, Roles: []identity.Role{identity.RoleDepartmentManager}},
		inspector: identity.Principal{SubjectID: inspector.SubjectID, OrganizationID: inspector.OrganizationID, Roles: []identity.Role{identity.RoleInspector}},
		lead:      identity.Principal{SubjectID: lead.SubjectID, OrganizationID: lead.OrganizationID, Roles: []identity.Role{identity.RoleLeadInspector}},
		auditee:   identity.Principal{SubjectID: auditee.SubjectID, OrganizationID: auditee.OrganizationID, Roles: []identity.Role{identity.RoleAuditee}},
		other:     identity.Principal{SubjectID: other.SubjectID, OrganizationID: other.OrganizationID, Roles: []identity.Role{identity.RoleAuditee}},
	}, nil
}

func lifecycleFacts(ctx context.Context, loaded preprod.LoadedWorkspace, recommendation aga.Recommendation) (workspace.LifecycleBindingFact, workspace.LifecycleBindingFact, error) {
	resolver := workspace.NewFixtureLifecycleBindingResolver()
	inspectors, err := resolver(ctx, loaded, recommendation, "INSPECTOR")
	if err != nil || len(inspectors) != 1 {
		return workspace.LifecycleBindingFact{}, workspace.LifecycleBindingFact{}, errors.New("inspector binding missing")
	}
	leads, err := resolver(ctx, loaded, recommendation, "LEAD")
	if err != nil || len(leads) != 1 {
		return workspace.LifecycleBindingFact{}, workspace.LifecycleBindingFact{}, errors.New("lead binding missing")
	}
	return inspectors[0], leads[0], nil
}

func latestPendingPotential(projection workspace.LifecycleProjection) *workspace.LifecyclePotentialFinding {
	for index := len(projection.PotentialFindings) - 1; index >= 0; index-- {
		if projection.PotentialFindings[index].State == workspace.PotentialFindingPending {
			value := projection.PotentialFindings[index]
			return &value
		}
	}
	return nil
}

func latestFinding(projection workspace.LifecycleProjection) *workspace.LifecycleFinding {
	if len(projection.Findings) == 0 {
		return nil
	}
	value := projection.Findings[len(projection.Findings)-1]
	return &value
}

func lifecycleTerminalFacts(projection workspace.LifecycleProjection) (string, string, string, string, bool) {
	if len(projection.Findings) == 0 || len(projection.CAPRevisions) == 0 || len(projection.EvidenceVersions) == 0 {
		return "", "", "", "", false
	}
	finding := projection.Findings[len(projection.Findings)-1]
	cap := projection.CAPRevisions[len(projection.CAPRevisions)-1]
	evidence := projection.EvidenceVersions[len(projection.EvidenceVersions)-1]
	separated := true
	for _, value := range projection.CAPRevisions {
		separated = separated && value.CommentToAuditee != "" && value.InternalCAANote == ""
	}
	for _, value := range projection.EvidenceVersions {
		separated = separated && value.CommentToAuditee != "" && value.InternalCAANote == ""
	}
	for _, value := range projection.VerificationDecisions {
		separated = separated && value.CommentToAuditee != "" && value.InternalCAANote == ""
	}
	return string(finding.State), string(cap.State), string(evidence.ReviewState), finding.ClosureBasis, separated
}
