package agademoworkspace

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/identity"
)

var lifecycleTestNow = time.Date(2026, 8, 4, 13, 0, 0, 0, time.UTC)

func lifecyclePrincipals(organizationID string) (identity.Principal, identity.Principal, identity.Principal, identity.Principal) {
	return identity.Principal{SubjectID: "aga-inspector-subject", OrganizationID: organizationID, Roles: []identity.Role{identity.RoleInspector}},
		identity.Principal{SubjectID: "aga-lead-subject", OrganizationID: organizationID, Roles: []identity.Role{identity.RoleLeadInspector}},
		identity.Principal{SubjectID: "aga-auditee-subject", OrganizationID: organizationID, Roles: []identity.Role{identity.RoleAuditee}},
		identity.Principal{SubjectID: "aga-manager-subject", OrganizationID: organizationID, Roles: []identity.Role{identity.RoleDepartmentManager}}
}

func lifecycleFixture(t *testing.T) (LifecycleAggregate, identity.Principal, identity.Principal, identity.Principal, identity.Principal) {
	t.Helper()
	snapshot := validRecommendationSnapshotForTest(t)
	command := CommandEnvelope{
		OperationID: OperationCreateInspection, ExpectedGenerationID: snapshot.Recommendation.GenerationID,
		RecommendationID: snapshot.Recommendation.RecommendationID, RecommendationDigest: snapshot.Recommendation.Digest,
		ExpectedRecommendationRevision: snapshot.Recommendation.Revision,
		InspectorBindingID:             "aga-binding-inspector", InspectorBindingRevision: 1,
		LeadBindingID: "aga-binding-lead", LeadBindingRevision: 1,
	}
	inspector, lead, auditee, manager := lifecyclePrincipals(snapshot.Recommendation.OrganizationID)
	inspectorFact := LifecycleBindingFact{
		BindingID: command.InspectorBindingID, BindingRevision: command.InspectorBindingRevision, SubjectID: inspector.SubjectID,
		MembershipSlot: "inspector-membership", OrganizationID: snapshot.Recommendation.OrganizationID,
		DepartmentID: snapshot.Recommendation.DepartmentID, OrganizationalUnitID: snapshot.Recommendation.OrganizationalUnitID,
		ProviderScopeID: snapshot.Recommendation.ProviderScopeID, Active: true,
	}
	leadFact := inspectorFact
	leadFact.BindingID, leadFact.BindingRevision, leadFact.SubjectID, leadFact.MembershipSlot = command.LeadBindingID, command.LeadBindingRevision, lead.SubjectID, "lead-membership"
	aggregate, err := buildInspectionFromRecommendation(snapshot, command, inspectorFact, leadFact, lifecycleTestNow)
	if err != nil {
		t.Fatalf("build lifecycle fixture: %v", err)
	}
	aggregate.Auditee = LifecycleBindingPin{BindingID: "aga-binding-auditee", BindingRevision: 1, SubjectID: auditee.SubjectID, MembershipSlot: "auditee-membership", OrganizationID: snapshot.Recommendation.OrganizationID, DepartmentID: snapshot.Recommendation.DepartmentID, OrganizationalUnitID: snapshot.Recommendation.OrganizationalUnitID}
	aggregate.Digest = lifecycleDigest(aggregate)
	return aggregate, inspector, lead, auditee, manager
}

func TestLifecycleQuestionSnapshotRoundTripsBaseRootSequence(t *testing.T) {
	aggregate, _, _, _, _ := lifecycleFixture(t)
	encoded, err := json.Marshal(aggregate.Questions[0])
	if err != nil {
		t.Fatalf("marshal lifecycle question snapshot: %v", err)
	}
	var decoded LifecycleQuestionSnapshot
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal lifecycle question snapshot: %v", err)
	}
	if decoded.RootSequence != aggregate.Questions[0].RootSequence || decoded.QuestionRef.RootSequence != aggregate.Questions[0].QuestionRef.RootSequence || decoded.QuestionKey != aggregate.Questions[0].QuestionKey {
		t.Fatalf("lifecycle question snapshot lost base root ordering: got root=%d refRoot=%d key=%s", decoded.RootSequence, decoded.QuestionRef.RootSequence, decoded.QuestionKey)
	}
}

