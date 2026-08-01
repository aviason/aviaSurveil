// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { MemoryRouter } from "react-router-dom";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import { ScenarioProvider } from "../../app/scenario-context";
import type { FindingView } from "../../backend/backend";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { completeMockChecklist } from "../../mock/test-checklist-fixtures";
import { EvidenceReviewPage } from "./evidence-review-page";

afterEach(cleanup);

type MockRuntime = ReturnType<typeof createMockBackendRuntime>;

async function seedFinding(runtime: MockRuntime): Promise<FindingView> {
  const inspector = runtime.backendForRole("inspector");
  const packageView = await inspector.inspections.getPackage({ packageId: "PKG-CAB-2026-001" });
  const response = await inspector.inspections.upsertChecklistResponse({
    operationId: "OP-TEST-RESPONSE",
    responseId: "RESP-CAB-EMEQ-PBE-001",
    auditId: "AUD-2026-001",
    questionId: "CAB-EMEQ-PBE-001",
    expectedResponseRevision: null,
    answer: "NON_COMPLIANT",
    comment: "PBE serviceability and accessibility could not be confirmed.",
  });
  const potentialFinding = await inspector.potentialFindings.create({
    operationId: "OP-TEST-PF",
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
    operationId: "OP-TEST-CHECKLIST",
    auditId: packageView.auditId,
    expectedChecklistRevision: packageView.checklistRevision,
  });
  const result = await runtime.backendForRole("leadInspector").potentialFindings.decide({
    operationId: "OP-TEST-CONVERT",
    potentialFindingId: potentialFinding.id,
    expectedPotentialFindingRevision: potentialFinding.revision,
    decision: "CONVERT",
    severity: "LEVEL_1_CRITICAL",
    capRequired: true,
    evidenceRequired: true,
    dueDate: "2026-07-15",
  });
  if (!result.finding) throw new Error("Expected seeded Finding.");
  return result.finding;
}

async function submitEvidenceVersion(runtime: MockRuntime, finding: FindingView, fileName: string) {
  const auditee = runtime.backendForRole("auditee");
  const upload = await auditee.evidence.beginUpload({
    operationId: `OP-TEST-EVIDENCE-BEGIN-${fileName}`,
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    fileName,
    declaredMediaType: "application/pdf",
    byteSize: 12,
    sha256: `sha256:${fileName}`,
  });
  await auditee.evidence.completeUpload({
    operationId: `OP-TEST-EVIDENCE-COMPLETE-${fileName}`,
    uploadId: upload.uploadId,
    sha256: `sha256:${fileName}`,
    byteSize: 12,
  });
  return runtime.backendForRole("leadInspector").findings.get({ findingId: finding.id });
}

