import type {
  Backend,
  BackendPrincipal,
  ChecklistAnswer,
  FindingView,
  PlanningItemView,
  PotentialFindingView,
} from "../../src/backend/backend";
import {
  createCanonicalFinding,
  FIXED_NOW,
  PRINCIPALS,
  submitEvidence,
  type BackendContractHarness,
} from "./backend-contract";

export const FULL_PLATFORM_SCENARIO_FAMILIES = [
  "routine-inspection-to-closure",
  "ad-hoc-planning-to-assignment",
  "checklist-and-potential-finding-authority",
  "cap-evidence-and-closure-authority",
  "preliminary-and-final-report-authority",
  "configuration-and-immutable-package-snapshot",
  "organization-and-platform-projections",
  "advisory-management-projections",
  "offline-causal-sync-and-session-boundaries",
  "advisory-draft-without-canonical-mutation",
] as const;

const REQUIRED_DENIALS = [
  "inspector-cannot-open-lead-potential-finding-queue",
  "manager-cannot-mutate-admin-configuration",
  "auditee-cannot-read-private-risk-projection",
  "offline-grant-cannot-cross-device-session",
] as const;

const REQUIRED_AUDIT_EVENT_TYPES = [
  "planning.intake_submitted",
  "PLANNING_BUDGET_APPROVED",
  "PLANNING_FORWARDED_FOR_FINAL_APPROVAL",
  "PLANNING_APPROVED",
  "PLANNING_RELEASED",
] as const;

export const REQUIRED_SCENARIO_PROOFS = [
  "routine-plan-released",
  "routine-finding-closed-by-evidence",
  "ad-hoc-intake-submitted",
  "ad-hoc-notice-withheld",
  "ad-hoc-plan-released",
  "lead-package-prepared",
  "executable-audit-materialized",
  "field-assignment-projected",
  "checklist-submitted-and-reopened",
  "potential-finding-returned",
  "potential-finding-dismissed",
  "potential-finding-converted",
  "observation-defaults-preserved",
  "cap-submitted",
  "cap-returned-for-revision",
  "cap-revised",
  "cap-accepted-without-closure",
  "evidence-uploaded-and-scan-clean",
  "evidence-not-closed",
  "evidence-partially-closed",
  "evidence-closed",
  "authorized-closure-separated",
  "preliminary-report-dm-gm-ed-issued",
  "final-report-dm-gm-ed-issued",
  "auditee-report-preview-scoped",
  "question-authored",
  "template-version-authored",
  "immutable-audit-package-snapshot",
  "organization-master-data-projected",
  "team-workload-projected",
  "calendar-projected",
  "communications-and-notifications-projected",
  "documents-projected",
  "reminders-projected",
  "settings-profile-updated",
  "users-and-roles-projected",
  "audit-log-projected",
  "management-risk-projected",
  "ssp-usoap-advisory-projections",
  "cap-effectiveness-advisory-only",
  "offline-checkout-projected",
  "offline-causal-replay",
  "offline-conflict-and-reentry",
  "offline-attachment-delivered",
  "assistant-draft-no-canonical-mutation",
] as const;

export interface FullPlatformTranscript {
  scenarioFamilies: readonly string[];
  scenarioProofs: readonly string[];
  entityIds: Record<string, readonly string[]>;
  revisions: Record<string, number | readonly number[]>;
  statuses: Record<string, string | readonly string[]>;
  owners: Record<string, string>;
  roles: readonly string[];
  organizationIds: readonly string[];
  versions: Record<string, number | readonly number[]>;
  auditEventTypes: readonly string[];
  notificationJobs: readonly string[];
  documentJobs: readonly string[];
  denials: readonly string[];
  dashboardProjections: Record<string, number | string | readonly string[]>;
}

interface ScenarioState {
  harness: BackendContractHarness;
  canonicalFinding: FindingView | null;
  potentialFindings: PotentialFindingView[];
  routinePlan: PlanningItemView | null;
  releasedPlan: PlanningItemView | null;
  denials: string[];
  scenarioProofs: string[];
  syncStatuses: string[];
}

function backendFor(state: ScenarioState, principal: BackendPrincipal): Backend {
  return state.harness.backendFor(principal);
}

function prove(
  state: ScenarioState,
  proof: (typeof REQUIRED_SCENARIO_PROOFS)[number],
  condition: boolean,
): void {
  if (!condition) {
    throw new Error(`Required scenario proof failed: ${proof}`);
  }
  state.scenarioProofs.push(proof);
}

async function expectDenied(label: string, command: () => Promise<unknown>, state: ScenarioState) {
  try {
    await command();
  } catch (cause) {
    const message = cause instanceof Error ? cause.message : String(cause);
    if (!/forbidden|authority|unavailable|role|Auditee|Lead Inspector|grant|denied/i.test(message)) {
      throw cause;
    }
    state.denials.push(label);
    return;
  }
  throw new Error(`Expected denial was not enforced: ${label}`);
}

async function createPotentialFinding(
  state: ScenarioState,
  suffix: string,
  questionId: string,
  answer: ChecklistAnswer,
): Promise<PotentialFindingView> {
  const inspector = backendFor(state, PRINCIPALS.inspector);
  const response = await inspector.inspections.upsertChecklistResponse({
    operationId: `OP-FULL-${suffix}-RESPONSE`,
    responseId: `RESP-FULL-${suffix}`,
    auditId: "AUD-2026-001",
    questionId,
    expectedResponseRevision: null,
    answer,
    comment: `${suffix} requires an exact Lead Inspector decision.`,
  });
  const potential = await inspector.potentialFindings.create({
    operationId: `OP-FULL-${suffix}-PF`,
    auditId: "AUD-2026-001",
    questionId,
    checklistResponseId: response.id,
    expectedChecklistResponseRevision: response.revision,
    title: `${suffix} Potential Finding`,
    description: `${suffix} branch for full-platform transcript parity.`,
    requiredComment: response.comment,
    inspectionAttachmentIds: [],
  });
  state.potentialFindings.push(potential);
  return potential;
}