func lifecycleCommandForTest(aggregate LifecycleAggregate, operation string) CommandEnvelope {
	return CommandEnvelope{
		OperationID: operation, IdempotencyKey: operation + "-" + time.Now().UTC().Format("150405.000000000"),
		ExpectedGenerationID: aggregate.GenerationID, ExpectedLifecycleRevision: aggregate.Revision,
		ExpectedLifecycleDigest: aggregate.Digest, InspectionID: aggregate.InspectionID,
	}
}

func applyLifecycleTest(t *testing.T, aggregate *LifecycleAggregate, principal identity.Principal, operation string, configure func(*CommandEnvelope)) CommandEnvelope {
	t.Helper()
	command := lifecycleCommandForTest(*aggregate, operation)
	if configure != nil {
		configure(&command)
	}
	if err := applyLifecycleCommand(aggregate, command, principal, lifecycleTestNow.Add(time.Duration(aggregate.Revision)*time.Minute)); err != nil {
		t.Fatalf("apply %s: %v", operation, err)
	}
	return command
}

func startAndCreatePotential(t *testing.T, aggregate *LifecycleAggregate, inspector identity.Principal) LifecyclePotentialFinding {
	t.Helper()
	applyLifecycleTest(t, aggregate, inspector, OperationStartInspection, nil)
	questionKey := aggregate.Questions[0].QuestionRef.Key()
	applyLifecycleTest(t, aggregate, inspector, OperationRecordResponse, func(command *CommandEnvelope) {
		command.TargetQuestionKey = questionKey
		command.Answer = string(AnswerNonCompliant)
		command.CommentToAuditee = "The synthetic checklist response requires follow-up."
	})
	applyLifecycleTest(t, aggregate, inspector, OperationCreateFinding, func(command *CommandEnvelope) {
		command.TargetQuestionKey = questionKey
		command.Answer = string(AnswerNonCompliant)
		command.CommentToAuditee = "The synthetic checklist response requires follow-up."
	})
	potential, found := aggregate.latestPotential(aggregate.PotentialFindings[0].RootID)
	if !found {
		t.Fatal("potential finding was not created")
	}
	return potential
}

func convertPotential(t *testing.T, aggregate *LifecycleAggregate, lead identity.Principal, capRequired, evidenceRequired, dueDateRequired bool) LifecycleFinding {
	t.Helper()
	potential := aggregate.PotentialFindings[len(aggregate.PotentialFindings)-1]
	var dueDate *time.Time
	if dueDateRequired {
		value := lifecycleTestNow.Add(30 * 24 * time.Hour)
		dueDate = &value
	}
	applyLifecycleTest(t, aggregate, lead, OperationConvertFinding, func(command *CommandEnvelope) {
		command.PotentialFindingID = potential.PotentialFindingID
		command.ReasonCode = "LEAD_REVIEW_CONVERT"
		command.Severity = "MAJOR"
		command.CapRequired = capRequired
		command.EvidenceRequired = evidenceRequired
		command.DueDateRequired = dueDateRequired
		command.DueDate = dueDate
	})
	return aggregate.Findings[len(aggregate.Findings)-1]
}

func findingFixture(t *testing.T, capRequired, evidenceRequired, dueDateRequired bool) (LifecycleAggregate, identity.Principal, identity.Principal, identity.Principal, identity.Principal, LifecycleFinding) {
	t.Helper()
	aggregate, inspector, lead, auditee, manager := lifecycleFixture(t)
	startAndCreatePotential(t, &aggregate, inspector)
	finding := convertPotential(t, &aggregate, lead, capRequired, evidenceRequired, dueDateRequired)
	return aggregate, inspector, lead, auditee, manager, finding
}

