import type { FindingView } from "../backend/backend";
import type { createMockBackendRuntime } from "./create-mock-backend";
import { completeMockChecklist } from "./test-checklist-fixtures";

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

async function seedPotentialFinding(runtime: MockRuntime) {
  const inspector = runtime.backendForRole("inspector");
  const packageView = await inspector.inspections.getPackage({ packageId: "PKG-CAB-2026-001" });
  const response = await inspector.inspections.upsertChecklistResponse({
    operationId: "OP-VISUAL-RESPONSE",
    responseId: "RESP-CAB-EMEQ-PBE-001",
    auditId: "AUD-2026-001",
    questionId: "CAB-EMEQ-PBE-001",
    expectedResponseRevision: null,
    answer: "NON_COMPLIANT",
    comment: "PBE serviceability and accessibility could not be confirmed.",
  });
  const potentialFinding = await inspector.potentialFindings.create({
    operationId: "OP-VISUAL-PF",
    auditId: "AUD-2026-001",
    questionId: "CAB-EMEQ-PBE-001",
    checklistResponseId: response.id,
    expectedChecklistResponseRevision: response.revision,
    title: "PBE serviceability and accessibility not confirmed",
    description: "The configured cabin check could not confirm PBE serviceability.",
    requiredComment: response.comment,
    inspectionAttachmentIds: [],
  });
  await completeMockChecklist(runtime, packageView.id);
  await inspector.inspections.submitChecklist({
    operationId: "OP-VISUAL-CHECKLIST",
    auditId: packageView.auditId,
    expectedChecklistRevision: packageView.checklistRevision,
  });
  return potentialFinding;
}

async function seedFinding(runtime: MockRuntime): Promise<FindingView> {
  try {
    return await runtime.backendForRole("leadInspector").findings.get({ findingId: "FND-CAB-2026-001" });
  } catch {
    // The visual runtime is persisted across route loads; seed only when absent.
  }
  const potentialFinding = await seedPotentialFinding(runtime);
  const result = await runtime.backendForRole("leadInspector").potentialFindings.decide({
    operationId: "OP-VISUAL-CONVERT",
    potentialFindingId: potentialFinding.id,
    expectedPotentialFindingRevision: potentialFinding.revision,
    decision: "CONVERT",
    severity: "LEVEL_1_CRITICAL",
    capRequired: true,
    evidenceRequired: true,
    dueDate: "2026-07-15",
  });
  if (!result.finding) throw new Error("Visual fixture did not create the canonical Finding.");
  return result.finding;
}

async function seedSubmittedCap(runtime: MockRuntime): Promise<FindingView> {
  const finding = await seedFinding(runtime);
  await runtime.backendForRole("auditee").caps.submit({
    operationId: "OP-VISUAL-CAP-FINDING-DETAIL",
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    rootCause: "The equipment register and serviceability record were maintained separately.",
    correctiveAction: "Reconcile the sampled position against the current serviceability register.",
    preventiveAction: "Add a single-register review before each cabin inspection.",
    responsiblePerson: "Fly Namibia Cabin Safety Manager",
    targetCompletionDate: "2026-07-15",
    commentToCaa: "CAP submitted for CAA review.",
  });
  return runtime.backendForRole("leadInspector").findings.get({ findingId: finding.id });
}

async function seedCapReview(runtime: MockRuntime): Promise<void> {
  const lead = runtime.backendForRole("leadInspector");
  const auditee = runtime.backendForRole("auditee");
  let finding = await seedFinding(runtime);
  const first = await auditee.caps.submit({
    operationId: "OP-VISUAL-CAP-R1",
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    rootCause: "Initial root cause retained for immutable history.",
    correctiveAction: "Initial corrective action.",
    preventiveAction: "Initial preventive action.",
    responsiblePerson: "Fly Namibia Cabin Safety Manager",
    targetCompletionDate: "2026-07-15",
    commentToCaa: "Initial CAP submitted for CAA review.",
  });
  finding = await lead.findings.get({ findingId: finding.id });
  await lead.caps.review({
    operationId: "OP-VISUAL-CAP-R1-REVIEW",
    capRevisionId: first.capRevisionId,
    expectedCapRevision: first.capRevision,
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    decision: "REQUEST_MORE_INFORMATION",
    commentToAuditee: "Clarify how PBE position records will be sampled.",
    internalCaaNote: "Internal CAA note for revision 1.",
  });
  finding = await lead.findings.get({ findingId: finding.id });
  await auditee.caps.submit({
    operationId: "OP-VISUAL-CAP-R2",
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    rootCause: "Revised root cause with record reconciliation.",
    correctiveAction: "Replace affected PBE and update the cabin defect record.",
    preventiveAction: "Add supervisor review and monthly sampling.",
    responsiblePerson: "Fly Namibia Cabin Safety Manager",
    targetCompletionDate: "2026-07-20",
    commentToCaa: "Revised CAP submitted for CAA review.",
  });
}