async function routineInspectionToClosure(state: ScenarioState): Promise<void> {
  const finance = backendFor(state, PRINCIPALS.finance);
  const initialPlan = (await finance.planning.list({ limit: 20 })).items.find(
    ({ id }) => id === "PLAN-2026-CAB-001",
  );
  if (!initialPlan) {
    throw new Error("The canonical routine Planning item is unavailable.");
  }
  const budget = await finance.planning.decide({
    operationId: "OP-FULL-ROUTINE-BUDGET",
    planningItemId: initialPlan.id,
    expectedPlanningRevision: initialPlan.revision,
    decision: "APPROVE_BUDGET",
    reason: "Approve the configured routine Cabin Inspection budget.",
  });
  const forwarded = await backendFor(state, PRINCIPALS.gm).planning.decide({
    operationId: "OP-FULL-ROUTINE-GM-FORWARD",
    planningItemId: budget.id,
    expectedPlanningRevision: budget.revision,
    decision: "FORWARD_FOR_FINAL_APPROVAL",
    reason: "Forward the routine Cabin Inspection plan.",
  });
  const approved = await backendFor(state, PRINCIPALS.executiveDirector).planning.decide({
    operationId: "OP-FULL-ROUTINE-ED-APPROVE",
    planningItemId: forwarded.id,
    expectedPlanningRevision: forwarded.revision,
    decision: "APPROVE_PLAN",
    reason: "Approve the exact routine plan revision.",
  });
  state.routinePlan = await backendFor(state, PRINCIPALS.gm).planning.decide({
    operationId: "OP-FULL-ROUTINE-GM-RELEASE",
    planningItemId: approved.id,
    expectedPlanningRevision: approved.revision,
    decision: "RELEASE_PLAN",
    reason: "Release the announced routine Cabin Inspection.",
  });
  prove(
    state,
    "routine-plan-released",
    state.routinePlan.status === "RELEASED" &&
      state.routinePlan.currentOwnerRole === "manager" &&
      state.routinePlan.revision === 5,
  );

  const finding = await createCanonicalFinding(state.harness);
  state.canonicalFinding = finding;
  state.potentialFindings.push(await backendFor(
    state,
    PRINCIPALS.leadInspector,
  ).potentialFindings.get({ potentialFindingId: "PF-2026-001" }));
  const auditee = backendFor(state, PRINCIPALS.auditee);
  const lead = backendFor(state, PRINCIPALS.leadInspector);
  const firstCap = await auditee.caps.submit({
    operationId: "OP-FULL-CAP-SUBMIT-R1",
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    rootCause: "Initial PBE serviceability reconciliation gap.",
    correctiveAction: "Service the affected PBE and reconcile the cabin defect record.",
    preventiveAction: "Introduce a supervisor sampling check.",
    responsiblePerson: "Fly Namibia Cabin Safety Manager",
    targetCompletionDate: "2026-07-15",
    commentToCaa: "Initial CAP submitted for exact review.",
  });
  prove(state, "cap-submitted", firstCap.capRevision === 1 && firstCap.capStatus === "SUBMITTED");
  const returnedCap = await lead.caps.review({
    operationId: "OP-FULL-CAP-RETURN-R1",
    capRevisionId: firstCap.capRevisionId,
    expectedCapRevision: firstCap.capRevision,
    findingId: finding.id,
    expectedFindingRevision: firstCap.findingRevision,
    decision: "REQUEST_MORE_INFORMATION",
    commentToAuditee: "Clarify the preventive-action verification sequence.",
    internalCaaNote: "Revision 1 lacks a measurable verification step.",
  });
  prove(
    state,
    "cap-returned-for-revision",
    returnedCap.capStatus === "MORE_INFORMATION_REQUESTED" &&
      returnedCap.findingStatus !== "CLOSED",
  );
  const secondCap = await auditee.caps.submit({
    operationId: "OP-FULL-CAP-SUBMIT-R2",
    findingId: finding.id,
    expectedFindingRevision: returnedCap.findingRevision,
    rootCause: "PBE position records were not reconciled with the deferred defect list.",
    correctiveAction: "Service the PBE and reconcile its exact cabin position.",
    preventiveAction: "Add monthly supervisor sampling with recorded verification.",
    responsiblePerson: "Fly Namibia Cabin Safety Manager",
    targetCompletionDate: "2026-07-20",
    commentToCaa: "Revised CAP adds the requested verification sequence.",
  });
  prove(state, "cap-revised", secondCap.capRevision === 2);
  const acceptedCap = await lead.caps.review({
    operationId: "OP-FULL-CAP-ACCEPT-R2",
    capRevisionId: secondCap.capRevisionId,
    expectedCapRevision: secondCap.capRevision,
    findingId: finding.id,
    expectedFindingRevision: secondCap.findingRevision,
    decision: "ACCEPT",
    commentToAuditee: "Revised CAP accepted; submit the exact Evidence.",
    internalCaaNote: "Acceptance does not close the Finding.",
  });
  prove(
    state,
    "cap-accepted-without-closure",
    acceptedCap.capStatus === "ACCEPTED" &&
      acceptedCap.findingStatus === "EVIDENCE_REQUIRED",
  );

  const versionOne = await submitEvidence(
    state.harness,
    "FULL-V1",
    "Fly_Namibia_PBE_Initial_Record.pdf",
  );
  prove(state, "evidence-uploaded-and-scan-clean", versionOne.scanState === "CLEAN");
  let current = await lead.findings.get({ findingId: finding.id });
  const notClosed = await lead.evidence.review({
    operationId: "OP-FULL-EVIDENCE-NOT-CLOSE-V1",
    evidenceVersionId: versionOne.id,
    expectedEvidenceVersionRevision: versionOne.revision,
    findingId: finding.id,
    expectedFindingRevision: current.revision,
    decision: "NOT_CLOSE",
    commentToAuditee: "The initial record does not establish serviceability.",
    internalCaaNote: "Version 1 is rejected on its exact content.",
  });
  prove(state, "evidence-not-closed", notClosed.findingStatus !== "CLOSED");

  const versionTwo = await submitEvidence(
    state.harness,
    "FULL-V2",
    "Fly_Namibia_PBE_Serviceability_Record.pdf",
  );
  current = await lead.findings.get({ findingId: finding.id });
  const partiallyClosed = await lead.evidence.review({
    operationId: "OP-FULL-EVIDENCE-PARTIAL-V2",
    evidenceVersionId: versionTwo.id,
    expectedEvidenceVersionRevision: versionTwo.revision,
    findingId: finding.id,
    expectedFindingRevision: current.revision,
    decision: "PARTIALLY_CLOSE",
    commentToAuditee: "Serviceability accepted; cabin position confirmation remains required.",
    internalCaaNote: "Version 2 is intentionally partial.",
  });
  prove(state, "evidence-partially-closed", partiallyClosed.findingStatus !== "CLOSED");

  const versionThree = await submitEvidence(
    state.harness,
    "FULL-V3",
    "Fly_Namibia_PBE_Position_Confirmation.pdf",
  );
  current = await lead.findings.get({ findingId: finding.id });
  const closed = await lead.evidence.review({
    operationId: "OP-FULL-EVIDENCE-CLOSE-V3",
    evidenceVersionId: versionThree.id,
    expectedEvidenceVersionRevision: versionThree.revision,
    findingId: finding.id,
    expectedFindingRevision: current.revision,
    decision: "CLOSE",
    commentToAuditee: "The exact latest Evidence version is accepted and verified.",
    internalCaaNote: "Version 3 completes the configured verification.",
  });
  prove(state, "evidence-closed", closed.findingStatus === "CLOSED");
  prove(
    state,
    "routine-finding-closed-by-evidence",
    (await lead.findings.get({ findingId: finding.id })).closureBasis === "EVIDENCE_VERIFIED",
  );
}

