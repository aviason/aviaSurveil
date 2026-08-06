//go:build preproddemo

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
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
	overlayURL := os.Getenv("AVIA_AGA_DEMO_DATABASE_URL")
	if readerURL == "" || commandURL == "" || overlayURL == "" {
		return errors.New("workspace and sealed overlay database urls are required")
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
	overlayPool, err := database.Open(ctx, overlayURL)
	if err != nil {
		return err
	}
	defer overlayPool.Close()
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
	questionBodies, err := workspace.NewPostgresQuestionBodyResolver(overlayPool)
	if err != nil {
		return err
	}
	scopeResolver := workspace.NewPostgresRecommendationScopeResolver(readerPool)
	service := workspace.NewService(workspace.ServiceConfig{
		ReaderStore: readerStore, CommandStore: commandStore, Resolver: bindingResolver,
		QuestionBodies: questionBodies, QuestionTextSearch: questionBodies,
		RecommendationScopes: scopeResolver, SimulationSetup: workspace.NewPostgresSimulationSetupResolver(readerPool), LifecycleBindings: workspace.NewFixtureLifecycleBindingResolver(),
	})

	current, err := commandStore.Snapshot(ctx)
	if err != nil {
		return err
	}
	accounts, err := accountsFromFixture(current.Fixture)
	if err != nil {
		return err
	}
	switch os.Getenv("AVIA_AGA_DEMO_PROBE_PHASE") {
	case "setup-only":
		return runManagerSetupOnly(ctx, service, accounts.manager)
	case "finalize-only":
		return runManagerFinalizeOnly(ctx, service, commandStore, accounts)
	}
	setup, err := probeSimulationSetup(ctx, service, accounts.manager)
	if err != nil {
		return fmt.Errorf("initial simulation setup failed: %w", err)
	}
	selected, domains, err := probeEligibleSubset(current, setup, 3)
	if err != nil {
		return err
	}
	classificationCount := 0
	sequence := 0
	for _, domainCode := range domains {
		for _, disposition := range []string{"UNSET", string(aga.DispositionInclude)} {
			filter := workspace.BatchFilter{DomainCode: domainCode, Disposition: disposition}
			changed, commandErr := probeBatch(ctx, service, accounts.manager, filter, workspace.BatchExclude, probeReason, &sequence)
			if commandErr != nil {
				return commandErr
			}
			if changed {
				classificationCount++
			}
		}
	}
	for _, item := range selected {
		setup, err = probeSimulationSetup(ctx, service, accounts.manager)
		if err != nil {
			return err
		}
		reason := probeReason
		if item.Governance.QuestionSourceProposalGap {
			reason = probeReadiness
		}
		response, commandErr := service.Command(ctx, accounts.manager, workspace.FamilyClassificationCommand, workspace.CommandEnvelope{
			OperationID: workspace.OperationInclude, IdempotencyKey: fmt.Sprintf("aga-ws-probe-selection-%06d", sequence+1),
			ExpectedGenerationID: setup.GenerationID, ExpectedDraftRevision: setup.DraftRevision, ExpectedDraftContentDigest: setup.DraftContentDigest,
			TargetQuestionKey: aga.BaseQuestionReference(item.Identity).Key(), ReasonCode: reason, Action: aga.DraftInclude,
		})
		if commandErr != nil || response.Draft == nil {
			return fmt.Errorf("deterministic subset include failed: %w", commandErr)
		}
		sequence++
		classificationCount++
	}
	setup, err = probeSimulationSetup(ctx, service, accounts.manager)
	if err != nil {
		return err
	}
	readyResponse, err := service.Command(ctx, accounts.manager, workspace.FamilyClassificationCommand, workspace.CommandEnvelope{
		OperationID: workspace.OperationMarkReady, IdempotencyKey: "aga-ws-probe-readiness-0001", ExpectedGenerationID: setup.GenerationID,
		ExpectedDraftRevision: setup.DraftRevision, ExpectedDraftContentDigest: setup.DraftContentDigest, Action: aga.DraftMarkReady, ReasonCode: probeReadiness,
		SetupDigest: setup.SimulationSetupDigest,
	})
	if err != nil {
		return fmt.Errorf("readiness command failed: %w (revision=%d contentDigestPresent=%t)", err, setup.DraftRevision, setup.DraftContentDigest != "")
	}
	if readyResponse.Draft == nil {
		return errors.New("readiness command returned no draft")
	}
	setup, err = probeSimulationSetup(ctx, service, accounts.manager)
	if err != nil {
		return err
	}
	recommendationResponse, err := service.Command(ctx, accounts.manager, workspace.FamilyRecommendationCommand, workspace.CommandEnvelope{
		OperationID: workspace.OperationCreateRecommendation, IdempotencyKey: "aga-ws-probe-recommendation-0001", ExpectedGenerationID: setup.GenerationID,
		DraftID: setup.DraftID, DraftRevision: setup.DraftRevision, ExpectedDraftRevision: setup.DraftRevision, DraftContentDigest: setup.DraftContentDigest,
		SetupDigest: setup.SimulationSetupDigest,
	})
	if err != nil {
		return fmt.Errorf("recommendation command failed: %w", err)
	}
	if recommendationResponse.Recommendation == nil {
		return errors.New("recommendation command failed")
	}
	if _, err = service.Query(ctx, accounts.manager, workspace.QueryRequest{OperationID: workspace.OperationGetCurrentRecommendation}); err != nil {
		return fmt.Errorf("current recommendation reload failed: %w", err)
	}
	setup, err = probeSimulationSetup(ctx, service, accounts.manager)
	if err != nil {
		return err
	}
	if len(setup.InspectorChoices) != 1 || len(setup.LeadChoices) != 1 {
		return errors.New("simulation role choices are not unique")
	}
	inspectionResponse, err := service.Command(ctx, accounts.manager, workspace.FamilyRecommendationCommand, workspace.CommandEnvelope{
		OperationID: workspace.OperationCreateInspection, IdempotencyKey: "aga-ws-probe-inspection-0001", ExpectedGenerationID: setup.GenerationID,
		SetupDigest: setup.SimulationSetupDigest, InspectorSelectionPin: setup.InspectorChoices[0].SelectionPin, LeadSelectionPin: setup.LeadChoices[0].SelectionPin,
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

func runManagerSetupOnly(ctx context.Context, service *workspace.Service, manager identity.Principal) error {
	setup, err := probeSimulationSetup(ctx, service, manager)
	if err != nil {
		return fmt.Errorf("manager setup-only read failed: %w", err)
	}
	seen := make(map[string]struct{})
	pages, itemCount := 0, 0
	for page := 0; ; page++ {
		response, queryErr := service.Query(ctx, manager, workspace.QueryRequest{OperationID: workspace.OperationSearchItems, Page: page, PageSize: workspace.MaxQuestionTextPage})
		if queryErr != nil {
			return fmt.Errorf("manager inventory page %d failed: %w", page, queryErr)
		}
		if len(response.Items) > workspace.MaxQuestionTextPage {
			return errors.New("manager inventory page exceeded the bounded text limit")
		}
		pages++
		itemCount += len(response.Items)
		for _, item := range response.Items {
			key := aga.BaseQuestionReference(item.Identity).Key()
			if _, exists := seen[key]; exists {
				return errors.New("manager inventory pagination returned a duplicate identity")
			}
			seen[key] = struct{}{}
			if item.QuestionText == nil || item.QuestionTextDigest == nil || *item.QuestionText == "" || *item.QuestionTextDigest == "" || *item.QuestionTextDigest != item.Identity.TextDigest {
				return errors.New("manager inventory pagination returned an incomplete or mismatched text projection")
			}
		}
		if response.NextPage == nil {
			break
		}
	}
	if itemCount != 1310 || len(seen) != 1310 {
		return fmt.Errorf("manager inventory reachability count=%d unique=%d want 1310", itemCount, len(seen))
	}
	if _, queryErr := service.Query(ctx, manager, workspace.QueryRequest{OperationID: workspace.OperationGetCurrentRecommendation}); !errors.Is(queryErr, workspace.ErrNeutralDenied) {
		return errors.New("setup-only generation unexpectedly has a current recommendation")
	}
	if _, queryErr := service.Query(ctx, manager, workspace.QueryRequest{OperationID: workspace.OperationGetCurrentInspection}); !errors.Is(queryErr, workspace.ErrNeutralDenied) {
		return errors.New("setup-only generation unexpectedly has a current inspection")
	}
	fmt.Printf("{\"schemaVersion\":\"aga-manager-package-setup/v1\",\"sourceKind\":\"connected-postgres\",\"inventoryCount\":%d,\"inventoryPageCount\":%d,\"uniqueInventoryCount\":%d,\"boundedBodyProjection\":true,\"bodyDigestProjection\":true,\"currentRecommendationAbsent\":true,\"currentInspectionAbsent\":true,\"readinessState\":\"%s\",\"draftRevision\":%d}\n", itemCount, pages, len(seen), setup.ReadinessState, setup.DraftRevision)
	return nil
}

func runManagerFinalizeOnly(ctx context.Context, service *workspace.Service, commandStore *preprod.PostgresStore, accounts probeAccounts) error {
	recommendationResponse, err := service.Query(ctx, accounts.manager, workspace.QueryRequest{OperationID: workspace.OperationGetCurrentRecommendation})
	if err != nil || recommendationResponse.RecommendationSnapshot == nil {
		return errors.New("manager finalizer could not reload the current recommendation")
	}
	recommendationID := recommendationResponse.RecommendationSnapshot.Recommendation.RecommendationID
	recommendationReload, err := service.Query(ctx, accounts.manager, workspace.QueryRequest{OperationID: workspace.OperationGetCurrentRecommendation})
	if err != nil || recommendationReload.RecommendationSnapshot == nil || recommendationReload.RecommendationSnapshot.Recommendation.RecommendationID != recommendationID {
		return errors.New("current recommendation reload was not stable")
	}
	inspectionResponse, err := service.Query(ctx, accounts.manager, workspace.QueryRequest{OperationID: workspace.OperationGetCurrentInspection})
	if err != nil || inspectionResponse.CurrentInspection == nil {
		return errors.New("manager finalizer could not reload the current inspection")
	}
	lifecycle := *inspectionResponse.CurrentInspection
	inspectionReload, err := service.Query(ctx, accounts.manager, workspace.QueryRequest{OperationID: workspace.OperationGetCurrentInspection})
	if err != nil || inspectionReload.CurrentInspection == nil || inspectionReload.CurrentInspection.InspectionID != lifecycle.InspectionID {
		return errors.New("current inspection reload was not stable")
	}
	findingState, capState, evidenceState, closureBasis, separated := lifecycleTerminalFacts(lifecycle)
	if findingState != string(workspace.FindingClosed) || capState != string(workspace.CAPAccepted) || evidenceState != string(workspace.EvidenceAccepted) || closureBasis != "EVIDENCE_VERIFIED" || !separated {
		return errors.New("manager browser lifecycle did not reach the required terminal projection")
	}
	if lifecycle.Revision == 0 {
		return errors.New("manager browser lifecycle revision is not usable for CAS verification")
	}
	_, casErr := service.Command(ctx, accounts.manager, workspace.FamilyLifecycleCommand, workspace.CommandEnvelope{
		OperationID: workspace.OperationAuthorizedClose, IdempotencyKey: "aga-ws-manager-demo-deny-cas-0001", ExpectedGenerationID: lifecycle.GenerationID,
		ExpectedLifecycleRevision: lifecycle.Revision - 1, ExpectedLifecycleDigest: lifecycle.Digest, InspectionID: lifecycle.InspectionID,
		FindingID: lifecycle.Findings[len(lifecycle.Findings)-1].FindingID, ReasonCode: "MANAGER_DEMO_CAS_NEGATIVE_TEST",
	})
	if !(errors.Is(casErr, preprod.ErrWorkspaceCAS) || errors.Is(casErr, workspace.ErrLifecycleConflict)) {
		return errors.New("manager demo stale lifecycle CAS was not rejected")
	}
	_, roleErr := service.Command(ctx, accounts.inspector, workspace.FamilyLifecycleCommand, workspace.CommandEnvelope{
		OperationID: workspace.OperationAuthorizedClose, IdempotencyKey: "aga-ws-manager-demo-deny-role-0001", ExpectedGenerationID: lifecycle.GenerationID,
		ExpectedLifecycleRevision: lifecycle.Revision, ExpectedLifecycleDigest: lifecycle.Digest, InspectionID: lifecycle.InspectionID,
		FindingID: lifecycle.Findings[len(lifecycle.Findings)-1].FindingID, ReasonCode: "MANAGER_DEMO_ROLE_NEGATIVE_TEST",
	})
	if !errors.Is(roleErr, workspace.ErrNeutralDenied) {
		return errors.New("manager demo role denial was not enforced")
	}
	if _, organizationErr := service.Query(ctx, accounts.other, workspace.QueryRequest{OperationID: workspace.OperationGetInspection, InspectionID: lifecycle.InspectionID}); !errors.Is(organizationErr, workspace.ErrNeutralDenied) {
		return errors.New("manager demo organization denial was not enforced")
	}
	beforeReset, err := commandStore.Snapshot(ctx)
	if err != nil {
		return err
	}
	resetCommand := workspace.CommandEnvelope{OperationID: workspace.OperationResetGeneration, IdempotencyKey: "aga-ws-manager-demo-reset-0001", ExpectedGenerationID: beforeReset.Generation.GenerationID, ExpectedGenerationRevision: beforeReset.Generation.Revision, ExpectedGenerationSealDigest: beforeReset.Generation.SealDigest, ReasonCode: "RESET_AFTER_MANAGER_DEMO"}
	resetResponse, err := service.Command(ctx, accounts.admin, workspace.FamilyAdminCommand, resetCommand)
	if err != nil || resetResponse.Generation == nil || resetResponse.Generation.State != preprod.GenerationActive {
		return fmt.Errorf("manager demo reset failed: %w", err)
	}
	resetReplay, err := service.Command(ctx, accounts.admin, workspace.FamilyAdminCommand, resetCommand)
	if err != nil || !resetReplay.Replayed {
		return errors.New("manager demo reset replay failed")
	}
	if _, err = service.Query(ctx, accounts.manager, workspace.QueryRequest{OperationID: workspace.OperationGetCurrentInspection}); !errors.Is(err, workspace.ErrNeutralDenied) {
		return errors.New("manager demo old inspection remained current after reset")
	}
	fmt.Printf("{\"schemaVersion\":\"aga-manager-multi-role-finalizer/v1\",\"sourceKind\":\"connected-postgres\",\"lifecycleCommandCount\":10,\"lifecycleReplayVerified\":%t,\"casConflictRejected\":true,\"roleDenied\":true,\"organizationDenied\":true,\"recommendationReloadVerified\":true,\"inspectionReloadVerified\":true,\"findingState\":\"%s\",\"capState\":\"%s\",\"evidenceState\":\"%s\",\"closureBasis\":\"%s\",\"commentInternalSeparated\":%t,\"resetSucceeded\":true,\"resetReplay\":true,\"oldInspectionDenied\":true,\"finalState\":\"%s\"}\n", resetReplay.Replayed, findingState, capState, evidenceState, closureBasis, separated, lifecycle.State)
	return nil
}

func probeSimulationSetup(ctx context.Context, service *workspace.Service, manager identity.Principal) (workspace.SimulationSetupProjection, error) {
	response, err := service.Query(ctx, manager, workspace.QueryRequest{OperationID: workspace.OperationGetSimulationSetup})
	if err != nil {
		return workspace.SimulationSetupProjection{}, err
	}
	if response.SimulationSetup == nil || response.SimulationSetup.SimulationSetupDigest == "" || response.SimulationSetup.ReadinessEventID != "" {
		return workspace.SimulationSetupProjection{}, errors.New("simulation setup is missing or stateful")
	}
	return *response.SimulationSetup, nil
}

func probeEligibleSubset(loaded preprod.LoadedWorkspace, setup workspace.SimulationSetupProjection, wanted int) ([]workspace.ClassificationReviewItem, []string, error) {
	selected := make([]workspace.ClassificationReviewItem, 0, wanted)
	domainCounts := make(map[string]int)
	for _, item := range loaded.Items {
		domainCode := item.Projection.MainDomainCode
		if domainCode == "" {
			return nil, nil, errors.New("deterministic batch domain is missing")
		}
		domainCounts[domainCode]++
		reviewItem := workspace.ClassificationReviewItem{ClassificationItem: item, QuestionRef: aga.BaseQuestionReference(item.Identity), QuestionOrigin: "SEALED_BASE"}
		if len(selected) < wanted && probeItemEligible(reviewItem, setup) {
			selected = append(selected, reviewItem)
		}
	}
	if len(selected) != wanted {
		return nil, nil, fmt.Errorf("deterministic eligible subset has %d rows, want %d", len(selected), wanted)
	}
	domains := make([]string, 0, len(domainCounts))
	for domainCode, count := range domainCounts {
		if count > workspace.MaxBatchPreviewSize {
			return nil, nil, fmt.Errorf("deterministic batch domain %q has %d rows, exceeds %d", domainCode, count, workspace.MaxBatchPreviewSize)
		}
		domains = append(domains, domainCode)
	}
	sort.Strings(domains)
	return selected, domains, nil
}

func probeItemEligible(item workspace.ClassificationReviewItem, setup workspace.SimulationSetupProjection) bool {
	projection := item.Projection
	if projection.CanonicalTargetKind != setup.CanonicalTargetKind || projection.TargetProfileCode != setup.TargetProfileCode || !probeContains(projection.InspectionProfileCodes, setup.InspectionProfileCode) || !probeContains(projection.InspectionTypeCodes, setup.InspectionTypeCode) || !probeQualifiersEqual(projection.OperationQualifiers, setup.OperationQualifiers) || !probeQualifiersEqual(projection.ActivityQualifiers, setup.ActivityQualifiers) {
		return false
	}
	switch projection.ApplicabilityDisposition {
	case "APPLICABLE", "CONDITIONAL_ON_CONFIGURATION", "CONDITIONAL_ON_FACILITY", "CONDITIONAL_ON_OPERATION":
		return true
	default:
		return false
	}
}

func probeContains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func probeQualifiersEqual(left, right []aga.Qualifier) bool {
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

func probeBatch(ctx context.Context, service *workspace.Service, manager identity.Principal, filter workspace.BatchFilter, action workspace.BatchAction, reason string, sequence *int) (bool, error) {
	setup, err := probeSimulationSetup(ctx, service, manager)
	if err != nil {
		return false, fmt.Errorf("batch setup failed: %w", err)
	}
	*sequence++
	previewKey := fmt.Sprintf("aga-ws-probe-batch-preview-%04d", *sequence)
	previewResponse, err := service.Command(ctx, manager, workspace.FamilyClassificationCommand, workspace.CommandEnvelope{
		OperationID: workspace.OperationPreviewBatch, IdempotencyKey: previewKey, ExpectedGenerationID: setup.GenerationID,
		ExpectedDraftRevision: setup.DraftRevision, ExpectedDraftContentDigest: setup.DraftContentDigest,
		BatchFilter: &filter, BatchAction: action, ReasonCode: reason, SetupDigest: setup.SimulationSetupDigest,
	})
	if err != nil {
		return false, fmt.Errorf("batch preview failed for %s/%s: %w", filter.FormCode, filter.Disposition, err)
	}
	if previewResponse.BatchPreview == nil {
		return false, errors.New("batch preview response missing")
	}
	preview := previewResponse.BatchPreview
	if preview.Count == 0 {
		return false, nil
	}
	if preview.Count > workspace.MaxBatchPreviewSize {
		return false, workspace.ErrBatchPreviewTooLarge
	}
	*sequence++
	executeKey := fmt.Sprintf("aga-ws-probe-batch-execute-%04d", *sequence)
	executeResponse, err := service.Command(ctx, manager, workspace.FamilyClassificationCommand, workspace.CommandEnvelope{
		OperationID: workspace.OperationExecuteBatch, IdempotencyKey: executeKey, ExpectedGenerationID: setup.GenerationID,
		ExpectedDraftRevision: preview.DraftRevision, ExpectedDraftContentDigest: preview.DraftContentDigest,
		PreviewID: preview.PreviewID, PreviewDigest: preview.PreviewDigest, BatchFilter: &filter, BatchAction: action, ReasonCode: reason, SetupDigest: setup.SimulationSetupDigest,
	})
	if err != nil {
		return false, fmt.Errorf("batch execution failed for %s/%s: %w", filter.FormCode, filter.Disposition, err)
	}
	if executeResponse.BatchPreview == nil || executeResponse.BatchPreview.ConsumedAt == nil {
		return false, errors.New("batch execution did not consume its server preview")
	}
	return true, nil
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