func submitCAP(t *testing.T, aggregate *LifecycleAggregate, auditee identity.Principal, finding LifecycleFinding, rootCause string) {
	t.Helper()
	applyLifecycleTest(t, aggregate, auditee, OperationSubmitCAP, func(command *CommandEnvelope) {
		command.FindingID = finding.FindingID
		command.RootCause = rootCause
		command.CorrectiveAction = "Correct the synthetic process control."
		command.PreventiveAction = "Review the synthetic control at the next cycle."
		command.ResponsiblePerson = "Synthetic accountable owner"
		command.CommentToAuditee = "CAP revision submitted for CAA review."
	})
}

func reviewCAPForTest(t *testing.T, aggregate *LifecycleAggregate, lead identity.Principal, finding LifecycleFinding, outcome string) {
	t.Helper()
	applyLifecycleTest(t, aggregate, lead, OperationReviewCAP, func(command *CommandEnvelope) {
		command.FindingID = finding.FindingID
		command.Outcome = outcome
		command.CommentToAuditee = "CAA review outcome for the synthetic CAP."
		command.InternalCAANote = "Internal CAA review note for the synthetic CAP."
	})
}

func submitEvidenceForTest(t *testing.T, aggregate *LifecycleAggregate, auditee identity.Principal, finding LifecycleFinding, fileName string) {
	t.Helper()
	applyLifecycleTest(t, aggregate, auditee, OperationSubmitEvidence, func(command *CommandEnvelope) {
		command.FindingID = finding.FindingID
		command.EvidenceFileName = fileName
		command.CommentToAuditee = "Synthetic evidence metadata submitted for review."
	})
}

func verifyEvidenceForTest(t *testing.T, aggregate *LifecycleAggregate, lead identity.Principal, finding LifecycleFinding, outcome EvidenceVerificationOutcome) {
	t.Helper()
	applyLifecycleTest(t, aggregate, lead, OperationVerifyEvidence, func(command *CommandEnvelope) {
		command.FindingID = finding.FindingID
		command.Outcome = string(outcome)
		command.CommentToAuditee = "CAA evidence review outcome for the synthetic record."
		command.InternalCAANote = "Internal CAA evidence review note."
	})
}

func TestLifecycleRequiresPotentialFindingConversion(t *testing.T) {
	aggregate, inspector, _, auditee, _ := lifecycleFixture(t)
	potential := startAndCreatePotential(t, &aggregate, inspector)
	if len(aggregate.Findings) != 0 {
		t.Fatal("inspector created a Finding before Lead conversion")
	}
	command := lifecycleCommandForTest(aggregate, OperationSubmitCAP)
	command.FindingID = "missing-finding"
	command.RootCause, command.CorrectiveAction, command.PreventiveAction, command.ResponsiblePerson = "root", "corrective", "preventive", "owner"
	if err := applyLifecycleCommand(&aggregate, command, auditee, lifecycleTestNow); !errors.Is(err, ErrLifecycleTransition) {
		t.Fatalf("CAP before conversion error = %v, want transition error", err)
	}
	if aggregate.PotentialFindings[0].PotentialFindingID != potential.PotentialFindingID || len(aggregate.Findings) != 0 {
		t.Fatal("failed pre-conversion command changed the potential/finding projection")
	}
}

func TestInspectionRequiresExactCurrentRecommendation(t *testing.T) {
	snapshot := validRecommendationSnapshotForTest(t)
	command := CommandEnvelope{
		OperationID: OperationCreateInspection, ExpectedGenerationID: snapshot.Recommendation.GenerationID,
		RecommendationID: snapshot.Recommendation.RecommendationID, RecommendationDigest: snapshot.Recommendation.Digest,
		ExpectedRecommendationRevision: snapshot.Recommendation.Revision, InspectorBindingID: "inspector-binding", InspectorBindingRevision: 1,
		LeadBindingID: "lead-binding", LeadBindingRevision: 1,
	}
	inspector := LifecycleBindingFact{BindingID: command.InspectorBindingID, BindingRevision: 1, SubjectID: "inspector", OrganizationID: snapshot.Recommendation.OrganizationID, DepartmentID: snapshot.Recommendation.DepartmentID, OrganizationalUnitID: snapshot.Recommendation.OrganizationalUnitID, ProviderScopeID: snapshot.Recommendation.ProviderScopeID, Active: true}
	lead := inspector
	lead.BindingID, lead.SubjectID = command.LeadBindingID, "lead"
	command.RecommendationDigest = "sha256:" + strings.Repeat("1", 64)
	if _, err := buildInspectionFromRecommendation(snapshot, command, inspector, lead, lifecycleTestNow); !errors.Is(err, ErrLifecycleRecommendationStale) {
		t.Fatalf("stale recommendation digest error = %v", err)
	}
	command.RecommendationDigest = snapshot.Recommendation.Digest
	command.ExpectedRecommendationRevision++
	if _, err := buildInspectionFromRecommendation(snapshot, command, inspector, lead, lifecycleTestNow); !errors.Is(err, ErrLifecycleRecommendationStale) {
		t.Fatalf("stale recommendation revision error = %v", err)
	}
}