async function adHocPlanningToAssignment(state: ScenarioState): Promise<void> {
  const manager = backendFor(state, PRINCIPALS.manager);
  const draft = await manager.planningIntake.getDraft({ draftId: "PLAN-DRAFT-2026-001" });
  const saved = await manager.planningIntake.saveDraft({
    idempotencyKey: "IDEM-FULL-AD-HOC-DRAFT",
    expectedRevision: draft.revision,
    draftId: draft.id,
    values: {
      organizationId: "ORG-FLY-NAMIBIA",
      organizationName: "Fly Namibia",
      applicationType: "Continued Surveillance",
      domain: "Cabin Safety",
      inspectionCategory: "Ad Hoc / Unannounced",
      noticePolicy: "WITHHELD",
      purpose: "Risk-triggered unannounced cabin inspection.",
      triggerType: "Advisory risk review",
      riskCategory: "Cabin Safety",
      plannedDate: "2026-12-10",
      mode: "On-site",
      location: "Windhoek",
      templateVersionId: "CTV-CABIN-1",
      scope: "Cabin safety and emergency equipment.",
      requestedBudget: 0,
      currency: "USD",
    },
  });
  const submitted = await manager.planningIntake.submit({
    idempotencyKey: "IDEM-FULL-AD-HOC-SUBMIT",
    expectedRevision: saved.revision,
    draftId: saved.id,
    planningItemId: "PLAN-2026-AD-HOC-001",
  });
  prove(
    state,
    "ad-hoc-intake-submitted",
    submitted.planningItem.status === "FINANCE_REVIEW" &&
      submitted.draft.submittedPlanningItemId === submitted.planningItem.id,
  );
  prove(
    state,
    "ad-hoc-notice-withheld",
    submitted.draft.inspectionCategory === "Ad Hoc / Unannounced" &&
      submitted.draft.noticePolicy === "WITHHELD",
  );
  const budget = await backendFor(state, PRINCIPALS.finance).planning.decide({
    operationId: "OP-FULL-AD-HOC-BUDGET",
    planningItemId: submitted.planningItem.id,
    expectedPlanningRevision: submitted.planningItem.revision,
    decision: "APPROVE_BUDGET",
    reason: "No external budget is required.",
  });
  const forwarded = await backendFor(state, PRINCIPALS.gm).planning.decide({
    operationId: "OP-FULL-AD-HOC-GM-FORWARD",
    planningItemId: budget.id,
    expectedPlanningRevision: budget.revision,
    decision: "FORWARD_FOR_FINAL_APPROVAL",
    reason: "Forward the withheld-notice plan.",
  });
  const approved = await backendFor(state, PRINCIPALS.executiveDirector).planning.decide({
    operationId: "OP-FULL-AD-HOC-EXECUTIVE-APPROVE",
    planningItemId: forwarded.id,
    expectedPlanningRevision: forwarded.revision,
    decision: "APPROVE_PLAN",
    reason: "Approve the exact unannounced plan version.",
  });
  state.releasedPlan = await backendFor(state, PRINCIPALS.gm).planning.decide({
    operationId: "OP-FULL-AD-HOC-GM-RELEASE",
    planningItemId: approved.id,
    expectedPlanningRevision: approved.revision,
    decision: "RELEASE_PLAN",
    reason: "Release to Department preparation without Auditee notice.",
  });
  prove(
    state,
    "ad-hoc-plan-released",
    state.releasedPlan.status === "RELEASED" &&
      state.releasedPlan.currentOwnerRole === "manager",
  );

  const packageDraft = await manager.packageDrafts.get({
    packageDraftId: "PKG-AUD-2026-001-CABIN",
  });
  const preparedPackage = await manager.packageDrafts.save({
    idempotencyKey: "IDEM-FULL-AD-HOC-PACKAGE-PREP",
    expectedRevision: packageDraft.revision,
    packageDraftId: packageDraft.id,
    riskFocus: ["PBE serviceability", "Unannounced cabin sampling"],
  });
  prove(
    state,
    "lead-package-prepared",
    preparedPackage.revision === packageDraft.revision + 1 &&
      preparedPackage.questions.length > 0,
  );

  const executable = await backendFor(state, PRINCIPALS.inspector).inspections.getPackage({
    packageId: "PKG-CAB-2026-001",
  });
  prove(
    state,
    "executable-audit-materialized",
    executable.auditId === "AUD-2026-001" &&
      executable.packageVersion === 1 &&
      executable.questions.length === 6,
  );
  const [leadCandidates, assignments] = await Promise.all([
    manager.teams.list({ role: "leadInspector" }),
    manager.assignments.list({ limit: 20 }),
  ]);
  prove(
    state,
    "field-assignment-projected",
    leadCandidates.items.some(({ subjectId }) => subjectId === "USR-LEAD-CANER") &&
      assignments.items.some(({ auditId }) => auditId === executable.auditId) &&
      executable.questions.some(({ assignedInspectorUserIds }) =>
        assignedInspectorUserIds.length > 0),
  );
}