async function submitEvidenceVersion(
  runtime: MockRuntime,
  finding: FindingView,
  fileName: string,
): Promise<FindingView> {
  const auditee = runtime.backendForRole("auditee");
  const upload = await auditee.evidence.beginUpload({
    operationId: `OP-VISUAL-EVIDENCE-BEGIN-${fileName}`,
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    fileName,
    declaredMediaType: "application/pdf",
    byteSize: 12,
    sha256: `sha256:${fileName}`,
  });
  await auditee.evidence.completeUpload({
    operationId: `OP-VISUAL-EVIDENCE-COMPLETE-${fileName}`,
    uploadId: upload.uploadId,
    sha256: `sha256:${fileName}`,
    byteSize: 12,
  });
  return runtime.backendForRole("leadInspector").findings.get({ findingId: finding.id });
}

async function seedEvidenceReview(runtime: MockRuntime): Promise<void> {
  const lead = runtime.backendForRole("leadInspector");
  const auditee = runtime.backendForRole("auditee");
  let finding = await seedFinding(runtime);
  if ((await lead.evidence.listVersions({ findingId: finding.id })).length >= 2) return;
  const cap = await auditee.caps.submit({
    operationId: "OP-VISUAL-CAP",
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    rootCause: "Root cause ready for evidence.",
    correctiveAction: "Replace affected PBE.",
    preventiveAction: "Monthly PBE sampling.",
    responsiblePerson: "Fly Namibia Cabin Safety Manager",
    targetCompletionDate: "2026-07-15",
    commentToCaa: "CAP submitted for CAA review.",
  });
  finding = await lead.findings.get({ findingId: finding.id });
  await lead.caps.review({
    operationId: "OP-VISUAL-CAP-ACCEPT",
    capRevisionId: cap.capRevisionId,
    expectedCapRevision: cap.capRevision,
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    decision: "ACCEPT",
    commentToAuditee: "CAP accepted. Submit evidence.",
    internalCaaNote: "Evidence verification remains required.",
  });
  finding = await lead.findings.get({ findingId: finding.id });
  finding = await submitEvidenceVersion(
    runtime,
    finding,
    "Fly_Namibia_PBE_Serviceability_Record_CAB-2026-001.pdf",
  );
  const firstVersion = (await lead.evidence.listVersions({ findingId: finding.id })).at(-1);
  if (!firstVersion) throw new Error("Visual fixture did not create the first Evidence version.");
  await lead.evidence.review({
    operationId: "OP-VISUAL-EVIDENCE-V1-REVIEW",
    evidenceVersionId: firstVersion.id,
    expectedEvidenceVersionRevision: firstVersion.revision,
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    decision: "PARTIALLY_CLOSE",
    commentToAuditee: "Serviceability accepted; provide cabin position confirmation.",
    internalCaaNote: "Version 1 does not verify accessibility.",
  });
  finding = await lead.findings.get({ findingId: finding.id });
  await submitEvidenceVersion(
    runtime,
    finding,
    "Fly_Namibia_PBE_Position_Confirmation_CAB-2026-001.pdf",
  );
}

async function advancePlanningToGeneralManager(runtime: MockRuntime) {
  const finance = runtime.backendForRole("finance");
  const plans = await finance.planning.list({ limit: 20 });
  const plan = plans.items.find((item) => item.id === "PLAN-2026-CAB-001") ?? plans.items[0];
  if (!plan) throw new Error("Visual fixture requires the canonical Planning item.");
  if (plan.status !== "FINANCE_REVIEW") return plan;
  return finance.planning.decide({
    operationId: `OP-VISUAL-FINANCE-${plan.id}-${plan.revision}`,
    planningItemId: plan.id,
    expectedPlanningRevision: plan.revision,
    decision: "APPROVE_BUDGET",
    reason: "Finance approved the exact visual Planning revision.",
  });
}

async function advancePlanningToExecutiveDirector(runtime: MockRuntime) {
  const atGeneralManager = await advancePlanningToGeneralManager(runtime);
  if (atGeneralManager.status !== "GM_REVIEW") return atGeneralManager;
  return runtime.backendForRole("gm").planning.decide({
    operationId: `OP-VISUAL-GM-${atGeneralManager.id}-${atGeneralManager.revision}`,
    planningItemId: atGeneralManager.id,
    expectedPlanningRevision: atGeneralManager.revision,
    decision: "FORWARD_FOR_FINAL_APPROVAL",
    reason: "General Manager forwarded the exact visual Planning revision.",
  });
}

async function advancePreliminaryReportToGeneralManager(runtime: MockRuntime) {
  const manager = runtime.backendForRole("manager");
  const report = await manager.reports.getVersion({ reportVersionId: "PR-2026-018-V1" });
  if (report.status !== "DEPARTMENT_REVIEW") return report;
  return manager.reports.decide({
    operationId: `OP-VISUAL-MANAGER-${report.reportVersionId}-${report.revision}`,
    reportVersionId: report.reportVersionId,
    expectedReportVersionRevision: report.revision,
    decision: "FORWARD",
    reason: "Department Manager forwarded the exact visual Preliminary Report version.",
  });
}