func TestFindingInitialStateCoversEveryCAPEvidenceChoice(t *testing.T) {
	cases := []struct {
		capRequired, evidenceRequired bool
		state                         FindingState
		next                          string
	}{
		{true, true, FindingWaitingForCAP, "SUBMIT_CAP_REVISION"},
		{true, false, FindingWaitingForCAP, "SUBMIT_CAP_REVISION"},
		{false, true, FindingEvidenceRequired, "SUBMIT_EVIDENCE_VERSION"},
		{false, false, FindingPendingClosure, "AUTHORIZED_CLOSE"},
	}
	for _, testCase := range cases {
		state, next := initialFindingState(testCase.capRequired, testCase.evidenceRequired)
		if state != testCase.state || next != testCase.next {
			t.Errorf("initial state cap=%v evidence=%v = %s/%s, want %s/%s", testCase.capRequired, testCase.evidenceRequired, state, next, testCase.state, testCase.next)
		}
	}
}

func TestDueDateChoiceIsIndependentAndExact(t *testing.T) {
	dueDate := lifecycleTestNow.Add(30 * 24 * time.Hour)
	for _, capRequired := range []bool{false, true} {
		for _, evidenceRequired := range []bool{false, true} {
			for _, dueDateRequired := range []bool{false, true} {
				command := CommandEnvelope{Severity: "MINOR", CapRequired: capRequired, EvidenceRequired: evidenceRequired, DueDateRequired: dueDateRequired}
				if dueDateRequired {
					command.DueDate = &dueDate
				}
				if err := validateFindingChoices(command); err != nil {
					t.Errorf("independent valid choices cap=%v evidence=%v due=%v: %v", capRequired, evidenceRequired, dueDateRequired, err)
				}
				if dueDateRequired {
					missing := command
					missing.DueDate = nil
					if err := validateFindingChoices(missing); !errors.Is(err, ErrLifecycleChoiceInvalid) {
						t.Errorf("missing required due date cap=%v evidence=%v error=%v", capRequired, evidenceRequired, err)
					}
				} else {
					unexpected := command
					unexpected.DueDate = &dueDate
					if err := validateFindingChoices(unexpected); !errors.Is(err, ErrLifecycleChoiceInvalid) {
						t.Errorf("unexpected due date cap=%v evidence=%v error=%v", capRequired, evidenceRequired, err)
					}
				}
			}
		}
	}
}