async function potentialFindingAuthority(state: ScenarioState): Promise<void> {
  const lead = backendFor(state, PRINCIPALS.leadInspector);
  const returned = await createPotentialFinding(
    state,
    "RETURN",
    "CAB-LAV-001",
    "NON_COMPLIANT",
  );
  state.potentialFindings[1] = (await lead.potentialFindings.decide({
    operationId: "OP-FULL-PF-RETURN",
    potentialFindingId: returned.id,
    expectedPotentialFindingRevision: returned.revision,
    decision: "RETURN",
    reason: "Inspector must clarify the exact Evidence basis.",
  })).potentialFinding;
  prove(state, "potential-finding-returned", state.potentialFindings[1]?.status === "RETURNED");

  const dismissed = await createPotentialFinding(
    state,
    "DISMISS",
    "CAB-PAX-SEAT-001",
    "NON_COMPLIANT",
  );
  state.potentialFindings[2] = (await lead.potentialFindings.decide({
    operationId: "OP-FULL-PF-DISMISS",
    potentialFindingId: dismissed.id,
    expectedPotentialFindingRevision: dismissed.revision,
    decision: "DISMISS",
    reason: "The configured observation does not establish non-compliance.",
  })).potentialFinding;
  prove(state, "potential-finding-dismissed", state.potentialFindings[2]?.status === "DISMISSED");

  const observation = await createPotentialFinding(
    state,
    "OBSERVATION",
    "CAB-VID-CREW-SEAT-001",
    "OBSERVATION",
  );
  const convertedObservation = await lead.potentialFindings.decide({
    operationId: "OP-FULL-PF-OBSERVATION-CONVERT",
    potentialFindingId: observation.id,
    expectedPotentialFindingRevision: observation.revision,
    decision: "CONVERT",
    severity: "OBSERVATION",
    capRequired: false,
    evidenceRequired: true,
    dueDate: null,
  });
  state.potentialFindings[3] = convertedObservation.potentialFinding;
  prove(
    state,
    "potential-finding-converted",
    convertedObservation.potentialFinding.status === "CONVERTED" &&
      convertedObservation.finding !== null,
  );
  prove(
    state,
    "observation-defaults-preserved",
    convertedObservation.finding?.severity === "OBSERVATION" &&
      convertedObservation.finding.capRequired === false &&
      convertedObservation.finding.evidenceRequired === true &&
      convertedObservation.finding.dueDate === null &&
      convertedObservation.finding.status === "EVIDENCE_REQUIRED",
  );

  const fixtureLead = backendFor(state, PRINCIPALS.leadInspector);
  const packageView = await fixtureLead.inspections.getPackage({ packageId: "PKG-CAB-2026-001" });
  for (const packageQuestion of packageView.questions) {
    if (packageQuestion.currentResponse) continue;
    await fixtureLead.inspections.upsertChecklistResponse({
      operationId: `OP-FULL-CHECKLIST-COMPLETE-${packageQuestion.id}`,
      responseId: `RESP-FULL-CHECKLIST-${packageQuestion.id}`,
      auditId: packageView.auditId,
      questionId: packageQuestion.id,
      expectedResponseRevision: null,
      answer: "COMPLIANT",
      comment: "",
    });
  }
  const submittedChecklist = await backendFor(state, PRINCIPALS.inspector)
    .inspections.submitChecklist({
      operationId: "OP-FULL-CHECKLIST-SUBMIT",
      auditId: "AUD-2026-001",
      expectedChecklistRevision: 1,
    });
  const reopenedChecklist = await lead.inspections.reopenChecklist({
    operationId: "OP-FULL-CHECKLIST-REOPEN",
    auditId: "AUD-2026-001",
    expectedChecklistRevision: submittedChecklist.checklistRevision,
    reason: "Reopen the exact checklist for continued configured sampling.",
  });
  prove(
    state,
    "checklist-submitted-and-reopened",
    submittedChecklist.checklistStatus === "SUBMITTED" &&
      reopenedChecklist.checklistStatus === "IN_PROGRESS" &&
      reopenedChecklist.checklistRevision === submittedChecklist.checklistRevision + 1,
  );

  const authorized = await backendFor(state, PRINCIPALS.manager).findings.authorizedClose({
    operationId: "OP-FULL-AUTHORIZED-CLOSE-OBSERVATION",
    findingId: convertedObservation.finding!.id,
    expectedFindingRevision: convertedObservation.finding!.revision,
    reason: "Exercise the separately authorized and audit-recorded closure path.",
  });
  prove(
    state,
    "authorized-closure-separated",
    authorized.status === "CLOSED" && authorized.closureBasis === "AUTHORIZED",
  );

  await expectDenied(
    "inspector-cannot-open-lead-potential-finding-queue",
    () => backendFor(state, PRINCIPALS.inspector).potentialFindings.list({
      status: "PENDING_LEAD_REVIEW",
    }),
    state,
  );
}

async function reportAuthority(state: ScenarioState): Promise<void> {
  const manager = backendFor(state, PRINCIPALS.manager);
  const preliminary = await manager.reports.getVersion({ reportVersionId: "PR-2026-018-V1" });
  const gmReview = await manager.reports.decide({
    operationId: "OP-FULL-PRELIMINARY-MANAGER-FORWARD",
    reportVersionId: preliminary.reportVersionId,
    expectedReportVersionRevision: preliminary.revision,
    decision: "FORWARD",
    reason: "Preliminary report package is complete.",
  });
  const executiveReview = await backendFor(state, PRINCIPALS.gm).reports.decide({
    operationId: "OP-FULL-PRELIMINARY-GM-FORWARD",
    reportVersionId: gmReview.reportVersionId,
    expectedReportVersionRevision: gmReview.revision,
    decision: "FORWARD",
    reason: "Forward the exact Preliminary Report version.",
  });
  const issuedPreliminary = await backendFor(state, PRINCIPALS.executiveDirector).reports.decide({
    operationId: "OP-FULL-PRELIMINARY-ISSUE",
    reportVersionId: executiveReview.reportVersionId,
    expectedReportVersionRevision: executiveReview.revision,
    decision: "ISSUE_AND_LOCK",
    reason: "Issue the exact Preliminary Report version.",
  });
  prove(
    state,
    "preliminary-report-dm-gm-ed-issued",
    issuedPreliminary.status === "LOCKED" && issuedPreliminary.revision === 4,
  );

  const final = await manager.reports.getVersion({
    reportVersionId: "RPT-CAB-2026-001-V1",
  });
  const finalGmReview = await manager.reports.decide({
    operationId: "OP-FULL-FINAL-MANAGER-FORWARD",
    reportVersionId: final.reportVersionId,
    expectedReportVersionRevision: final.revision,
    decision: "FORWARD",
    reason: "Final report package is complete.",
  });
  const finalExecutiveReview = await backendFor(state, PRINCIPALS.gm).reports.decide({
    operationId: "OP-FULL-FINAL-GM-FORWARD",
    reportVersionId: finalGmReview.reportVersionId,
    expectedReportVersionRevision: finalGmReview.revision,
    decision: "FORWARD",
    reason: "Forward the exact Final Report version.",
  });
  const issuedFinal = await backendFor(state, PRINCIPALS.executiveDirector).reports.decide({
    operationId: "OP-FULL-FINAL-ISSUE",
    reportVersionId: finalExecutiveReview.reportVersionId,
    expectedReportVersionRevision: finalExecutiveReview.revision,
    decision: "ISSUE_AND_LOCK",
    reason: "Issue the exact Final Report version.",
  });
  prove(
    state,
    "final-report-dm-gm-ed-issued",
    issuedFinal.status === "LOCKED" && issuedFinal.revision === 4,
  );

  const auditee = backendFor(state, PRINCIPALS.auditee);
  const [releasedPreliminary, releasedFinal] = await Promise.all([
    auditee.auditeeReports.listReleased({ kind: "PRELIMINARY" }),
    auditee.auditeeReports.listReleased({ kind: "FINAL" }),
  ]);
  const preliminaryPreview = await auditee.auditeeReports.getReleased({
    reportVersionId: issuedPreliminary.reportVersionId,
  });
  const finalPreview = await auditee.auditeeReports.getReleased({
    reportVersionId: issuedFinal.reportVersionId,
  });
  prove(
    state,
    "auditee-report-preview-scoped",
    releasedPreliminary.items.some(({ reportVersionId }) =>
      reportVersionId === issuedPreliminary.reportVersionId) &&
      releasedFinal.items.some(({ reportVersionId }) =>
        reportVersionId === issuedFinal.reportVersionId) &&
      preliminaryPreview.organizationId === "ORG-FLY-NAMIBIA" &&
      finalPreview.organizationId === "ORG-FLY-NAMIBIA",
  );
}