async function seedEvidenceVersions(runtime: MockRuntime) {
  const lead = runtime.backendForRole("leadInspector");
  const auditee = runtime.backendForRole("auditee");
  let finding = await seedFinding(runtime);
  const cap = await auditee.caps.submit({
    operationId: "OP-TEST-CAP",
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
    operationId: "OP-TEST-CAP-ACCEPT",
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
  if (!firstVersion) throw new Error("Expected first Evidence version.");
  await lead.evidence.review({
    operationId: "OP-TEST-EVIDENCE-V1-REVIEW",
    evidenceVersionId: firstVersion.id,
    expectedEvidenceVersionRevision: firstVersion.revision,
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    decision: "PARTIALLY_CLOSE",
    commentToAuditee: "Serviceability accepted; provide cabin position confirmation.",
    internalCaaNote: "Version 1 does not verify accessibility.",
  });
  finding = await lead.findings.get({ findingId: finding.id });
  finding = await submitEvidenceVersion(
    runtime,
    finding,
    "Fly_Namibia_PBE_Position_Confirmation_CAB-2026-001.pdf",
  );
  return finding;
}

function renderPage(runtime: MockRuntime) {
  render(
    <AppProviders
      runtime={{
        backend: runtime.backend,
        backendForRole: runtime.backendForRole,
        buildProfile: "demo",
        environmentLabel: "test",
        identityMode: "demo-role-switch",
        subjectId: "USR-MANAGER-NORA",
      }}
    >
      <ScenarioProvider>
        <MemoryRouter initialEntries={["/department-manager/evidence/FND-CAB-2026-001"]}>
          <EvidenceReviewPage />
        </MemoryRouter>
      </ScenarioProvider>
    </AppProviders>,
  );
}

describe("EvidenceReviewPage", () => {
  it("direct-loads the Manager-owned Inspection Evidence projection and records exact partial-close audit identity", async () => {
    const runtime = createMockBackendRuntime();
    await seedEvidenceVersions(runtime);
    const managerEvidence = runtime.backendForRole("manager").evidence;
    const listSpy = vi.spyOn(managerEvidence, "listVersions");
    const reviewSpy = vi.spyOn(managerEvidence, "review");

    renderPage(runtime);

    const page = await screen.findByTestId("manager-inspection-evidence-page");
    expect(within(page).getByTestId("reviewing-evidence-version")).toHaveTextContent("Version 2");
    const findingRow = within(page).getByRole("row", { name: /PBE serviceability and accessibility not confirmed/i });
    expect(findingRow).toHaveTextContent("Lead Inspector · USR-LEAD-CANER");
    expect(findingRow).toHaveTextContent("CAA reviews Evidence");
    expect(findingRow).toHaveTextContent("PENDING_CAA_REVIEW");
    expect(within(page).getByText("Evidence Waiting Review", { selector: ".evidence-root-attention span" }).parentElement).toHaveTextContent("1 finding");
    expect(screen.getByTestId("application-shell")).toHaveAttribute("data-active-role", "manager");
    expect(document.querySelector(".evidence-root-page")).not.toHaveTextContent("Lead Inspector workspace");
    expect(listSpy).toHaveBeenCalledWith({ findingId: "FND-CAB-2026-001" });
    const history = screen.getByRole("list", { name: "Evidence version history" });
    expect(within(history).getAllByTestId("evidence-history-row")).toHaveLength(2);
    expect(within(history).getByText("Fly_Namibia_PBE_Serviceability_Record_CAB-2026-001.pdf")).toBeVisible();
    expect(within(history).getByText("PARTIALLY_ACCEPTED")).toBeVisible();
    expect(within(history).getByText("Fly_Namibia_PBE_Position_Confirmation_CAB-2026-001.pdf")).toBeVisible();

    await userEvent.click(within(page).getByRole("button", { name: "View Evidence version EV-CAB-2026-001-V1" }));
    expect(within(page).getByTestId("reviewing-evidence-version")).toHaveTextContent("Version 1");
    expect(within(page).getByRole("button", { name: "Record Evidence review" })).toBeDisabled();
    expect(within(page).getByRole("button", { name: "Record Evidence review" })).toHaveAttribute(
      "title",
      "Evidence version EV-CAB-2026-001-V1 is historical; only exact latest version EV-CAB-2026-001-V2 can be reviewed.",
    );
    await userEvent.click(within(page).getByRole("button", { name: "Review Evidence version EV-CAB-2026-001-V2" }));
    expect(within(page).getByTestId("reviewing-evidence-version")).toHaveTextContent("Version 2");

    await userEvent.selectOptions(within(page).getByLabelText("Evidence review decision"), "PARTIALLY_CLOSE");
    await userEvent.click(within(page).getByRole("button", { name: "Record Evidence review" }));
    expect(await within(page).findByRole("alert")).toHaveTextContent("Comment to Auditee is required");
    await userEvent.type(within(page).getByLabelText("Comment to Auditee"), "Serviceability is accepted; position confirmation remains incomplete.");
    await userEvent.type(within(page).getByLabelText("Internal CAA Note"), "Department review preserves version 2 and keeps the Finding open.");
    await userEvent.click(within(page).getByRole("button", { name: "Record Evidence review" }));
    expect(await within(page).findByTestId("finding-status")).toHaveTextContent("EVIDENCE_MORE_INFORMATION_REQUESTED");
    expect(within(page).getByTestId("closure-state")).toHaveTextContent("Finding remains open");
    expect(findingRow).toHaveTextContent("Auditee · ORG-FLY-NAMIBIA");
    expect(findingRow).toHaveTextContent("Auditee to provide remaining Evidence or information");
    expect(findingRow).toHaveTextContent("EVIDENCE_MORE_INFORMATION_REQUESTED");
    expect(within(page).getByText("Evidence Waiting Review", { selector: ".evidence-root-attention span" }).parentElement).toHaveTextContent("0 findings");
    expect(reviewSpy).toHaveBeenCalledWith(expect.objectContaining({
      evidenceVersionId: "EV-CAB-2026-001-V2",
      expectedEvidenceVersionRevision: 2,
      decision: "PARTIALLY_CLOSE",
      operationId: "OP-EVIDENCE-EV-CAB-2026-001-V2-R2-PARTIALLY_CLOSE",
    }));
    expect(within(page).getByRole("button", { name: "Record Evidence review" })).toBeDisabled();
    expect(within(page).getByRole("button", { name: "Record Evidence review" })).toHaveAttribute(
      "title",
      "Evidence version EV-CAB-2026-001-V2 is PARTIALLY_ACCEPTED and is no longer pending CAA review.",
    );
    const events = await runtime.backendForRole("manager").auditTrail.list({ entityType: "finding", entityId: "FND-CAB-2026-001" });
    expect(events.items.at(-1)).toMatchObject({
      actorSubjectId: "USR-MANAGER-NORA",
      actorRole: "manager",
      action: "evidence.reviewed",
      entityType: "finding",
      entityId: "FND-CAB-2026-001",
      beforeStatus: "PENDING_CAA_REVIEW",
      afterStatus: "EVIDENCE_MORE_INFORMATION_REQUESTED",
      reason: "Serviceability is accepted; position confirmation remains incomplete.",
      entityRevision: 9,
    });
    expect(within(page).getByRole("link", { name: "Open Department CAP Closure Review for FND-CAB-2026-001" })).toHaveAttribute(
      "href",
      "/department-manager/findings/FND-CAB-2026-001/closure-review",
    );
  });

  it("allows the same decision for a later exact Evidence version without replaying the prior operation", async () => {
    const runtime = createMockBackendRuntime();
    let finding = await seedEvidenceVersions(runtime);
    const manager = runtime.backendForRole("manager");
    const second = (await manager.evidence.listVersions({ findingId: finding.id })).at(-1);
    if (!second) throw new Error("Expected Evidence version 2.");
    await manager.evidence.review({
      operationId: `OP-EVIDENCE-${second.id}-R${second.revision}-NOT_CLOSE`,
      evidenceVersionId: second.id,
      expectedEvidenceVersionRevision: second.revision,
      findingId: finding.id,
      expectedFindingRevision: finding.revision,
      decision: "NOT_CLOSE",
      commentToAuditee: "Version 2 remains incomplete.",
      internalCaaNote: "Record the exact version 2 decision.",
    });

    finding = await manager.findings.get({ findingId: finding.id });
    finding = await submitEvidenceVersion(runtime, finding, "Fly_Namibia_PBE_Position_Confirmation_V3.pdf");
    const third = (await manager.evidence.listVersions({ findingId: finding.id })).at(-1);
    if (!third) throw new Error("Expected Evidence version 3.");
    const thirdReview = await manager.evidence.review({
      operationId: `OP-EVIDENCE-${third.id}-R${third.revision}-NOT_CLOSE`,
      evidenceVersionId: third.id,
      expectedEvidenceVersionRevision: third.revision,
      findingId: finding.id,
      expectedFindingRevision: finding.revision,
      decision: "NOT_CLOSE",
      commentToAuditee: "Version 3 remains incomplete.",
      internalCaaNote: "Record the later exact version 3 decision independently.",
    });

    expect(thirdReview).toMatchObject({ evidenceVersionId: "EV-CAB-2026-001-V3", evidenceVersionRevision: 3 });
    const events = await manager.auditTrail.list({ entityType: "finding", entityId: finding.id });
    expect(events.items.filter((event) => event.action === "evidence.reviewed")).toHaveLength(3);
  });
});