func TestReopenAndInspectionCompletionTransitionsAreTotal(t *testing.T) {
	aggregate, inspector, lead, _, _ := lifecycleFixture(t)
	potential := startAndCreatePotential(t, &aggregate, inspector)
	applyLifecycleTest(t, &aggregate, inspector, OperationSubmitChecklist, nil)
	if aggregate.State != InspectionSubmitted {
		t.Fatalf("pending potential submission state = %s, want %s", aggregate.State, InspectionSubmitted)
	}
	applyLifecycleTest(t, &aggregate, lead, OperationReturnFinding, func(command *CommandEnvelope) {
		command.PotentialFindingID = potential.PotentialFindingID
		command.ReasonCode = "RETURN_FOR_CORRECTION"
	})
	if aggregate.State != InspectionInProgress || aggregate.PotentialFindings[0].State != PotentialFindingReturned {
		t.Fatalf("return state=%s potential=%s, want in-progress/returned", aggregate.State, aggregate.PotentialFindings[0].State)
	}
	questionKey := aggregate.Questions[0].QuestionRef.Key()
	applyLifecycleTest(t, &aggregate, inspector, OperationRecordResponse, func(command *CommandEnvelope) {
		command.TargetQuestionKey = questionKey
		command.Answer = string(AnswerObservation)
		command.CommentToAuditee = "Corrected synthetic response after lead return."
	})
	applyLifecycleTest(t, &aggregate, inspector, OperationCreateFinding, func(command *CommandEnvelope) {
		command.TargetQuestionKey = questionKey
		command.Answer = string(AnswerObservation)
		command.CommentToAuditee = "Corrected synthetic response after lead return."
		command.PotentialFindingRootID = potential.RootID
	})
	applyLifecycleTest(t, &aggregate, inspector, OperationSubmitChecklist, nil)
	latest := aggregate.PotentialFindings[len(aggregate.PotentialFindings)-1]
	applyLifecycleTest(t, &aggregate, lead, OperationDismissFinding, func(command *CommandEnvelope) {
		command.PotentialFindingID = latest.PotentialFindingID
		command.ReasonCode = "DISMISS_AFTER_CORRECTION"
	})
	if aggregate.State != InspectionCompleted || len(aggregate.PotentialFindings) != 2 || aggregate.PotentialFindings[0].State != PotentialFindingReturned || aggregate.PotentialFindings[1].State != PotentialFindingDismissed {
		t.Fatalf("terminal transition aggregate=%+v", aggregate)
	}
	questionCount, potentialCount := len(aggregate.Questions), len(aggregate.PotentialFindings)
	applyLifecycleTest(t, &aggregate, inspector, OperationReopenChecklist, func(command *CommandEnvelope) {
		command.ReasonCode = "REOPEN_FOR_REVIEW"
	})
	if aggregate.State != InspectionInProgress || len(aggregate.Questions) != questionCount || len(aggregate.PotentialFindings) != potentialCount {
		t.Fatalf("reopen did not preserve pinned graph: state=%s questions=%d potentials=%d", aggregate.State, len(aggregate.Questions), len(aggregate.PotentialFindings))
	}
}

func TestCAPAcceptanceLeavesFindingOpen(t *testing.T) {
	aggregate, _, lead, auditee, manager, finding := findingFixture(t, true, false, false)
	submitCAP(t, &aggregate, auditee, finding, "root cause one")
	reviewCAPForTest(t, &aggregate, lead, finding, "ACCEPT")
	latestCAP, found := aggregate.latestCAP(finding.FindingID)
	latestFinding, findingFound := aggregate.latestFinding(finding.FindingID)
	if !found || latestCAP.State != CAPAccepted || !findingFound || latestFinding.State != FindingPendingClosure || latestFinding.State == FindingClosed {
		t.Fatalf("CAP acceptance projection = cap=%s finding=%s", latestCAP.State, latestFinding.State)
	}
	applyLifecycleTest(t, &aggregate, manager, OperationAuthorizedClose, func(command *CommandEnvelope) {
		command.FindingID = finding.FindingID
		command.ReasonCode = "MANAGER_AUTHORIZED_CLOSURE"
	})
	closed, _ := aggregate.latestFinding(finding.FindingID)
	if closed.State != FindingClosed || closed.ClosureBasis != "AUTHORIZED_CLOSURE" {
		t.Fatalf("authorized closure = %+v", closed)
	}
}