async function configurationSnapshot(state: ScenarioState): Promise<void> {
  const admin = backendFor(state, PRINCIPALS.admin);
  const version = await admin.configuration.getChecklistTemplateVersion({
    templateVersionId: "CTV-CABIN-1",
  });
  const packageBeforeAuthoring = await admin.adminWorkspace.getInspectionPackage({
    packageId: "PKG-CAB-2026-001",
  });
  const question = await admin.adminWorkspace.createQuestion({
    idempotencyKey: "IDEM-FULL-CONFIG-QUESTION",
    expectedRevision: null,
    prompt: "Does the unannounced cabin sample retain its exact immutable Evidence basis?",
    configuredReference: "Configured Cabin Inspection reference — immutable package proof",
    expectedEvidence: "Exact package snapshot and source Template version",
  });
  prove(
    state,
    "question-authored",
    question.id === "Q-ADMIN-2026-007" && question.revision === 1,
  );
  const template = await admin.adminWorkspace.getTemplate({
    templateId: "TPL-CABIN-2026",
  });
  const draft = await admin.adminWorkspace.createDraft({
    idempotencyKey: "IDEM-FULL-CONFIG-DRAFT",
    expectedRevision: template.revision,
    templateId: template.id,
    changeReason: "Prove immutable package isolation from later Template authoring.",
  });
  const added = await admin.adminWorkspace.addDraftQuestion({
    idempotencyKey: "IDEM-FULL-CONFIG-ADD-QUESTION",
    expectedRevision: draft.revision,
    templateId: template.id,
    draftVersionId: draft.id,
    questionId: question.id,
  });
  const moved = await admin.adminWorkspace.moveDraftQuestion({
    idempotencyKey: "IDEM-FULL-CONFIG-MOVE-QUESTION",
    expectedRevision: added.revision,
    templateId: template.id,
    draftVersionId: added.id,
    questionId: question.id,
    direction: "UP",
  });
  prove(
    state,
    "template-version-authored",
    moved.status === "DRAFT" &&
      moved.questionIds.includes(question.id) &&
      moved.revision === 3,
  );
  const packageAfterAuthoring = await admin.adminWorkspace.getInspectionPackage({
    packageId: "PKG-CAB-2026-001",
  });
  if (
    version.version !== 1 ||
    version.questions.length !== 6 ||
    packageAfterAuthoring.questionIds.length !== 6 ||
    packageAfterAuthoring.questionIds.some(
      (id) => !version.questions.some((publishedQuestion) => publishedQuestion.id === id),
    ) ||
    packageAfterAuthoring.questionIds.includes(question.id) ||
    JSON.stringify(packageBeforeAuthoring) !== JSON.stringify(packageAfterAuthoring)
  ) {
    throw new Error("Published Template and executable package snapshot drifted.");
  }
  prove(state, "immutable-audit-package-snapshot", true);
}

async function organizationAndPlatformProjections(state: ScenarioState): Promise<void> {
  const inspector = backendFor(state, PRINCIPALS.inspector);
  const auditee = backendFor(state, PRINCIPALS.auditee);
  const sent = await inspector.communications.send({
    idempotencyKey: "IDEM-FULL-PUBLIC-COMMUNICATION",
    expectedRevision: null,
    organizationId: "ORG-FLY-NAMIBIA",
    subject: "Full-platform Evidence follow-up",
    body: "Public organization-scoped follow-up.",
    audience: "AUDITEE",
  });
  await inspector.communications.send({
    idempotencyKey: "IDEM-FULL-INTERNAL-COMMUNICATION",
    expectedRevision: null,
    organizationId: "ORG-FLY-NAMIBIA",
    subject: "Internal CAA Note",
    body: "Private authority deliberation.",
    audience: "CAA",
  });
  const visible = await auditee.communications.list({ organizationId: "ORG-FLY-NAMIBIA" });
  if (
    !visible.items.some(({ id }) => id === sent.id) ||
    JSON.stringify(visible).includes("Private authority deliberation.")
  ) {
    throw new Error("Internal CAA Note leaked to the Auditee projection.");
  }
  const notifications = await auditee.notifications.list({});
  const notification = notifications.items.find(({ title, body }) =>
    title === "New CAA communication" && body.includes(sent.subject));
  if (!notification) {
    throw new Error("The public communication did not create its Auditee notification.");
  }
  const readNotification = await auditee.notifications.markRead({
    idempotencyKey: "IDEM-FULL-NOTIFICATION-READ",
    expectedRevision: notification.revision,
    notificationId: notification.id,
  });
  prove(
    state,
    "communications-and-notifications-projected",
    readNotification.readAt !== null && readNotification.revision === notification.revision + 1,
  );

  const organizations = await auditee.organizations.list({});
  if (organizations.items.length !== 1 || organizations.items[0]?.id !== "ORG-FLY-NAMIBIA") {
    throw new Error("Auditee organization isolation drifted.");
  }
  const admin = backendFor(state, PRINCIPALS.admin);
  const masterData = await admin.adminWorkspace.listOrganizations({
    search: "",
    organizationType: "",
    status: "",
    scope: "",
  });
  prove(
    state,
    "organization-master-data-projected",
    masterData.items.some(({ id }) => id === "ORG-FLY-NAMIBIA") &&
      masterData.items.some(({ id }) => id === "ORG-SKYCARGO"),
  );

  const manager = backendFor(state, PRINCIPALS.manager);
  const teamPage = await manager.teams.listAuditTeams({ limit: 20 });
  const exactTeam = await manager.teams.openAuditTeam({ auditId: "AUD-2026-001" });
  prove(
    state,
    "team-workload-projected",
    teamPage.items.some(({ auditId }) => auditId === exactTeam.auditId) &&
      exactTeam.leadInspector.subjectId === "USR-LEAD-CANER" &&
      exactTeam.assignments.length === 6,
  );

  const calendar = await inspector.calendar.list({});
  const calendarItem = calendar.items.find(({ auditId }) => auditId === "AUD-2026-001");
  const openedCalendarItem = calendarItem
    ? await inspector.calendar.openItem({ calendarItemId: calendarItem.id })
    : null;
  prove(
    state,
    "calendar-projected",
    openedCalendarItem?.auditId === "AUD-2026-001" &&
      openedCalendarItem.organizationId === "ORG-FLY-NAMIBIA",
  );

  const auditeeDocuments = await auditee.documents.list({
    organizationId: "ORG-FLY-NAMIBIA",
  });
  const evidenceDocument = auditeeDocuments.items.find(
    ({ id }) => id === "EV-CAB-2026-001-V3",
  );
  const openedDocument = evidenceDocument
    ? await auditee.documents.open({ documentId: evidenceDocument.id })
    : null;
  prove(
    state,
    "documents-projected",
    openedDocument?.organizationId === "ORG-FLY-NAMIBIA" &&
      openedDocument.kind === "EVIDENCE",
  );

  const reminders = await admin.configuration.listReminderRules({});
  prove(
    state,
    "reminders-projected",
    reminders.items.map(({ offsetDays }) => offsetDays).join(",") === "30,15,7,0,-1",
  );

  const profile = await manager.profiles.getMine({});
  const updatedProfile = await manager.profiles.updateMine({
    idempotencyKey: "IDEM-FULL-PROFILE-UPDATE",
    expectedRevision: profile.revision,
    displayName: "Nora Department Manager",
  });
  prove(
    state,
    "settings-profile-updated",
    updatedProfile.displayName === "Nora Department Manager" &&
      updatedProfile.revision === profile.revision + 1,
  );

  const directory = await admin.adminWorkspace.listAccessDirectory({
    search: "",
  });
  prove(
    state,
    "users-and-roles-projected",
    directory.items.some(({ roles }) => roles.includes("inspector")) &&
      directory.items.some(({ roles }) => roles.includes("manager")) &&
      directory.items.some(({ roles }) => roles.includes("auditee")),
  );

  const auditLog = await admin.adminWorkspace.listAuditEvents({});
  prove(
    state,
    "audit-log-projected",
    auditLog.items.some(({ action }) => action === "planning.intake_submitted") &&
      auditLog.items.some(({ action }) => action === "evidence.reviewed"),
  );

  await expectDenied(
    "manager-cannot-mutate-admin-configuration",
    () => manager.adminWorkspace.createQuestion({
      idempotencyKey: "IDEM-FULL-DENIED-ADMIN-QUESTION",
      expectedRevision: null,
      prompt: "This must not be created.",
      configuredReference: "Denied",
      expectedEvidence: "Denied",
    }),
    state,
  );
}

