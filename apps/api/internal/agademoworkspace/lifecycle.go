package agademoworkspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	aga "github.com/MarlonJD/aviaSurveil360/apps/api/internal/agaapplicability"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
	preprod "github.com/MarlonJD/aviaSurveil360/apps/api/internal/preproddata/agademoworkspace"
)

func lifecycleIDFor(generationID, recommendationID string) string {
	hash := sha256.Sum256([]byte("AGA-DEMO-INSPECTION-ID-V1\n" + generationID + "\x00" + recommendationID))
	return "aga-ws-inspection-" + hex.EncodeToString(hash[:8])
}

func lifecycleObjectID(prefix, inspectionID, value string, revision int) string {
	hash := sha256.Sum256([]byte("AGA-DEMO-LIFECYCLE-ID-V1\n" + prefix + "\x00" + inspectionID + "\x00" + value + "\x00" + fmt.Sprint(revision)))
	return "aga-ws-" + strings.ToLower(prefix) + "-" + hex.EncodeToString(hash[:8])
}

func lifecycleDigest(aggregate LifecycleAggregate) string {
	digest, _ := aga.DigestExcludingJSONFields("AGA-DEMO-LIFECYCLE-AGGREGATE-V1", aggregate, "digest")
	return digest
}

func (aggregate *LifecycleAggregate) advance(now time.Time) error {
	if aggregate == nil {
		return ErrLifecycleConflict
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	aggregate.Revision++
	aggregate.UpdatedAt = now.UTC()
	aggregate.Digest = lifecycleDigest(*aggregate)
	return aggregate.Validate()
}

func lifecycleEventFor(aggregate LifecycleAggregate, operationID, commandKey, actor string, previousDigest string, now time.Time) (preprod.LifecycleEvent, error) {
	payload, err := jsonMarshal(aggregate)
	if err != nil {
		return preprod.LifecycleEvent{}, err
	}
	event := preprod.LifecycleEvent{
		EventID: lifecycleObjectID("event", aggregate.InspectionID, commandKey, aggregate.Revision), LifecycleID: aggregate.InspectionID,
		Sequence: aggregate.Revision, OperationID: operationID, CommandKey: commandKey, EventType: operationID,
		Payload: payload, ActorSubjectID: actor, CreatedAt: aggregate.UpdatedAt, PreviousDigest: previousDigest,
	}
	event.EventDigest, err = aga.DigestExcludingJSONFields("AGA-DEMO-LIFECYCLE-EVENT-V1", event, "eventDigest")
	if err != nil {
		return preprod.LifecycleEvent{}, err
	}
	return event, nil
}

func jsonMarshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func buildInspectionFromRecommendation(snapshot preprod.RecommendationSnapshot, command CommandEnvelope, inspector, lead LifecycleBindingFact, now time.Time) (LifecycleAggregate, error) {
	if err := preprod.ValidateRecommendationSnapshot(snapshot); err != nil {
		return LifecycleAggregate{}, fmt.Errorf("%w: stored recommendation validation failed: %v", ErrLifecycleRecommendationStale, err)
	}
	recommendation := snapshot.Recommendation
	if command.OperationID != OperationCreateInspection || command.ExpectedGenerationID != recommendation.GenerationID || command.RecommendationID != recommendation.RecommendationID || command.RecommendationDigest != recommendation.Digest || command.ExpectedRecommendationRevision != recommendation.Revision {
		return LifecycleAggregate{}, fmt.Errorf("%w: inspection command recommendation pin mismatch", ErrLifecycleRecommendationStale)
	}
	if !bindingMatches(inspector, command.InspectorBindingID, command.InspectorBindingRevision, recommendation) || !bindingMatches(lead, command.LeadBindingID, command.LeadBindingRevision, recommendation) {
		return LifecycleAggregate{}, fmt.Errorf("%w: inspector or lead binding pin mismatch", ErrLifecycleBindingMismatch)
	}
	if inspector.SubjectID == lead.SubjectID || inspector.SubjectID == "" || lead.SubjectID == "" {
		return LifecycleAggregate{}, ErrLifecycleBindingMismatch
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	aggregate := LifecycleAggregate{
		InspectionID: lifecycleIDFor(recommendation.GenerationID, recommendation.RecommendationID), GenerationID: recommendation.GenerationID,
		RecommendationID: recommendation.RecommendationID, RecommendationDigest: recommendation.Digest, OrganizationID: recommendation.OrganizationID,
		ProviderScopeID: recommendation.ProviderScopeID, ProviderScopeVersion: recommendation.ProviderScopeVersion, State: InspectionReady, Revision: 1,
		Inspector: bindingPin(inspector), Lead: bindingPin(lead), CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
		Questions: make([]LifecycleQuestionSnapshot, 0, len(recommendation.Items)), Responses: []LifecycleResponse{}, PotentialFindings: []LifecyclePotentialFinding{},
		Findings: []LifecycleFinding{}, CAPRevisions: []LifecycleCAPRevision{}, EvidenceVersions: []LifecycleEvidenceVersion{}, VerificationDecisions: []LifecycleVerificationDecision{},
	}
	for _, item := range recommendation.Items {
		aggregate.Questions = append(aggregate.Questions, LifecycleQuestionSnapshot{QuestionKey: item.QuestionRef.Key(), QuestionRef: item.QuestionRef, RootSequence: item.RootSequence, Projection: item.Projection})
	}
	aggregate.Digest = lifecycleDigest(aggregate)
	if err := aggregate.Validate(); err != nil {
		return LifecycleAggregate{}, err
	}
	return aggregate, nil
}

func bindingMatches(binding LifecycleBindingFact, bindingID string, revision int, recommendation aga.Recommendation) bool {
	return binding.Active && binding.BindingID != "" && binding.BindingID == bindingID && binding.BindingRevision == revision && binding.SubjectID != "" && binding.OrganizationID == recommendation.OrganizationID && binding.ProviderScopeID == recommendation.ProviderScopeID && binding.DepartmentID == recommendation.DepartmentID && binding.OrganizationalUnitID == recommendation.OrganizationalUnitID
}

func bindingPin(binding LifecycleBindingFact) LifecycleBindingPin {
	sourceOrganizationID := binding.SourceOrganizationID
	if sourceOrganizationID == "" {
		sourceOrganizationID = binding.OrganizationID
	}
	return LifecycleBindingPin{BindingID: binding.BindingID, BindingRevision: binding.BindingRevision, SubjectID: binding.SubjectID, MembershipSlot: binding.MembershipSlot, OrganizationID: binding.OrganizationID, SourceOrganizationID: sourceOrganizationID, DepartmentID: binding.DepartmentID, OrganizationalUnitID: binding.OrganizationalUnitID}
}

func (service *Service) createInspection(ctx context.Context, principal identity.Principal, workspace preprod.LoadedWorkspace, command CommandEnvelope) (LifecycleAggregate, error) {
	recommendationStore, ok := service.command.(preprod.RecommendationSnapshotStore)
	if !ok || service.lifecycleBindings == nil {
		return LifecycleAggregate{}, ErrLifecycleRecommendationStale
	}
	lifecycleStore, ok := service.command.(preprod.LifecycleStore)
	if !ok {
		return LifecycleAggregate{}, ErrCapabilityUnavailable
	}
	if currentStore, currentOK := service.command.(CurrentLifecycleStore); currentOK {
		current, currentErr := currentStore.ListLifecycleStreams(ctx, workspace.Generation.GenerationID)
		if currentErr != nil {
			return LifecycleAggregate{}, ErrCapabilityUnavailable
		}
		if len(current) > 0 {
			if len(current) > 1 {
				return LifecycleAggregate{}, ErrCurrentObjectAmbiguous
			}
			return LifecycleAggregate{}, ErrCommandConflict
		}
	}
	snapshot, found, err := recommendationStore.GetRecommendationSnapshot(ctx, command.ExpectedGenerationID, command.RecommendationID)
	if err != nil {
		return LifecycleAggregate{}, fmt.Errorf("%w: recommendation snapshot lookup failed: %v", ErrLifecycleRecommendationStale, err)
	}
	if !found {
		return LifecycleAggregate{}, fmt.Errorf("%w: recommendation snapshot was not found", ErrLifecycleRecommendationStale)
	}
	if snapshot.Recommendation.RecommendationID != command.RecommendationID || snapshot.Recommendation.Digest != command.RecommendationDigest || snapshot.Recommendation.Revision != command.ExpectedRecommendationRevision {
		return LifecycleAggregate{}, fmt.Errorf("%w: recommendation command pin mismatch", ErrLifecycleRecommendationStale)
	}
	inspectorFacts, err := service.lifecycleBindings(ctx, workspace, snapshot.Recommendation, "INSPECTOR")
	if err != nil {
		return LifecycleAggregate{}, fmt.Errorf("%w: inspector binding resolver failed: %v", ErrLifecycleBindingMismatch, err)
	}
	if len(inspectorFacts) != 1 {
		return LifecycleAggregate{}, fmt.Errorf("%w: inspector binding cardinality mismatch", ErrLifecycleBindingMismatch)
	}
	leadFacts, err := service.lifecycleBindings(ctx, workspace, snapshot.Recommendation, "LEAD")
	if err != nil {
		return LifecycleAggregate{}, fmt.Errorf("%w: lead binding resolver failed: %v", ErrLifecycleBindingMismatch, err)
	}
	if len(leadFacts) != 1 {
		return LifecycleAggregate{}, fmt.Errorf("%w: lead binding cardinality mismatch", ErrLifecycleBindingMismatch)
	}
	auditeeFacts, err := service.lifecycleBindings(ctx, workspace, snapshot.Recommendation, "AUDITEE")
	if err != nil {
		return LifecycleAggregate{}, fmt.Errorf("%w: auditee binding resolver failed: %v", ErrLifecycleBindingMismatch, err)
	}
	if len(auditeeFacts) != 1 {
		return LifecycleAggregate{}, fmt.Errorf("%w: auditee binding cardinality mismatch", ErrLifecycleBindingMismatch)
	}
	aggregate, err := buildInspectionFromRecommendation(snapshot, command, inspectorFacts[0], leadFacts[0], service.clock())
	if err != nil {
		return LifecycleAggregate{}, fmt.Errorf("%w: build inspection aggregate failed: %v", ErrLifecycleRecommendationStale, err)
	}
	if !bindingMatches(auditeeFacts[0], auditeeFacts[0].BindingID, auditeeFacts[0].BindingRevision, snapshot.Recommendation) {
		return LifecycleAggregate{}, fmt.Errorf("%w: auditee binding pin mismatch", ErrLifecycleBindingMismatch)
	}
	aggregate.Auditee = bindingPin(auditeeFacts[0])
	aggregate.Digest = lifecycleDigest(aggregate)
	if aggregate.Auditee.BindingID == "" || aggregate.Auditee.SubjectID == "" {
		return LifecycleAggregate{}, ErrLifecycleBindingMismatch
	}
	if err := aggregate.Validate(); err != nil {
		return LifecycleAggregate{}, err
	}
	event, err := lifecycleEventFor(aggregate, command.OperationID, command.IdempotencyKey, principal.SubjectID, "", aggregate.CreatedAt)
	if err != nil {
		return LifecycleAggregate{}, err
	}
	if _, err := lifecycleStore.AppendLifecycleEvent(ctx, event); err != nil {
		return LifecycleAggregate{}, fmt.Errorf("append inspection event: %w", err)
	}
	return aggregate, nil
}

func (service *Service) loadLifecycle(ctx context.Context, generationID, inspectionID string) (LifecycleAggregate, []preprod.LifecycleEvent, error) {
	store, ok := service.command.(preprod.LifecycleStore)
	if !ok {
		return LifecycleAggregate{}, nil, ErrCapabilityUnavailable
	}
	return loadLifecycleFromStore(ctx, store, generationID, inspectionID)
}

func loadLifecycleFromStore(ctx context.Context, store preprod.LifecycleStore, generationID, inspectionID string) (LifecycleAggregate, []preprod.LifecycleEvent, error) {
	if store == nil {
		return LifecycleAggregate{}, nil, ErrCapabilityUnavailable
	}
	events, err := store.GetLifecycleEvents(ctx, generationID, inspectionID)
	if err != nil {
		return LifecycleAggregate{}, nil, err
	}
	if len(events) == 0 {
		return LifecycleAggregate{}, nil, ErrLifecycleNotFound
	}
	var aggregate LifecycleAggregate
	if err := json.Unmarshal(events[len(events)-1].Payload, &aggregate); err != nil {
		return LifecycleAggregate{}, nil, ErrLifecycleNotFound
	}
	if aggregate.GenerationID != generationID || aggregate.InspectionID != inspectionID || aggregate.Digest != lifecycleDigest(aggregate) {
		return LifecycleAggregate{}, nil, ErrLifecycleConflict
	}
	if err := aggregate.Validate(); err != nil {
		return LifecycleAggregate{}, nil, err
	}
	return aggregate, events, nil
}

func (service *Service) appendLifecycle(ctx context.Context, aggregate LifecycleAggregate, events []preprod.LifecycleEvent, command CommandEnvelope, principal identity.Principal) error {
	store, ok := service.command.(preprod.LifecycleStore)
	if !ok {
		return ErrCapabilityUnavailable
	}
	previous := ""
	if len(events) > 0 {
		previous = events[len(events)-1].EventDigest
	}
	event, err := lifecycleEventFor(aggregate, command.OperationID, command.IdempotencyKey, principal.SubjectID, previous, aggregate.UpdatedAt)
	if err != nil {
		return err
	}
	_, err = store.AppendLifecycleEvent(ctx, event)
	return err
}

func applyLifecycleCommand(aggregate *LifecycleAggregate, command CommandEnvelope, principal identity.Principal, now time.Time) error {
	if aggregate == nil || command.ExpectedLifecycleRevision != aggregate.Revision || command.ExpectedLifecycleDigest != aggregate.Digest {
		return ErrLifecycleConflict
	}
	if !principalMayTouchAggregate(*aggregate, principal, command.OperationID) {
		return ErrNeutralDenied
	}
	switch command.OperationID {
	case OperationStartInspection:
		if aggregate.State != InspectionReady || principal.SubjectID != aggregate.Inspector.SubjectID {
			return ErrLifecycleTransition
		}
		aggregate.State = InspectionInProgress
	case OperationRecordResponse:
		if aggregate.State != InspectionInProgress || principal.SubjectID != aggregate.Inspector.SubjectID || !aggregate.questionExists(command.TargetQuestionKey) || !validChecklistAnswer(ChecklistAnswer(command.Answer)) {
			return ErrLifecycleTransition
		}
		previous, _ := aggregate.latestResponse(command.TargetQuestionKey)
		response := LifecycleResponse{ResponseID: lifecycleObjectID("response", aggregate.InspectionID, command.TargetQuestionKey, previous.Revision+1), QuestionKey: command.TargetQuestionKey, Revision: previous.Revision + 1, Answer: ChecklistAnswer(command.Answer), CommentToAuditee: command.CommentToAuditee, EvidenceFileName: command.EvidenceFileName, ActorSubjectID: principal.SubjectID, CreatedAt: now.UTC()}
		response.ResponseDigest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-CHECKLIST-RESPONSE-V1", response, "responseDigest")
		aggregate.Responses = append(aggregate.Responses, response)
	case OperationCreateFinding:
		if aggregate.State != InspectionInProgress || principal.SubjectID != aggregate.Inspector.SubjectID {
			return ErrLifecycleTransition
		}
		if strings.TrimSpace(command.CommentToAuditee) == "" {
			return ErrLifecycleCommentRequired
		}
		response, ok := aggregate.latestResponse(command.TargetQuestionKey)
		if !ok || !findingEligibleAnswer(response.Answer) || response.Answer != ChecklistAnswer(command.Answer) || strings.TrimSpace(response.CommentToAuditee) == "" {
			return ErrLifecycleTransition
		}
		rootID := strings.TrimSpace(command.PotentialFindingRootID)
		version := 1
		if rootID != "" {
			latest, found := aggregate.latestPotential(rootID)
			if !found || latest.State != PotentialFindingReturned || response.Revision <= latest.ResponseRevision || response.ResponseDigest == latest.ResponseDigest {
				return ErrLifecycleTransition
			}
			version = latest.Version + 1
		} else {
			rootID = lifecycleObjectID("potential-root", aggregate.InspectionID, command.TargetQuestionKey, len(aggregate.PotentialFindings)+1)
		}
		potential := LifecyclePotentialFinding{PotentialFindingID: lifecycleObjectID("potential", rootID, response.ResponseID, version), RootID: rootID, Version: version, InspectionID: aggregate.InspectionID, QuestionKey: command.TargetQuestionKey, ResponseID: response.ResponseID, ResponseRevision: response.Revision, ResponseDigest: response.ResponseDigest, Answer: response.Answer, CommentToAuditee: response.CommentToAuditee, State: PotentialFindingPending, ActorSubjectID: principal.SubjectID, CreatedAt: now.UTC()}
		potential.Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-POTENTIAL-FINDING-V1", potential, "digest")
		aggregate.PotentialFindings = append(aggregate.PotentialFindings, potential)
	case OperationSubmitChecklist:
		if aggregate.State != InspectionInProgress || principal.SubjectID != aggregate.Inspector.SubjectID {
			return ErrLifecycleTransition
		}
		latest := latestPotentialsByRoot(*aggregate)
		for _, potential := range latest {
			if potential.State == PotentialFindingReturned {
				return ErrLifecycleTransition
			}
		}
		if len(latest) == 0 || allPotentialFindingsTerminal(latest) {
			aggregate.State = InspectionCompleted
		} else {
			aggregate.State = InspectionSubmitted
		}
	case OperationReopenChecklist:
		if (aggregate.State != InspectionSubmitted && aggregate.State != InspectionCompleted) || strings.TrimSpace(command.ReasonCode) == "" {
			return ErrLifecycleTransition
		}
		aggregate.State = InspectionInProgress
	case OperationReturnFinding, OperationDismissFinding, OperationConvertFinding:
		if principal.SubjectID != aggregate.Lead.SubjectID {
			return ErrNeutralDenied
		}
		potential, index, found := aggregate.latestPotentialByID(command.PotentialFindingID)
		if !found || potential.State != PotentialFindingPending {
			return ErrLifecycleTransition
		}
		updated := potential
		updated.Version = potential.Version + 1
		updated.PotentialFindingID = lifecycleObjectID("potential", potential.RootID, command.OperationID, updated.Version)
		updated.ReasonCode = command.ReasonCode
		updated.ActorSubjectID = principal.SubjectID
		updated.CreatedAt = now.UTC()
		switch command.OperationID {
		case OperationReturnFinding:
			if strings.TrimSpace(command.ReasonCode) == "" {
				return ErrLifecycleCommentRequired
			}
			updated.State = PotentialFindingReturned
			aggregate.State = InspectionInProgress
		case OperationDismissFinding:
			if strings.TrimSpace(command.ReasonCode) == "" {
				return ErrLifecycleCommentRequired
			}
			updated.State = PotentialFindingDismissed
		case OperationConvertFinding:
			if err := validateFindingChoices(command); err != nil {
				return err
			}
			updated.State = PotentialFindingConverted
			finding := LifecycleFinding{FindingID: lifecycleObjectID("finding", potential.RootID, aggregate.InspectionID, len(aggregate.Findings)+1), PotentialFindingRootID: potential.RootID, InspectionID: aggregate.InspectionID, QuestionKey: potential.QuestionKey, Severity: command.Severity, CAPRequired: command.CapRequired, EvidenceRequired: command.EvidenceRequired, DueDateRequired: command.DueDateRequired, DueDate: command.DueDate, Revision: 1, CreatedAt: now.UTC()}
			finding.State, finding.NextAction = initialFindingState(command.CapRequired, command.EvidenceRequired)
			finding.Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-FINDING-V1", finding, "digest")
			aggregate.Findings = append(aggregate.Findings, finding)
		}
		updated.Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-POTENTIAL-FINDING-V1", updated, "digest")
		aggregate.PotentialFindings[index] = updated
		if command.OperationID != OperationReturnFinding && aggregate.State == InspectionSubmitted && allPotentialFindingsTerminal(latestPotentialsByRoot(*aggregate)) {
			aggregate.State = InspectionCompleted
		}
	case OperationSubmitCAP:
		if err := appendCAPRevision(aggregate, command, principal, now); err != nil {
			return err
		}
	case OperationReviewCAP:
		if err := reviewCAP(aggregate, command, principal, now); err != nil {
			return err
		}
	case OperationSubmitEvidence:
		if err := appendEvidence(aggregate, command, principal, now); err != nil {
			return err
		}
	case OperationVerifyEvidence:
		if err := verifyEvidence(aggregate, command, principal, now); err != nil {
			return err
		}
	case OperationAuthorizedClose:
		if err := authorizedClose(aggregate, command, now); err != nil {
			return err
		}
	default:
		return ErrCapabilityUnavailable
	}
	return aggregate.advance(now)
}

func principalMayTouchAggregate(aggregate LifecycleAggregate, principal identity.Principal, operation string) bool {
	if !lifecycleBindingPinMatchesPrincipal(aggregate.Inspector, principal) &&
		!lifecycleBindingPinMatchesPrincipal(aggregate.Lead, principal) &&
		!lifecycleBindingPinMatchesPrincipal(aggregate.Auditee, principal) &&
		!workspaceOrganizationMatchesPrincipal(principal.OrganizationID, aggregate.OrganizationID) {
		return false
	}
	switch operation {
	case OperationStartInspection, OperationRecordResponse, OperationCreateFinding, OperationSubmitChecklist:
		return principal.SubjectID == aggregate.Inspector.SubjectID
	case OperationReopenChecklist, OperationVerifyEvidence:
		return principal.SubjectID == aggregate.Inspector.SubjectID || principal.SubjectID == aggregate.Lead.SubjectID
	case OperationReturnFinding, OperationDismissFinding, OperationConvertFinding:
		return principal.SubjectID == aggregate.Lead.SubjectID
	case OperationSubmitCAP, OperationSubmitEvidence:
		return principal.HasRole(identity.RoleAuditee) && aggregate.Auditee.SubjectID != "" && principal.SubjectID == aggregate.Auditee.SubjectID
	case OperationReviewCAP:
		return principal.HasRole(identity.RoleLeadInspector) || principal.HasRole(identity.RoleDepartmentManager)
	case OperationAuthorizedClose:
		return principal.HasRole(identity.RoleDepartmentManager)
	default:
		return false
	}
}

func validateFindingChoices(command CommandEnvelope) error {
	if strings.TrimSpace(command.Severity) == "" || (command.DueDateRequired != (command.DueDate != nil)) {
		return ErrLifecycleChoiceInvalid
	}
	if command.DueDate != nil && command.DueDate.IsZero() {
		return ErrLifecycleChoiceInvalid
	}
	return nil
}

func initialFindingState(capRequired, evidenceRequired bool) (FindingState, string) {
	if capRequired {
		return FindingWaitingForCAP, "SUBMIT_CAP_REVISION"
	}
	if evidenceRequired {
		return FindingEvidenceRequired, "SUBMIT_EVIDENCE_VERSION"
	}
	return FindingPendingClosure, "AUTHORIZED_CLOSE"
}

func latestPotentialsByRoot(aggregate LifecycleAggregate) map[string]LifecyclePotentialFinding {
	latest := make(map[string]LifecyclePotentialFinding)
	for _, potential := range aggregate.PotentialFindings {
		current, found := latest[potential.RootID]
		if !found || potential.Version > current.Version {
			latest[potential.RootID] = potential
		}
	}
	return latest
}

func allPotentialFindingsTerminal(values map[string]LifecyclePotentialFinding) bool {
	for _, value := range values {
		if !isTerminalPotentialFinding(value.State) {
			return false
		}
	}
	return true
}

func (aggregate LifecycleAggregate) latestPotentialByID(id string) (LifecyclePotentialFinding, int, bool) {
	for index := len(aggregate.PotentialFindings) - 1; index >= 0; index-- {
		if aggregate.PotentialFindings[index].PotentialFindingID == id {
			return aggregate.PotentialFindings[index], index, true
		}
	}
	return LifecyclePotentialFinding{}, -1, false
}

func appendCAPRevision(aggregate *LifecycleAggregate, command CommandEnvelope, principal identity.Principal, now time.Time) error {
	finding, findingFound := aggregate.latestFinding(command.FindingID)
	if !findingFound || !principalMayTouchAggregate(*aggregate, principal, OperationSubmitCAP) || (finding.State != FindingWaitingForCAP && finding.State != FindingCAPRejected && finding.State != FindingCAPMoreInformationRequested) {
		return ErrLifecycleTransition
	}
	if strings.TrimSpace(command.RootCause) == "" || strings.TrimSpace(command.CorrectiveAction) == "" || strings.TrimSpace(command.PreventiveAction) == "" || strings.TrimSpace(command.ResponsiblePerson) == "" {
		return ErrLifecycleChoiceInvalid
	}
	previous, hasPrevious := aggregate.latestCAP(command.FindingID)
	if hasPrevious && previous.State != CAPAccepted {
		superseded := previous
		superseded.State = CAPSuperseded
		superseded.CreatedAt = now.UTC()
		superseded.Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-CAP-REVISION-V1", superseded, "digest")
		aggregate.CAPRevisions = append(aggregate.CAPRevisions, superseded)
	}
	revision := 1
	if hasPrevious {
		revision = previous.Revision + 1
	}
	cap := LifecycleCAPRevision{CAPID: lifecycleObjectID("cap", aggregate.InspectionID, command.FindingID, 1), FindingID: command.FindingID, Revision: revision, State: CAPSubmitted, RootCause: command.RootCause, CorrectiveAction: command.CorrectiveAction, PreventiveAction: command.PreventiveAction, ResponsiblePerson: command.ResponsiblePerson, TargetDate: command.DueDate, CommentToAuditee: command.CommentToAuditee, ActorSubjectID: principal.SubjectID, CreatedAt: now.UTC()}
	cap.Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-CAP-REVISION-V1", cap, "digest")
	aggregate.CAPRevisions = append(aggregate.CAPRevisions, cap)
	cap.State = CAPPendingCAAReview
	cap.Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-CAP-REVISION-V1", cap, "digest")
	aggregate.CAPRevisions = append(aggregate.CAPRevisions, cap)
	for index := range aggregate.Findings {
		if aggregate.Findings[index].FindingID == finding.FindingID {
			aggregate.Findings[index].State = FindingPendingCAAReview
			aggregate.Findings[index].NextAction = "REVIEW_CAP"
			aggregate.Findings[index].Revision++
			aggregate.Findings[index].Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-FINDING-V1", aggregate.Findings[index], "digest")
		}
	}
	return nil
}

func reviewCAP(aggregate *LifecycleAggregate, command CommandEnvelope, principal identity.Principal, now time.Time) error {
	finding, found := aggregate.latestFinding(command.FindingID)
	if !found || finding.State != FindingPendingCAAReview || strings.TrimSpace(command.CommentToAuditee) == "" || strings.TrimSpace(command.InternalCAANote) == "" {
		return ErrLifecycleTransition
	}
	cap, found := aggregate.latestCAP(command.FindingID)
	if !found || cap.State != CAPPendingCAAReview {
		return ErrLifecycleTransition
	}
	updated := cap
	updated.CommentToAuditee = command.CommentToAuditee
	updated.InternalCAANote = command.InternalCAANote
	updated.ActorSubjectID = principal.SubjectID
	updated.CreatedAt = now.UTC()
	switch command.Outcome {
	case "ACCEPT":
		updated.State = CAPAccepted
		for index := range aggregate.Findings {
			if aggregate.Findings[index].FindingID == finding.FindingID {
				if finding.EvidenceRequired {
					setFindingState(aggregate, finding.FindingID, FindingEvidenceRequired, "SUBMIT_EVIDENCE_VERSION")
				} else {
					setFindingState(aggregate, finding.FindingID, FindingPendingClosure, "AUTHORIZED_CLOSE")
				}
			}
		}
	case "REJECT":
		updated.State = CAPRejected
		setFindingState(aggregate, finding.FindingID, FindingCAPRejected, "SUBMIT_CAP_REVISION")
	case "MORE_INFORMATION_REQUESTED":
		updated.State = CAPMoreInformation
		setFindingState(aggregate, finding.FindingID, FindingCAPMoreInformationRequested, "SUBMIT_CAP_REVISION")
	default:
		return ErrLifecycleChoiceInvalid
	}
	updated.Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-CAP-REVISION-V1", updated, "digest")
	aggregate.CAPRevisions = append(aggregate.CAPRevisions, updated)
	return nil
}

func appendEvidence(aggregate *LifecycleAggregate, command CommandEnvelope, principal identity.Principal, now time.Time) error {
	finding, found := aggregate.latestFinding(command.FindingID)
	if !found || (finding.State != FindingEvidenceRequired && finding.State != FindingEvidenceMoreInformation) || strings.TrimSpace(command.EvidenceFileName) == "" || strings.TrimSpace(command.CommentToAuditee) == "" {
		return ErrLifecycleTransition
	}
	previous, _ := aggregate.latestEvidence(command.FindingID)
	evidence := LifecycleEvidenceVersion{EvidenceID: lifecycleObjectID("evidence", aggregate.InspectionID, command.FindingID, previous.Version+1), FindingID: command.FindingID, Version: previous.Version + 1, FileName: command.EvidenceFileName, ReviewState: EvidencePendingCAAReview, CommentToAuditee: command.CommentToAuditee, ActorSubjectID: principal.SubjectID, CreatedAt: now.UTC()}
	evidence.Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-EVIDENCE-VERSION-V1", evidence, "digest")
	aggregate.EvidenceVersions = append(aggregate.EvidenceVersions, evidence)
	setFindingState(aggregate, finding.FindingID, FindingPendingCAAReview, "VERIFY_EVIDENCE")
	return nil
}

func verifyEvidence(aggregate *LifecycleAggregate, command CommandEnvelope, principal identity.Principal, now time.Time) error {
	finding, found := aggregate.latestFinding(command.FindingID)
	if !found || finding.State != FindingPendingCAAReview || strings.TrimSpace(command.CommentToAuditee) == "" || strings.TrimSpace(command.InternalCAANote) == "" {
		return ErrLifecycleTransition
	}
	evidence, found := aggregate.latestEvidence(command.FindingID)
	if !found || evidence.ReviewState != EvidencePendingCAAReview {
		return ErrLifecycleTransition
	}
	updated := evidence
	updated.CommentToAuditee = command.CommentToAuditee
	updated.InternalCAANote = command.InternalCAANote
	updated.ActorSubjectID = principal.SubjectID
	switch EvidenceVerificationOutcome(command.Outcome) {
	case EvidenceClose:
		updated.ReviewState = EvidenceAccepted
		setFindingState(aggregate, finding.FindingID, FindingClosed, "CLOSED")
		setFindingClosure(aggregate, finding.FindingID, "EVIDENCE_VERIFIED")
	case EvidencePartiallyClose:
		updated.ReviewState = EvidencePartiallyAccepted
		setFindingState(aggregate, finding.FindingID, FindingEvidenceMoreInformation, "SUBMIT_EVIDENCE_VERSION")
	case EvidenceNotClose:
		updated.ReviewState = EvidenceRejected
		setFindingState(aggregate, finding.FindingID, FindingEvidenceMoreInformation, "SUBMIT_EVIDENCE_VERSION")
	case EvidenceRequestMoreInformation:
		updated.ReviewState = EvidenceMoreInformation
		setFindingState(aggregate, finding.FindingID, FindingEvidenceMoreInformation, "SUBMIT_EVIDENCE_VERSION")
	default:
		return ErrLifecycleChoiceInvalid
	}
	updated.Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-EVIDENCE-VERSION-V1", updated, "digest")
	aggregate.EvidenceVersions = append(aggregate.EvidenceVersions, updated)
	decision := LifecycleVerificationDecision{VerificationID: lifecycleObjectID("verification", aggregate.InspectionID, evidence.EvidenceID, len(aggregate.VerificationDecisions)+1), FindingID: finding.FindingID, EvidenceID: evidence.EvidenceID, EvidenceVersion: evidence.Version, Outcome: EvidenceVerificationOutcome(command.Outcome), CommentToAuditee: command.CommentToAuditee, InternalCAANote: command.InternalCAANote, ActorSubjectID: principal.SubjectID, CreatedAt: now.UTC()}
	decision.Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-VERIFICATION-DECISION-V1", decision, "digest")
	aggregate.VerificationDecisions = append(aggregate.VerificationDecisions, decision)
	return nil
}

func authorizedClose(aggregate *LifecycleAggregate, command CommandEnvelope, now time.Time) error {
	if strings.TrimSpace(command.ReasonCode) == "" {
		return ErrLifecycleCommentRequired
	}
	finding, found := aggregate.latestFinding(command.FindingID)
	if !found || finding.State != FindingPendingClosure {
		return ErrLifecycleTransition
	}
	setFindingState(aggregate, finding.FindingID, FindingClosed, "CLOSED")
	setFindingClosure(aggregate, finding.FindingID, "AUTHORIZED_CLOSURE")
	return nil
}

func setFindingState(aggregate *LifecycleAggregate, findingID string, state FindingState, next string) {
	for index := range aggregate.Findings {
		if aggregate.Findings[index].FindingID == findingID {
			aggregate.Findings[index].State, aggregate.Findings[index].NextAction = state, next
			aggregate.Findings[index].Revision++
			aggregate.Findings[index].Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-FINDING-V1", aggregate.Findings[index], "digest")
		}
	}
}

func setFindingClosure(aggregate *LifecycleAggregate, findingID, basis string) {
	for index := range aggregate.Findings {
		if aggregate.Findings[index].FindingID == findingID {
			aggregate.Findings[index].ClosureBasis = basis
			aggregate.Findings[index].Revision++
			aggregate.Findings[index].Digest, _ = aga.DigestExcludingJSONFields("AGA-DEMO-FINDING-V1", aggregate.Findings[index], "digest")
		}
	}
}

func (service *Service) applyLifecycle(ctx context.Context, principal identity.Principal, command CommandEnvelope) (LifecycleAggregate, error) {
	aggregate, events, err := service.loadLifecycle(ctx, command.ExpectedGenerationID, command.InspectionID)
	if err != nil {
		return LifecycleAggregate{}, err
	}
	if err := applyLifecycleCommand(&aggregate, command, principal, service.clock().UTC()); err != nil {
		return LifecycleAggregate{}, err
	}
	if err := service.appendLifecycle(ctx, aggregate, events, command, principal); err != nil {
		return LifecycleAggregate{}, fmt.Errorf("append lifecycle event: %w", err)
	}
	return aggregate, nil
}

func (service *Service) lifecycleCommand(ctx context.Context, principal identity.Principal, command CommandEnvelope) (LifecycleAggregate, error) {
	if strings.TrimSpace(command.InspectionID) == "" {
		return LifecycleAggregate{}, ErrLifecycleNotFound
	}
	return service.applyLifecycle(ctx, principal, command)
}

func validateLifecycleCommand(command CommandEnvelope) error {
	if strings.TrimSpace(command.InspectionID) == "" {
		return ErrMalformedCommand
	}
	switch command.OperationID {
	case OperationStartInspection, OperationRecordResponse, OperationCreateFinding, OperationSubmitChecklist, OperationReopenChecklist:
		if command.OperationID == OperationRecordResponse || command.OperationID == OperationCreateFinding {
			if strings.TrimSpace(command.TargetQuestionKey) == "" {
				return ErrMalformedCommand
			}
		}
	case OperationReturnFinding, OperationDismissFinding, OperationConvertFinding:
		if strings.TrimSpace(command.PotentialFindingID) == "" || strings.TrimSpace(command.ReasonCode) == "" {
			return ErrMalformedCommand
		}
		if command.OperationID == OperationConvertFinding {
			if err := validateFindingChoices(command); err != nil {
				return ErrMalformedCommand
			}
		}
	case OperationSubmitCAP:
		if strings.TrimSpace(command.FindingID) == "" || strings.TrimSpace(command.RootCause) == "" || strings.TrimSpace(command.CorrectiveAction) == "" || strings.TrimSpace(command.PreventiveAction) == "" || strings.TrimSpace(command.ResponsiblePerson) == "" {
			return ErrMalformedCommand
		}
	case OperationReviewCAP:
		if strings.TrimSpace(command.FindingID) == "" || strings.TrimSpace(command.Outcome) == "" || strings.TrimSpace(command.CommentToAuditee) == "" || strings.TrimSpace(command.InternalCAANote) == "" {
			return ErrMalformedCommand
		}
	case OperationSubmitEvidence:
		if strings.TrimSpace(command.FindingID) == "" || strings.TrimSpace(command.EvidenceFileName) == "" || strings.TrimSpace(command.CommentToAuditee) == "" {
			return ErrMalformedCommand
		}
	case OperationVerifyEvidence:
		if strings.TrimSpace(command.FindingID) == "" || strings.TrimSpace(command.Outcome) == "" || strings.TrimSpace(command.CommentToAuditee) == "" || strings.TrimSpace(command.InternalCAANote) == "" {
			return ErrMalformedCommand
		}
	case OperationAuthorizedClose:
		if strings.TrimSpace(command.FindingID) == "" || strings.TrimSpace(command.ReasonCode) == "" {
			return ErrMalformedCommand
		}
	default:
		return ErrMalformedCommand
	}
	return nil
}

func errorsAsLifecycleNeutral(err error) bool {
	return errors.Is(err, ErrLifecycleNotFound) || errors.Is(err, ErrLifecycleRecommendationStale) || errors.Is(err, ErrLifecycleBindingMismatch)
}