async function lockAuditeeReports(runtime: MockRuntime) {
  const atGeneralManager = await advancePreliminaryReportToGeneralManager(runtime);
  const atExecutive = atGeneralManager.status === "GM_REVIEW"
    ? await runtime.backendForRole("gm").reports.decide({
        operationId: `OP-VISUAL-GM-${atGeneralManager.reportVersionId}-${atGeneralManager.revision}`,
        reportVersionId: atGeneralManager.reportVersionId,
        expectedReportVersionRevision: atGeneralManager.revision,
        decision: "FORWARD",
        reason: "General Manager forwarded the exact visual Preliminary Report version.",
      })
    : atGeneralManager;
  if (atExecutive.status === "EXECUTIVE_DIRECTOR_REVIEW") {
    await runtime.backendForRole("executiveDirector").reports.decide({
      operationId: `OP-VISUAL-EXEC-${atExecutive.reportVersionId}-${atExecutive.revision}`,
      reportVersionId: atExecutive.reportVersionId,
      expectedReportVersionRevision: atExecutive.revision,
      decision: "ISSUE_AND_LOCK",
      reason: "Executive Director issued and locked the exact visual Preliminary Report version.",
    });
  }
  const finalReport = await runtime.backendForRole("executiveDirector").reports.getVersion({ reportVersionId: "RPT-CAB-2026-001-V1" });
  if (finalReport.status === "EXECUTIVE_DIRECTOR_REVIEW") {
    await runtime.backendForRole("executiveDirector").reports.decide({
      operationId: `OP-VISUAL-EXEC-${finalReport.reportVersionId}-${finalReport.revision}`,
      reportVersionId: finalReport.reportVersionId,
      expectedReportVersionRevision: finalReport.revision,
      decision: "ISSUE_AND_LOCK",
      reason: "Executive Director issued and locked the exact visual Final Report version.",
    });
  }
}

export async function seedVisualRuntimeForPath(runtime: MockRuntime, pathname: string): Promise<void> {
  if ([
    "/auditee/preliminary-reports",
    "/auditee/final-reports",
    "/auditee/reports/RPT-CAB-2026-001",
    "/auditee/documents",
  ].includes(pathname)) {
    if (pathname === "/auditee/documents") await seedEvidenceReview(runtime);
    await lockAuditeeReports(runtime);
    return;
  }
  if (pathname === "/auditee/messages") {
    await runtime.backendForRole("inspector").communications.send({
      expectedRevision: null,
      idempotencyKey: "MSG-VISUAL-AUDITEE-1",
      organizationId: "ORG-FLY-NAMIBIA",
      subject: "Inspection coordination update",
      body: "The proposed inspection date is ready for your confirmation.",
      audience: "AUDITEE",
    });
    return;
  }
  if (pathname === "/general-manager/planning") {
    await advancePlanningToGeneralManager(runtime);
    return;
  }
  if (pathname === "/general-manager/report-approvals") {
    await advancePreliminaryReportToGeneralManager(runtime);
    return;
  }
  if (pathname === "/executive-director/planning") {
    await advancePlanningToExecutiveDirector(runtime);
    return;
  }
  if (pathname === "/inspector/profile") {
    const profiles = runtime.backendForRole("inspector").profiles;
    if (!profiles) throw new Error("Visual fixture requires Inspector profiles.");
    const profile = await profiles.getMine({});
    await profiles.updateMine({
      expectedRevision: profile.revision,
      idempotencyKey: "PROFILE-VISUAL-AYLIN",
      displayName: "Aylin Sezer",
    });
    return;
  }
  if (pathname === "/lead-inspector/lead-review") {
    await seedPotentialFinding(runtime);
    return;
  }
  if (pathname === "/lead-inspector/preliminary-reports/PR-2026-018") {
    await seedFinding(runtime);
    return;
  }
  if (
    pathname === "/inspector/findings" ||
    pathname === "/inspector/findings/FND-CAB-2026-001" ||
    pathname === "/inspector/closure-reports/CR-CAB-2026-001" ||
    pathname === "/inspector/assistant"
  ) {
    await seedSubmittedCap(runtime);
    return;
  }
  if (pathname === "/lead-inspector/cap-review/FND-CAB-2026-001") {
    await seedCapReview(runtime);
    return;
  }
  if (
    pathname === "/department-manager/findings-review" ||
    pathname === "/department-manager/cap-monitoring" ||
    pathname === "/department-manager/evidence/FND-CAB-2026-001" ||
    pathname === "/department-manager/findings/FND-CAB-2026-001/closure-review" ||
    pathname === "/department-manager/organizations/ORG-FLY-NAMIBIA"
  ) {
    await seedEvidenceReview(runtime);
    return;
  }
  if (pathname === "/auditee/service-provider-cap") {
    await seedFinding(runtime);
  }
}