async function advisoryManagementProjections(state: ScenarioState): Promise<void> {
  const manager = backendFor(state, PRINCIPALS.manager);
  const before = await manager.risk.getManagementProjection({});
  const riskOverview = await manager.risk.getOverview({
    organizationId: "ORG-FLY-NAMIBIA",
  });
  prove(
    state,
    "management-risk-projected",
    before.findings.some(({ findingId }) => findingId === state.canonicalFinding?.id) &&
      riskOverview.organizationId === "ORG-FLY-NAMIBIA",
  );
  const ssp = await manager.administration.getScreenProjection({
    screenId: "manager-ssp-nasp",
  });
  const usoap = await manager.administration.getScreenProjection({
    screenId: "manager-usoap-readiness",
  });
  const [sspAction, usoapAction] = await Promise.all([
    manager.administration.invokeVisibleAction({
      screenId: ssp.screenId,
      actionId: ssp.visibleActions[0]!.id,
    }),
    manager.administration.invokeVisibleAction({
      screenId: usoap.screenId,
      actionId: usoap.visibleActions[0]!.id,
    }),
  ]);
  prove(
    state,
    "ssp-usoap-advisory-projections",
    sspAction.effect.type === "localProjection" &&
      usoapAction.effect.type === "localProjection",
  );
  const after = await manager.risk.getManagementProjection({});
  prove(
    state,
    "cap-effectiveness-advisory-only",
    before.capEffectiveness.some(({ findingId, state: effectivenessState }) =>
      findingId === state.canonicalFinding?.id &&
      effectivenessState === "PENDING_POST_CLOSURE_VERIFICATION") &&
      JSON.stringify(before) === JSON.stringify(after),
  );
  await expectDenied(
    "auditee-cannot-read-private-risk-projection",
    () => backendFor(state, PRINCIPALS.auditee).risk.getOverview({
      organizationId: "ORG-FLY-NAMIBIA",
    }),
    state,
  );
}

async function offlineCausalSync(state: ScenarioState): Promise<void> {
  const inspector = backendFor(state, PRINCIPALS.inspector);
  const checkout = await inspector.inspections.checkout({
    operationId: "OP-FULL-OFFLINE-CHECKOUT",
    packageId: "PKG-CAB-2026-001",
    expectedPackageVersion: 1,
    deviceInstanceId: "DEVICE-CANDIDATE-001",
  });
  prove(
    state,
    "offline-checkout-projected",
    checkout.offlineGrant.grantId === "GRANT-CANDIDATE-001" &&
      checkout.offlineGrant.subjectId.length > 0 &&
      checkout.offlineGrant.deviceInstanceId === "DEVICE-CANDIDATE-001" &&
      checkout.offlineGrant.assignmentScope.questionIds.includes("CAB-EMEQ-PBE-001"),
  );

  const attachmentBody = new TextEncoder().encode(
    "%PDF-1.7\n1 0 obj\n<</Type/Catalog/Label(FullPlatformOfflineAttachment)>>\nendobj\n%%EOF\n",
  );
  const attachmentDigest = await crypto.subtle.digest("SHA-256", attachmentBody);
  const attachmentSha256 = `sha256:${Array.from(
    new Uint8Array(attachmentDigest),
    (byte) => byte.toString(16).padStart(2, "0"),
  ).join("")}`;
  const attachmentRegistration = await inspector.sync.pushOperation({
    operation: {
      operationId: "OP-FULL-OFFLINE-ATTACHMENT-REGISTER",
      protocolVersion: checkout.offlineGrant.protocolVersion,
      offlineGrantId: checkout.offlineGrant.grantId,
      packageId: checkout.offlineGrant.packageId,
      packageVersion: checkout.offlineGrant.packageVersion,
      entityId: "ATT-FULL-OFFLINE-001",
      commandType: "REGISTER_INSPECTION_ATTACHMENT",
      baseRevision: null,
      deviceInstanceId: checkout.offlineGrant.deviceInstanceId,
      clientOccurredAt: FIXED_NOW,
      payload: {
        auditId: "AUD-2026-001",
        checklistResponseId: "RESP-CAB-EMEQ-PBE-001",
        potentialFindingOperationId: null,
        fileName: "full-platform-offline-attachment.pdf",
        mediaType: "application/pdf",
        byteSize: attachmentBody.byteLength,
        sha256: attachmentSha256,
      },
    },
  });
  if (
    attachmentRegistration.status !== "accepted" ||
    !attachmentRegistration.authoritativeEntityId
  ) {
    throw new Error("Inspection Attachment registration was not accepted.");
  }
  const authoritativeAttachmentId = attachmentRegistration.authoritativeEntityId;
  const attachmentUpload = await inspector.inspectionAttachments.beginUpload({
    operationId: "OP-FULL-OFFLINE-ATTACHMENT-BEGIN",
    inspectionAttachmentId: authoritativeAttachmentId,
    packageId: checkout.offlineGrant.packageId,
    byteSize: attachmentBody.byteLength,
    sha256: attachmentSha256,
    fileName: "full-platform-offline-attachment.pdf",
    declaredMediaType: "application/pdf",
  });
  if (inspector.mode === "http") {
    const response = await fetch(attachmentUpload.uploadUrl, {
      method: "PUT",
      headers: attachmentUpload.requiredHeaders,
      body: attachmentBody,
    });
    if (!response.ok) {
      throw new Error(`Inspection Attachment upload failed with HTTP ${response.status}.`);
    }
  }
  const attachment = await inspector.inspectionAttachments.completeUpload({
    operationId: "OP-FULL-OFFLINE-ATTACHMENT-COMPLETE",
    uploadId: attachmentUpload.uploadId,
    sha256: attachmentSha256,
    byteSize: attachmentBody.byteLength,
  });
  prove(
    state,
    "offline-attachment-delivered",
    attachment.inspectionAttachmentId === authoritativeAttachmentId &&
      attachment.uploadState === "UPLOADED",
  );

  const request = {
    operation: {
      operationId: "OP-FULL-OFFLINE-SYNC",
      protocolVersion: checkout.offlineGrant.protocolVersion,
      offlineGrantId: checkout.offlineGrant.grantId,
      packageId: checkout.offlineGrant.packageId,
      packageVersion: checkout.offlineGrant.packageVersion,
      entityId: "RESP-FULL-OFFLINE",
      commandType: "UPSERT_CHECKLIST_RESPONSE" as const,
      baseRevision: null,
      deviceInstanceId: checkout.offlineGrant.deviceInstanceId,
      clientOccurredAt: FIXED_NOW,
      payload: {
        auditId: "AUD-2026-001",
        questionId: "CAB-COCKPIT-GEN-001",
        answer: "COMPLIANT" as const,
        comment: "",
      },
    },
  };
  const accepted = await inspector.sync.pushOperation(request);
  const replay = await inspector.sync.pushOperation(request);
  if (accepted.status !== "accepted" || JSON.stringify(replay) !== JSON.stringify(accepted)) {
    throw new Error("Offline causal replay drifted.");
  }
  prove(state, "offline-causal-replay", replay.status === "accepted");
  const conflict = await inspector.sync.pushOperation({
    operation: {
      ...request.operation,
      operationId: "OP-FULL-OFFLINE-SYNC-STALE",
      payload: {
        ...request.operation.payload,
        answer: "OBSERVATION" as const,
        comment: "Stale offline observation must retain its conflict-safe local draft.",
      },
    },
  });
  const forbidden = await inspector.sync.pushOperation({
    operation: {
      ...request.operation,
      operationId: "OP-FULL-OFFLINE-SYNC-WRONG-DEVICE",
      entityId: "RESP-FULL-OFFLINE-WRONG-DEVICE",
      deviceInstanceId: "DEVICE-NOT-GRANTED",
    },
  });
  const reentered = await inspector.sync.pushOperation({
    operation: {
      ...request.operation,
      operationId: "OP-FULL-OFFLINE-SYNC-REENTRY",
      baseRevision: accepted.authoritativeRevision,
      payload: {
        ...request.operation.payload,
        answer: "NON_COMPLIANT" as const,
        comment: "Explicitly re-entered after reviewing the authoritative revision.",
      },
    },
  });
  prove(
    state,
    "offline-conflict-and-reentry",
    conflict.status === "conflict" &&
      reentered.status === "accepted" &&
      reentered.authoritativeRevision === 2,
  );
  state.syncStatuses.push(
    accepted.status,
    replay.status,
    conflict.status,
    reentered.status,
    forbidden.status,
  );
  const pulled = await inspector.sync.pull({
    packageId: checkout.offlineGrant.packageId,
    offlineGrantId: checkout.offlineGrant.grantId,
    cursor: null,
  });
  if (!pulled.changes.some(({ kind }) => kind === "checklist_response")) {
    throw new Error("Offline pull omitted the accepted checklist response.");
  }
  await expectDenied(
    "offline-grant-cannot-cross-device-session",
    () => inspector.sync.pull({
      packageId: "PKG-CAB-2026-001",
      offlineGrantId: "GRANT-NOT-AUTHORIZED",
      cursor: null,
    }),
    state,
  );
}