func TestCAPAndEvidenceResubmissionTransitionsAreTotal(t *testing.T) {
	aggregate, _, lead, auditee, _, finding := findingFixture(t, true, true, true)
	submitCAP(t, &aggregate, auditee, finding, "root cause one")
	reviewCAPForTest(t, &aggregate, lead, finding, "REJECT")
	rejected, _ := aggregate.latestCAP(finding.FindingID)
	if rejected.State != CAPRejected {
		t.Fatalf("CAP rejection state = %s", rejected.State)
	}
	submitCAP(t, &aggregate, auditee, finding, "root cause corrected")
	reviewCAPForTest(t, &aggregate, lead, finding, "ACCEPT")
	accepted, _ := aggregate.latestCAP(finding.FindingID)
	updatedFinding, _ := aggregate.latestFinding(finding.FindingID)
	if accepted.State != CAPAccepted || updatedFinding.State != FindingEvidenceRequired {
		t.Fatalf("CAP resubmission acceptance cap=%s finding=%s", accepted.State, updatedFinding.State)
	}
	submitEvidenceForTest(t, &aggregate, auditee, finding, "evidence-v1.pdf")
	firstEvidence, _ := aggregate.latestEvidence(finding.FindingID)
	if firstEvidence.Version != 1 || firstEvidence.ReviewState != EvidencePendingCAAReview {
		t.Fatalf("first evidence = %+v", firstEvidence)
	}
	verifyEvidenceForTest(t, &aggregate, lead, finding, EvidencePartiallyClose)
	partialEvidence, _ := aggregate.latestEvidence(finding.FindingID)
	partialFinding, _ := aggregate.latestFinding(finding.FindingID)
	if partialEvidence.ReviewState != EvidencePartiallyAccepted || partialFinding.State != FindingEvidenceMoreInformation {
		t.Fatalf("partial evidence review evidence=%s finding=%s", partialEvidence.ReviewState, partialFinding.State)
	}
	submitEvidenceForTest(t, &aggregate, auditee, finding, "evidence-v2.pdf")
	secondEvidence, _ := aggregate.latestEvidence(finding.FindingID)
	if secondEvidence.Version != 2 || secondEvidence.ReviewState != EvidencePendingCAAReview || len(aggregate.EvidenceVersions) != 3 {
		t.Fatalf("evidence resubmission latest=%+v history=%d", secondEvidence, len(aggregate.EvidenceVersions))
	}
}

func TestEvidenceReviewOutcomeMappingIsAtomic(t *testing.T) {
	cases := []struct {
		outcome EvidenceVerificationOutcome
		review  EvidenceReviewState
		finding FindingState
		basis   string
	}{
		{EvidenceClose, EvidenceAccepted, FindingClosed, "EVIDENCE_VERIFIED"},
		{EvidencePartiallyClose, EvidencePartiallyAccepted, FindingEvidenceMoreInformation, ""},
		{EvidenceNotClose, EvidenceRejected, FindingEvidenceMoreInformation, ""},
		{EvidenceRequestMoreInformation, EvidenceMoreInformation, FindingEvidenceMoreInformation, ""},
	}
	for _, testCase := range cases {
		t.Run(string(testCase.outcome), func(t *testing.T) {
			aggregate, _, lead, auditee, _, finding := findingFixture(t, false, true, false)
			submitEvidenceForTest(t, &aggregate, auditee, finding, "evidence.pdf")
			verifyEvidenceForTest(t, &aggregate, lead, finding, testCase.outcome)
			evidence, _ := aggregate.latestEvidence(finding.FindingID)
			updatedFinding, _ := aggregate.latestFinding(finding.FindingID)
			if evidence.ReviewState != testCase.review || updatedFinding.State != testCase.finding || updatedFinding.ClosureBasis != testCase.basis || len(aggregate.VerificationDecisions) != 1 {
				t.Fatalf("outcome %s evidence=%s finding=%s basis=%s decisions=%d", testCase.outcome, evidence.ReviewState, updatedFinding.State, updatedFinding.ClosureBasis, len(aggregate.VerificationDecisions))
			}
		})
	}
	aggregate, _, lead, auditee, _, finding := findingFixture(t, false, true, false)
	submitEvidenceForTest(t, &aggregate, auditee, finding, "evidence-invalid.pdf")
	before := aggregate
	command := lifecycleCommandForTest(aggregate, OperationVerifyEvidence)
	command.FindingID, command.Outcome, command.CommentToAuditee, command.InternalCAANote = finding.FindingID, "INVALID", "public", "internal"
	if err := applyLifecycleCommand(&aggregate, command, lead, lifecycleTestNow); !errors.Is(err, ErrLifecycleChoiceInvalid) {
		t.Fatalf("invalid evidence outcome error = %v", err)
	}
	if len(aggregate.VerificationDecisions) != len(before.VerificationDecisions) || len(aggregate.EvidenceVersions) != len(before.EvidenceVersions) || aggregate.Revision != before.Revision {
		t.Fatal("invalid evidence outcome was not atomic")
	}
}