async function advisoryDraftWithoutMutation(state: ScenarioState): Promise<void> {
  const otherInspector = backendFor(state, PRINCIPALS.otherInspector);
  const before = await otherInspector.findings.get({ findingId: "FND-SKYCARGO-2026-099" });
  const draft = await otherInspector.assistantDrafts.createDraft({
    findingId: before.id,
    prompt: "Draft an Evidence request only.",
  });
  const after = await otherInspector.findings.get({ findingId: before.id });
  if (
    !draft.advisoryOnly ||
    draft.canCreateFinding ||
    draft.canSetSeverity ||
    draft.canCloseFinding ||
    JSON.stringify(before) !== JSON.stringify(after)
  ) {
    throw new Error("Advisory assistant changed canonical Finding authority.");
  }
  prove(state, "assistant-draft-no-canonical-mutation", true);
}

export async function runFullPlatformScenarios(
  harness: BackendContractHarness,
): Promise<FullPlatformTranscript> {
  const state: ScenarioState = {
    harness,
    canonicalFinding: null,
    potentialFindings: [],
    routinePlan: null,
    releasedPlan: null,
    denials: [],
    scenarioProofs: [],
    syncStatuses: [],
  };
  await routineInspectionToClosure(state);
  await adHocPlanningToAssignment(state);
  await potentialFindingAuthority(state);
  const lead = backendFor(state, PRINCIPALS.leadInspector);
  const closed = await lead.findings.get({ findingId: state.canonicalFinding!.id });
  const evidence = await lead.evidence.listVersions({ findingId: closed.id });
  const caps = await lead.caps.listRevisions({ findingId: closed.id });
  await reportAuthority(state);
  await configurationSnapshot(state);
  await organizationAndPlatformProjections(state);
  await advisoryManagementProjections(state);
  await offlineCausalSync(state);
  await advisoryDraftWithoutMutation(state);

  const admin = backendFor(state, PRINCIPALS.admin);
  const auditee = backendFor(state, PRINCIPALS.auditee);
  const manager = backendFor(state, PRINCIPALS.manager);
  const [auditEvents, notifications, documents, dashboard, management] = await Promise.all([
    admin.auditTrail.list({
      entityType: "SURVEILLANCE_PLAN",
      entityId: state.releasedPlan!.id,
      limit: 100,
    }),
    auditee.notifications.list({}),
    auditee.documents.list({}),
    manager.dashboards.getManagerProjection({}),
    manager.risk.getManagementProjection({}),
  ]);
  const preliminaryV0 = await manager.reports.getVersion({ reportVersionId: "PR-2026-018-V0" });
  const preliminaryV1 = await manager.reports.getVersion({ reportVersionId: "PR-2026-018-V1" });
  const finalV1 = await manager.reports.getVersion({ reportVersionId: "RPT-CAB-2026-001-V1" });
  const observation = state.potentialFindings[3]!;

  const raw: FullPlatformTranscript = {
    scenarioFamilies: FULL_PLATFORM_SCENARIO_FAMILIES,
    scenarioProofs: REQUIRED_SCENARIO_PROOFS.filter((proof) =>
      state.scenarioProofs.includes(proof)),
    entityIds: {
      audit: ["AUD-2026-001"],
      planning: [state.routinePlan!.id, state.releasedPlan!.id],
      potentialFindings: state.potentialFindings.map(({ id }) => id),
      findings: [closed.id, observation.convertedFindingId!],
      capRevisions: caps.items.map(({ id }) => id),
      evidenceVersions: evidence.map(({ id }) => id),
      reports: [preliminaryV0.reportVersionId, preliminaryV1.reportVersionId, finalV1.reportVersionId],
    },
    revisions: {
      finding: closed.revision,
      planning: [state.routinePlan!.revision, state.releasedPlan!.revision],
      potentialFindings: state.potentialFindings.map(({ revision }) => revision),
      evidence: evidence.map(({ revision }) => revision),
      reports: [preliminaryV0.revision, preliminaryV1.revision, finalV1.revision],
    },
    statuses: {
      finding: closed.status,
      planning: [state.routinePlan!.status, state.releasedPlan!.status],
      potentialFindings: state.potentialFindings.map(({ status }) => status),
      evidenceReview: evidence.map(({ reviewState }) => reviewState),
      reports: [preliminaryV0.status, preliminaryV1.status, finalV1.status],
      sync: state.syncStatuses,
    },
    owners: {
      finding: closed.currentOwnerType,
      planning: state.releasedPlan!.currentOwnerRole,
      observation: "CAA",
    },
    roles: [
      "inspector",
      "leadInspector",
      "manager",
      "finance",
      "gm",
      "executiveDirector",
      "auditee",
      "admin",
    ],
    organizationIds: ["ORG-FLY-NAMIBIA", "ORG-SKYCARGO"],
    versions: {
      package: 1,
      cap: caps.items.map(({ revision }) => revision),
      evidence: evidence.map(({ version }) => version),
      reports: [preliminaryV0.version, preliminaryV1.version, finalV1.version],
    },
    auditEventTypes: auditEvents.items.map(({ action }) => action),
    notificationJobs: notifications.items
      .filter(({ title }) => title === "New CAA communication")
      .map(({ id }) => id),
    documentJobs: documents.items.map(({ id }) => id).sort(),
    denials: state.denials,
    dashboardProjections: {
      openFindings: dashboard.openFindings,
      closedFindings: dashboard.closedFindings,
      overdueFindings: dashboard.overdueFindings,
      pendingCapReviews: dashboard.pendingCapReviews,
      pendingEvidenceReviews: dashboard.pendingEvidenceReviews,
      managementFindingIds: management.findings.map(({ findingId }) => findingId).sort(),
    },
  };
  return normalizeFullPlatformTranscript(raw);
}

export function normalizeFullPlatformTranscript(value: FullPlatformTranscript): FullPlatformTranscript {
  const serialized = JSON.stringify(value);
  if (/internalCaaNote|enforcementDeliberation|privateRisk|uiOnlyState/i.test(serialized)) {
    throw new Error("Forbidden private or UI-only field entered the normalized transcript.");
  }
  if (value.scenarioFamilies.length !== 10) {
    throw new Error("The normalized transcript must contain exactly ten scenario families.");
  }
  if (
    value.scenarioProofs.length !== REQUIRED_SCENARIO_PROOFS.length ||
    REQUIRED_SCENARIO_PROOFS.some((proof) => !value.scenarioProofs.includes(proof))
  ) {
    throw new Error("The normalized transcript is missing a required scenario proof.");
  }
  if (
    value.denials.length !== REQUIRED_DENIALS.length ||
    REQUIRED_DENIALS.some((denial) => !value.denials.includes(denial))
  ) {
    throw new Error("The normalized transcript is missing a required denial.");
  }
  if (REQUIRED_AUDIT_EVENT_TYPES.some((eventType) => !value.auditEventTypes.includes(eventType))) {
    throw new Error("The normalized transcript is missing a required audit event.");
  }
  return JSON.parse(serialized) as FullPlatformTranscript;
}

// Frozen after the first successful MockBackend RED→GREEN transcript and then
// required byte-for-byte from the independently orchestrated HttpBackend run.
export const FULL_PLATFORM_EXPECTED_TRANSCRIPT: FullPlatformTranscript = {
  scenarioFamilies: FULL_PLATFORM_SCENARIO_FAMILIES,
  scenarioProofs: REQUIRED_SCENARIO_PROOFS,
  entityIds: {
    audit: ["AUD-2026-001"],
    planning: ["PLAN-2026-CAB-001", "PLAN-2026-AD-HOC-001"],
    potentialFindings: ["PF-2026-001", "PF-2026-002", "PF-2026-003", "PF-2026-004"],
    findings: ["FND-CAB-2026-001", "FND-CAB-2026-002"],
    capRevisions: ["CAP-CAB-2026-001-R1", "CAP-CAB-2026-001-R2"],
    evidenceVersions: [
      "EV-CAB-2026-001-V1",
      "EV-CAB-2026-001-V2",
      "EV-CAB-2026-001-V3",
    ],
    reports: ["PR-2026-018-V0", "PR-2026-018-V1", "RPT-CAB-2026-001-V1"],
  },
  revisions: {
    finding: 14,
    planning: [5, 5],
    potentialFindings: [2, 2, 2, 2],
    evidence: [3, 3, 3],
    reports: [1, 4, 4],
  },
  statuses: {
    finding: "CLOSED",
    planning: ["RELEASED", "RELEASED"],
    potentialFindings: ["CONVERTED", "RETURNED", "DISMISSED", "CONVERTED"],
    evidenceReview: ["REJECTED", "PARTIALLY_ACCEPTED", "ACCEPTED"],
    reports: ["RETURNED", "LOCKED", "LOCKED"],
    sync: ["accepted", "accepted", "conflict", "accepted", "forbidden"],
  },
  owners: {
    finding: "CAA",
    planning: "manager",
    observation: "CAA",
  },
  roles: [
    "inspector",
    "leadInspector",
    "manager",
    "finance",
    "gm",
    "executiveDirector",
    "auditee",
    "admin",
  ],
  organizationIds: ["ORG-FLY-NAMIBIA", "ORG-SKYCARGO"],
  versions: {
    package: 1,
    cap: [1, 2],
    evidence: [1, 2, 3],
    reports: [0, 1, 1],
  },
  auditEventTypes: [
    "planning.intake_submitted",
    "PLANNING_BUDGET_APPROVED",
    "PLANNING_FORWARDED_FOR_FINAL_APPROVAL",
    "PLANNING_APPROVED",
    "PLANNING_RELEASED",
  ],
  notificationJobs: ["notification-candidate-001"],
  documentJobs: [
    "EV-CAB-2026-001-V1",
    "EV-CAB-2026-001-V2",
    "EV-CAB-2026-001-V3",
    "PR-2026-018-V1",
    "RPT-CAB-2026-001-V1",
  ],
  denials: REQUIRED_DENIALS,
  dashboardProjections: {
    openFindings: 1,
    closedFindings: 2,
    overdueFindings: 1,
    pendingCapReviews: 0,
    pendingEvidenceReviews: 0,
    managementFindingIds: [
      "FND-CAB-2026-001",
      "FND-CAB-2026-002",
      "FND-SKYCARGO-2026-099",
    ],
  },
};