func TestEvidenceVerificationAndAuthorizedClosureAreSeparate(t *testing.T) {
	aggregate, _, lead, auditee, manager, finding := findingFixture(t, false, true, false)
	closeCommand := lifecycleCommandForTest(aggregate, OperationAuthorizedClose)
	closeCommand.FindingID, closeCommand.ReasonCode = finding.FindingID, "WRONG_BRANCH"
	if err := applyLifecycleCommand(&aggregate, closeCommand, manager, lifecycleTestNow); !errors.Is(err, ErrLifecycleTransition) {
		t.Fatalf("authorized close before evidence error = %v", err)
	}
	if len(aggregate.VerificationDecisions) != 0 || len(aggregate.EvidenceVersions) != 0 {
		t.Fatal("authorized close touched evidence state")
	}
	// The no-CAP/no-Evidence branch can only close through the separate Manager path.
	noFollowUp, _, _, _, manager, noFollowUpFinding := findingFixture(t, false, false, false)
	applyLifecycleTest(t, &noFollowUp, manager, OperationAuthorizedClose, func(command *CommandEnvelope) {
		command.FindingID = noFollowUpFinding.FindingID
		command.ReasonCode = "AUTHORIZED_WITHOUT_EVIDENCE"
	})
	closed, _ := noFollowUp.latestFinding(noFollowUpFinding.FindingID)
	if closed.State != FindingClosed || len(noFollowUp.VerificationDecisions) != 0 || len(noFollowUp.EvidenceVersions) != 0 {
		t.Fatalf("separate authorized close = state=%s verification=%d evidence=%d", closed.State, len(noFollowUp.VerificationDecisions), len(noFollowUp.EvidenceVersions))
	}
	_ = lead
	_ = auditee
}

func TestAuditeeProjectionIsOrganizationScoped(t *testing.T) {
	aggregate, _, lead, auditee, _, finding := findingFixture(t, true, false, false)
	submitCAP(t, &aggregate, auditee, finding, "private root cause")
	reviewCAPForTest(t, &aggregate, lead, finding, "ACCEPT")
	projection, err := ProjectAuditeeLifecycle(aggregate, auditee)
	if err != nil {
		t.Fatalf("auditee projection: %v", err)
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	public := string(encoded)
	for _, forbidden := range []string{"internalCaaNote", "actorSubjectId", "bindingId", "membershipSlot", "subjectId", "recommendationId", "recommendationDigest", "roleHistory"} {
		if strings.Contains(public, forbidden) {
			t.Fatalf("auditee projection contains forbidden field %q: %s", forbidden, public)
		}
	}
	if !strings.Contains(public, "commentToAuditee") || !strings.Contains(public, "nextAction") {
		t.Fatalf("auditee projection omitted public workflow fields: %s", public)
	}
	wrongOrganization := auditee
	wrongOrganization.OrganizationID = "AGA-DEMO-OTHER-ORG"
	if _, err := ProjectAuditeeLifecycle(aggregate, wrongOrganization); !errors.Is(err, ErrNeutralDenied) {
		t.Fatalf("wrong organization projection error = %v", err)
	}
	nonAuditee := lead
	if _, err := ProjectAuditeeLifecycle(aggregate, nonAuditee); !errors.Is(err, ErrNeutralDenied) {
		t.Fatalf("non-auditee projection error = %v", err)
	}
	if !strings.Contains(string(marshalLifecycle(ProjectCAALifecycle(aggregate, lead))), "internalCaaNote") {
		t.Fatal("CAA projection lost internal note field")
	}
}
