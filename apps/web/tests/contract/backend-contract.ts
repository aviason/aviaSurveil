import { describe, expect, it } from "vitest";

import type {
  Backend,
  BackendPrincipal,
  DocumentMetadataView,
  EvidenceVersionView,
  FindingView,
  VisibleActionEffect,
} from "../../src/backend/backend";
import { REACT_ROUTE_CONTRACTS } from "../../src/app/route-contracts";
import { SCREEN_VISIBLE_ACTIONS } from "../../src/mock/seed-data";

export const FIXED_NOW = "2026-06-15T09:00:00.000Z";
const HTTP_CANONICAL_INSPECTOR_SUBJECT_ID = "154ec5ac-6f97-4f55-916f-d2f142fc6211";

export interface BackendContractHarness {
  backendFor(principal: BackendPrincipal): Backend;
}

export type BackendContractHarnessFactory = () => Promise<BackendContractHarness>;

function normalizeReleasedDocument(
  document: DocumentMetadataView,
): DocumentMetadataView {
  return {
    id: document.id,
    organizationId: document.organizationId,
    title: document.title,
    kind: document.kind,
    version: document.version,
    revision: document.revision,
    createdAt: document.createdAt,
    publicReviewResult: document.publicReviewResult,
    downloadFileName: document.downloadFileName,
  };
}

export const PRINCIPALS = {
  inspector: {
    subjectId: "USR-INSPECTOR-AMINA",
    role: "inspector",
    organizationId: null,
  },
  otherInspector: {
    subjectId: "USR-INSPECTOR-DAVID",
    role: "inspector",
    organizationId: null,
  },
  leadInspector: {
    subjectId: "USR-LEAD-CANER",
    role: "leadInspector",
    organizationId: null,
  },
  manager: {
    subjectId: "USR-MANAGER-NORA",
    role: "manager",
    organizationId: null,
  },
  finance: {
    subjectId: "USR-FINANCE-LINA",
    role: "finance",
    organizationId: null,
  },
  gm: {
    subjectId: "USR-GM-OMAR",
    role: "gm",
    organizationId: null,
  },
  executiveDirector: {
    subjectId: "USR-ED-ZARA",
    role: "executiveDirector",
    organizationId: null,
  },
  auditee: {
    subjectId: "USR-AUDITEE-FLY",
    role: "auditee",
    organizationId: "ORG-FLY-NAMIBIA",
  },
  admin: {
    subjectId: "USR-ADMIN-ADA",
    role: "admin",
    organizationId: null,
  },
} as const satisfies Record<string, BackendPrincipal>;

export async function createCanonicalFinding(
  harness: BackendContractHarness,
): Promise<FindingView> {
  const inspector = harness.backendFor(PRINCIPALS.inspector);
  const response = await inspector.inspections.upsertChecklistResponse({
    operationId: "OP-RESPONSE-CANONICAL",
    responseId: "RESP-CAB-EMEQ-PBE-001",
    auditId: "AUD-2026-001",
    questionId: "CAB-EMEQ-PBE-001",
    expectedResponseRevision: null,
    answer: "NON_COMPLIANT",
    comment: "PBE serviceability and accessibility could not be confirmed for this Audit.",
  });
  const potential = await inspector.potentialFindings.create({
    operationId: "OP-PF-CANONICAL",
    auditId: "AUD-2026-001",
    questionId: "CAB-EMEQ-PBE-001",
    checklistResponseId: response.id,
    expectedChecklistResponseRevision: response.revision,
    title: "PBE serviceability and accessibility not confirmed",
    description: "The configured cabin check could not confirm that the PBE was serviceable and accessible.",
    requiredComment: response.comment,
    inspectionAttachmentIds: [],
  });

  const lead = harness.backendFor(PRINCIPALS.leadInspector);
  const converted = await lead.potentialFindings.decide({
    operationId: "OP-PF-CONVERT-CANONICAL",
    potentialFindingId: potential.id,
    expectedPotentialFindingRevision: potential.revision,
    decision: "CONVERT",
    severity: "LEVEL_1_CRITICAL",
    capRequired: true,
    evidenceRequired: true,
    dueDate: "2026-07-15",
  });
  expect(converted.finding).not.toBeNull();
  return converted.finding!;
}

export async function submitCanonicalCap(
  harness: BackendContractHarness,
  finding: FindingView,
) {
  const auditee = harness.backendFor(PRINCIPALS.auditee);
  return auditee.caps.submit({
    operationId: "OP-CAP-SUBMIT-CANONICAL",
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    rootCause:
      "Pre-flight cabin equipment serviceability checks did not reconcile the PBE position with the deferred defect list.",
    correctiveAction:
      "Replace or service the affected PBE, update the cabin defect record, and confirm serviceability before release.",
    preventiveAction:
      "Add a supervisor review of emergency equipment checks and monthly sampling of PBE serviceability records.",
    responsiblePerson: "Fly Namibia Cabin Safety Manager",
    targetCompletionDate: "2026-07-15",
    commentToCaa: "CAP submitted for CAA review.",
  });
}

export async function submitAndAcceptCanonicalCap(
  harness: BackendContractHarness,
  finding: FindingView,
) {
  const submitted = await submitCanonicalCap(harness, finding);

  const lead = harness.backendFor(PRINCIPALS.leadInspector);
  const accepted = await lead.caps.review({
    operationId: "OP-CAP-ACCEPT-CANONICAL",
    capRevisionId: submitted.capRevisionId,
    expectedCapRevision: submitted.capRevision,
    findingId: finding.id,
    expectedFindingRevision: submitted.findingRevision,
    decision: "ACCEPT",
    commentToAuditee: "CAP accepted. Submit the required PBE serviceability record.",
    internalCaaNote: "CAP actions are credible; Evidence verification remains required.",
  });
  return { submitted, accepted };
}

export async function submitEvidence(
  harness: BackendContractHarness,
  operationSuffix: string,
  fileName: string,
): Promise<EvidenceVersionView> {
  const auditee = harness.backendFor(PRINCIPALS.auditee);
  const finding = await auditee.findings.get({ findingId: "FND-CAB-2026-001" });
  const body = new TextEncoder().encode(
    `%PDF-1.7\n1 0 obj\n<</Type/Catalog/Label(${operationSuffix})>>\nendobj\n%%EOF\n`,
  );
  const digest = await crypto.subtle.digest("SHA-256", body);
  const sha256 = `sha256:${Array.from(new Uint8Array(digest), (byte) =>
    byte.toString(16).padStart(2, "0"),
  ).join("")}`;
  const upload = await auditee.evidence.beginUpload({
    operationId: `OP-EVIDENCE-BEGIN-${operationSuffix}`,
    findingId: finding.id,
    expectedFindingRevision: finding.revision,
    fileName,
    declaredMediaType: "application/pdf",
    byteSize: body.byteLength,
    sha256,
  });
  if (auditee.mode === "http") {
    const response = await fetch(upload.uploadUrl, {
      method: "PUT",
      headers: upload.requiredHeaders,
      body,
    });
    expect(response.ok).toBe(true);
  }
  const completed = await auditee.evidence.completeUpload({
    operationId: `OP-EVIDENCE-COMPLETE-${operationSuffix}`,
    uploadId: upload.uploadId,
    sha256,
    byteSize: body.byteLength,
  });
  let versions: EvidenceVersionView[] = [];
  for (let attempt = 0; attempt < 100; attempt += 1) {
    versions = await auditee.evidence.listVersions({ findingId: finding.id });
    if (auditee.mode === "mock" || versions.find(({ id }) => id === completed.evidenceVersionId)?.scanState === "CLEAN") {
      break;
    }
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  const version = versions.find((candidate) => candidate.id === completed.evidenceVersionId);
  expect(version).toBeDefined();
  return version!;
}

export function backendContract(createHarness: BackendContractHarnessFactory): void {
  describe("shared Backend contract", () => {
    it("uses stable exact Audit/question scope and keeps another Inspector's question read-only", async () => {
      const harness = await createHarness();
      const inspector = harness.backendFor(PRINCIPALS.inspector);
      const packageView = await inspector.inspections.getPackage({
        packageId: "PKG-CAB-2026-001",
      });
      expect(packageView.auditId).toBe("AUD-2026-001");
      expect(packageView.questions.map((question) => question.id)).toContain("CAB-EMEQ-PBE-001");
      expect(
        packageView.questions.find((question) => question.id === "CAB-GALLEY-001")
          ?.assignedInspectorUserIds,
      ).toEqual(["USR-INSPECTOR-DAVID"]);

      await expect(
        inspector.inspections.upsertChecklistResponse({
          operationId: "OP-OTHER-INSPECTOR-QUESTION",
          responseId: "RESP-CAB-GALLEY-001",
          auditId: "AUD-2026-001",
          questionId: "CAB-GALLEY-001",
          expectedResponseRevision: null,
          answer: "COMPLIANT",
          comment: "",
        }),
      ).rejects.toThrow(/(?:assigned Inspector|Inspector.*assigned)/i);
    });

    it("requires a separate Lead conversion before a canonical Finding exists", async () => {
      const harness = await createHarness();
      const inspector = harness.backendFor(PRINCIPALS.inspector);
      const response = await inspector.inspections.upsertChecklistResponse({
        operationId: "OP-RESPONSE-PF-ONLY",
        responseId: "RESP-CAB-EMEQ-PBE-001",
        auditId: "AUD-2026-001",
        questionId: "CAB-EMEQ-PBE-001",
        expectedResponseRevision: null,
        answer: "NON_COMPLIANT",
        comment: "Required exact-Audit comment.",
      });
      const potential = await inspector.potentialFindings.create({
        operationId: "OP-PF-ONLY",
        auditId: "AUD-2026-001",
        questionId: "CAB-EMEQ-PBE-001",
        checklistResponseId: response.id,
        expectedChecklistResponseRevision: response.revision,
        title: "PBE serviceability and accessibility not confirmed",
        description: "Configured check exception.",
        requiredComment: response.comment,
        inspectionAttachmentIds: [],
      });
      expect(potential).toMatchObject({
        id: "PF-2026-001",
        status: "PENDING_LEAD_REVIEW",
        convertedFindingId: null,
      });
      const beforeConversion = await harness
        .backendFor(PRINCIPALS.leadInspector)
        .findings.list({});
      expect(beforeConversion.items.some((finding) => finding.findingNumber === "CAB-2026-001")).toBe(
        false,
      );
    });

    it("exposes Potential Finding reads without giving Inspectors Lead queue authority", async () => {
      const harness = await createHarness();
      const inspector = harness.backendFor(PRINCIPALS.inspector);
      const response = await inspector.inspections.upsertChecklistResponse({
        operationId: "OP-RESPONSE-PF-READS",
        responseId: "RESP-CAB-EMEQ-PBE-001",
        auditId: "AUD-2026-001",
        questionId: "CAB-EMEQ-PBE-001",
        expectedResponseRevision: null,
        answer: "NON_COMPLIANT",
        comment: "Required exact-Audit comment for Potential Finding reads.",
      });
      const potential = await inspector.potentialFindings.create({
        operationId: "OP-PF-READS",
        auditId: "AUD-2026-001",
        questionId: "CAB-EMEQ-PBE-001",
        checklistResponseId: response.id,
        expectedChecklistResponseRevision: response.revision,
        title: "PBE serviceability and accessibility not confirmed",
        description: "Configured check exception.",
        requiredComment: response.comment,
        inspectionAttachmentIds: [],
      });

      const lead = harness.backendFor(PRINCIPALS.leadInspector);
      await expect(inspector.potentialFindings.list({ status: "PENDING_LEAD_REVIEW", limit: 50 }))
        .rejects.toThrow(/Lead Inspector/i);
      expect(await inspector.potentialFindings.get({ potentialFindingId: potential.id })).toEqual(potential);
      const queue = await lead.potentialFindings.list({ status: "PENDING_LEAD_REVIEW", limit: 50 });
      expect(queue).toEqual({ items: [potential], nextCursor: null });
      expect(await lead.potentialFindings.get({ potentialFindingId: potential.id })).toEqual(potential);
      await expect(harness.backendFor(PRINCIPALS.finance).potentialFindings.get({
        potentialFindingId: potential.id,
      })).rejects.toThrow(/Forbidden|Lead Inspector|authority/i);
    });

    it("starts an explicitly Evidence-only Observation at Evidence submission", async () => {
      const harness = await createHarness();
      const inspector = harness.backendFor(PRINCIPALS.inspector);
      const response = await inspector.inspections.upsertChecklistResponse({
        operationId: "OP-RESPONSE-EVIDENCE-ONLY-OBSERVATION",
        responseId: "RESP-CAB-EMEQ-PBE-001",
        auditId: "AUD-2026-001",
        questionId: "CAB-EMEQ-PBE-001",
        expectedResponseRevision: null,
        answer: "OBSERVATION",
        comment: "Observation requires Evidence but no CAP.",
      });
      const potential = await inspector.potentialFindings.create({
        operationId: "OP-PF-EVIDENCE-ONLY-OBSERVATION",
        auditId: "AUD-2026-001",
        questionId: "CAB-EMEQ-PBE-001",
        checklistResponseId: response.id,
        expectedChecklistResponseRevision: response.revision,
        title: "Evidence-only PBE Observation",
        description: "The Lead may configure Evidence without requiring a CAP.",
        requiredComment: response.comment,
        inspectionAttachmentIds: [],
      });
      const converted = await harness.backendFor(PRINCIPALS.leadInspector).potentialFindings.decide({
        operationId: "OP-PF-CONVERT-EVIDENCE-ONLY-OBSERVATION",
        potentialFindingId: potential.id,
        expectedPotentialFindingRevision: potential.revision,
        decision: "CONVERT",
        severity: "OBSERVATION",
        capRequired: false,
        evidenceRequired: true,
        dueDate: null,
      });
      expect(converted.finding).toMatchObject({
        severity: "OBSERVATION",
        status: "EVIDENCE_REQUIRED",
        capRequired: false,
        evidenceRequired: true,
        dueDate: null,
        currentOwnerType: "AUDITEE",
      });
    });

    it("separates Auditee CAP submission from CAA acceptance and keeps the Finding open", async () => {
      const harness = await createHarness();
      const finding = await createCanonicalFinding(harness);
      const { submitted, accepted } = await submitAndAcceptCanonicalCap(harness, finding);
      expect(submitted.capStatus).toBe("SUBMITTED");
      expect(submitted.findingStatus).toBe("CAP_SUBMITTED");
      expect(accepted.capStatus).toBe("ACCEPTED");
      expect(accepted.findingStatus).toBe("EVIDENCE_REQUIRED");
      expect(accepted.findingStatus).not.toBe("CLOSED");
    });

    it("returns immutable role-shaped CAP revisions and omits Internal CAA Notes for Auditee", async () => {
      const harness = await createHarness();
      const finding = await createCanonicalFinding(harness);
      const auditee = harness.backendFor(PRINCIPALS.auditee);
      const first = await auditee.caps.submit({
        operationId: "OP-CAP-SUBMIT-READS-R1",
        findingId: finding.id,
        expectedFindingRevision: finding.revision,
        rootCause: "Initial root cause",
        correctiveAction: "Initial corrective action",
        preventiveAction: "Initial preventive action",
        responsiblePerson: "Fly Namibia Cabin Safety Manager",
        targetCompletionDate: "2026-07-15",
        commentToCaa: "Initial CAP submitted for CAA review.",
      });
      const lead = harness.backendFor(PRINCIPALS.leadInspector);
      const moreInfo = await lead.caps.review({
        operationId: "OP-CAP-MORE-INFO-READS-R1",
        capRevisionId: first.capRevisionId,
        expectedCapRevision: first.capRevision,
        findingId: finding.id,
        expectedFindingRevision: first.findingRevision,
        decision: "REQUEST_MORE_INFORMATION",
        commentToAuditee: "Clarify the implementation sequence and resubmit.",
        internalCaaNote: "Revision 1 needs stronger preventive-action sequencing.",
      });
      const second = await auditee.caps.submit({
        operationId: "OP-CAP-SUBMIT-READS-R2",
        findingId: finding.id,
        expectedFindingRevision: moreInfo.findingRevision,
        rootCause: "Revised root cause",
        correctiveAction: "Revised corrective action",
        preventiveAction: "Revised preventive action",
        responsiblePerson: "Fly Namibia Cabin Safety Manager",
        targetCompletionDate: "2026-07-20",
        commentToCaa: "Revised CAP submitted for CAA review.",
      });

      const caaList = await lead.caps.listRevisions({ findingId: finding.id });
      expect(caaList.items.map(({ id, revision, audience }) => [id, revision, audience])).toEqual([
        [first.capRevisionId, 1, "CAA"],
        [second.capRevisionId, 2, "CAA"],
      ]);
      expect(caaList.nextCursor).toBeNull();
      expect(caaList.items[0]?.latestReview).toMatchObject({
        decision: "REQUEST_MORE_INFORMATION",
        internalCaaNote: "Revision 1 needs stronger preventive-action sequencing.",
      });
      expect(caaList.items[0]?.status).toBe("SUPERSEDED");
      expect(await lead.caps.getRevision({ capRevisionId: first.capRevisionId })).toEqual(caaList.items[0]);

      const auditeeList = await auditee.caps.listRevisions({ findingId: finding.id });
      expect(auditeeList.items.map(({ audience }) => audience)).toEqual(["AUDITEE", "AUDITEE"]);
      expect(JSON.stringify(auditeeList)).not.toMatch(/internalCaaNote|Internal CAA Note/i);
      const auditeeDetail = await auditee.caps.getRevision({ capRevisionId: first.capRevisionId });
      expect(auditeeDetail.audience).toBe("AUDITEE");
      expect(auditeeDetail.status).toBe("SUPERSEDED");
      expect(JSON.stringify(auditeeDetail)).not.toMatch(/internalCaaNote|Internal CAA Note/i);
      await expect(harness.backendFor(PRINCIPALS.gm).caps.getRevision({
        capRevisionId: first.capRevisionId,
      })).rejects.toThrow(/Forbidden|authority/i);
    });

    it("preserves immutable Evidence versions and closes only the exact latest accepted version", async () => {
      const harness = await createHarness();
      const finding = await createCanonicalFinding(harness);
      await submitAndAcceptCanonicalCap(harness, finding);
      const versionOne = await submitEvidence(
        harness,
        "V1",
        "Fly_Namibia_PBE_Serviceability_Record_CAB-2026-001.pdf",
      );
      const lead = harness.backendFor(PRINCIPALS.leadInspector);
      let current = await lead.findings.get({ findingId: finding.id });
      expect(current.status).toBe("PENDING_CAA_REVIEW");
      const partial = await lead.evidence.review({
        operationId: "OP-EVIDENCE-PARTIAL-V1",
        evidenceVersionId: versionOne.id,
        expectedEvidenceVersionRevision: versionOne.revision,
        findingId: finding.id,
        expectedFindingRevision: current.revision,
        decision: "PARTIALLY_CLOSE",
        commentToAuditee: "The serviceability record is accepted; provide the cabin position confirmation.",
        internalCaaNote: "Version 1 covers serviceability but not accessibility.",
      });
      expect(partial.findingStatus).toBe("EVIDENCE_MORE_INFORMATION_REQUESTED");

      const versionTwo = await submitEvidence(
        harness,
        "V2",
        "Fly_Namibia_PBE_Position_Confirmation_CAB-2026-001.pdf",
      );
      const versionsBeforeClosure = await lead.evidence.listVersions({ findingId: finding.id });
      expect(versionsBeforeClosure.map(({ id, version }) => [id, version])).toEqual([
        [versionOne.id, 1],
        [versionTwo.id, 2],
      ]);

      current = await lead.findings.get({ findingId: finding.id });
      const closed = await lead.evidence.review({
        operationId: "OP-EVIDENCE-CLOSE-V2",
        evidenceVersionId: versionTwo.id,
        expectedEvidenceVersionRevision: versionTwo.revision,
        findingId: finding.id,
        expectedFindingRevision: current.revision,
        decision: "CLOSE",
        commentToAuditee: "Evidence accepted and verified.",
        internalCaaNote: "Version 2 completes serviceability and accessibility verification.",
      });
      expect(closed.findingStatus).toBe("CLOSED");
      const closedFinding = await lead.findings.get({ findingId: finding.id });
      expect(closedFinding.closureBasis).toBe("EVIDENCE_VERIFIED");
      expect((await lead.evidence.listVersions({ findingId: finding.id }))).toHaveLength(2);
    });

    it("scopes Auditee projections to its organization and omits internal CAA data", async () => {
      const harness = await createHarness();
      const finding = await createCanonicalFinding(harness);
      await submitAndAcceptCanonicalCap(harness, finding);
      const auditee = harness.backendFor(PRINCIPALS.auditee);
      const projection = await auditee.findings.list({});
      expect(projection.items.length).toBeGreaterThan(0);
      expect(projection.items.every((finding) => finding.organizationId === "ORG-FLY-NAMIBIA")).toBe(
        true,
      );
      const raw = JSON.stringify(projection);
      expect(raw).not.toMatch(
        /Internal CAA Note|internalCaaNote|SkyCargo|internalRisk|inspectorWorkload|enforcementDeliberation/i,
      );
      await expect(
        auditee.reports.getVersion({ reportVersionId: "RPT-CAB-2026-001-V1" }),
      ).rejects.toThrow(/unavailable to this Auditee/i);
    });

    it("keeps authorized closure reason-required and separate from Evidence verification", async () => {
      const harness = await createHarness();
      const finding = await createCanonicalFinding(harness);
      await expect(
        harness.backendFor(PRINCIPALS.inspector).findings.authorizedClose({
          operationId: "OP-AUTH-CLOSE-WRONG-ROLE",
          findingId: finding.id,
          expectedFindingRevision: finding.revision,
          reason: "Inspector may not use the authorized path.",
        }),
      ).rejects.toThrow(/Department Manager/i);

      const manager = harness.backendFor(PRINCIPALS.manager);
      await expect(
        manager.findings.authorizedClose({
          operationId: "OP-AUTH-CLOSE-NO-REASON",
          findingId: finding.id,
          expectedFindingRevision: finding.revision,
          reason: "",
        }),
      ).rejects.toThrow(/reason/i);
      const closed = await manager.findings.authorizedClose({
        operationId: "OP-AUTH-CLOSE-VALID",
        findingId: finding.id,
        expectedFindingRevision: finding.revision,
        reason: "Authorized mock closure path exercised separately for contract verification.",
      });
      expect(closed).toMatchObject({ status: "CLOSED", closureBasis: "AUTHORIZED" });
    });

    it("binds report decisions to versions and never closes an open Finding", async () => {
      const harness = await createHarness();
      const finding = await createCanonicalFinding(harness);
      const manager = harness.backendFor(PRINCIPALS.manager);
      const gm = harness.backendFor(PRINCIPALS.gm);
      const executive = harness.backendFor(PRINCIPALS.executiveDirector);
      const before = await manager.reports.getVersion({ reportVersionId: "RPT-CAB-2026-001-V1" });
      const gmReview = await manager.reports.decide({
        operationId: "OP-REPORT-MANAGER-FORWARD",
        reportVersionId: before.reportVersionId,
        expectedReportVersionRevision: before.revision,
        decision: "FORWARD",
        reason: "Forward the exact candidate Final Report version to the General Manager.",
      });
      const executiveReview = await gm.reports.decide({
        operationId: "OP-REPORT-GM-FORWARD",
        reportVersionId: gmReview.reportVersionId,
        expectedReportVersionRevision: gmReview.revision,
        decision: "FORWARD",
        reason: "Forward the exact candidate Final Report version to the Executive Director.",
      });
      const issued = await executive.reports.decide({
        operationId: "OP-REPORT-ISSUE",
        reportVersionId: executiveReview.reportVersionId,
        expectedReportVersionRevision: executiveReview.revision,
        decision: "ISSUE_AND_LOCK",
        reason: "Issue the exact candidate report version.",
      });
      expect(issued.status).toBe("LOCKED");
      expect(issued.issuedAt).not.toBeNull();
      expect((await executive.findings.get({ findingId: finding.id })).status).not.toBe("CLOSED");

      const auditee = harness.backendFor(PRINCIPALS.auditee);
      expect(auditee.auditeeReports).toBeDefined();
      expect(auditee.documents).toBeDefined();
      const releasedReports = await auditee.auditeeReports!.listReleased({ kind: "FINAL" });
      expect(releasedReports).toEqual({
        items: [{
          reportVersionId: "RPT-CAB-2026-001-V1",
          reportId: "RPT-CAB-2026-001",
          kind: "FINAL",
          organizationId: "ORG-FLY-NAMIBIA",
          auditId: "AUD-2026-001",
          findingIds: [],
          version: 1,
          status: "LOCKED",
          revision: 4,
          issuedAt: issued.issuedAt,
          responseDueDate: null,
          caaVisibleCommentState: "NO_COMMENT_RECORDED",
          caaVisibleComment: null,
        }],
        nextCursor: null,
      });
      expect(await auditee.auditeeReports!.getReleased({
        reportVersionId: issued.reportVersionId,
      })).toEqual(releasedReports.items[0]);

      let releasedDocuments = await auditee.documents!.list({});
      for (let attempt = 0; attempt < 100; attempt += 1) {
        const document = releasedDocuments.items.find(
          ({ id }) => id === issued.reportVersionId,
        );
        if (
          document?.downloadFileName &&
          (auditee.mode === "mock" || document.renderStatus === "SUCCEEDED")
        ) {
          break;
        }
        await new Promise((resolve) => setTimeout(resolve, 50));
        releasedDocuments = await auditee.documents!.list({});
      }
      const normalizedReleasedDocuments = {
        items: releasedDocuments.items.map(normalizeReleasedDocument),
        nextCursor: releasedDocuments.nextCursor,
      };
      expect(normalizedReleasedDocuments).toEqual({
        items: [{
          id: "RPT-CAB-2026-001-V1",
          organizationId: "ORG-FLY-NAMIBIA",
          title: "Report RPT-CAB-2026-001",
          kind: "REPORT",
          version: 1,
          revision: 4,
          createdAt: issued.issuedAt,
          publicReviewResult: "RELEASED",
          downloadFileName: "RPT-CAB-2026-001.pdf",
        }],
        nextCursor: null,
      });
      expect(normalizeReleasedDocument(await auditee.documents!.open({
        documentId: issued.reportVersionId,
      }))).toEqual(normalizedReleasedDocuments.items[0]);
    });

    it("produces the same normalized communication, notification, and calendar transcript", async () => {
      const harness = await createHarness();
      const inspector = harness.backendFor(PRINCIPALS.inspector);
      const manager = harness.backendFor(PRINCIPALS.manager);
      const auditee = harness.backendFor(PRINCIPALS.auditee);
      if (!inspector.communications || !inspector.calendar || !auditee.communications ||
          !manager.calendar || !auditee.notifications || !auditee.calendar) {
        throw new Error("Task 8 communication, notification, or calendar capability is unavailable.");
      }

      const input = {
        idempotencyKey: "IDEM-TASK8-CONTRACT-MESSAGE",
        expectedRevision: null,
        organizationId: "ORG-FLY-NAMIBIA",
        subject: "Cabin Inspection follow-up",
        body: "Please provide the requested public training record.",
        audience: "AUDITEE" as const,
      };
      const sent = await inspector.communications.send(input);
      const replayed = await inspector.communications.send(input);
      const internal = await inspector.communications.send({
        idempotencyKey: "IDEM-TASK8-CONTRACT-INTERNAL",
        expectedRevision: null,
        organizationId: "ORG-FLY-NAMIBIA",
        subject: "Internal CAA Note",
        body: "Private enforcement deliberation.",
        audience: "CAA",
      });
      const auditeeMessages = await auditee.communications.list({
        organizationId: "ORG-FLY-NAMIBIA",
      });
      const visibleMessage = auditeeMessages.items.find(
        (message) => message.subject === input.subject,
      );
      const notifications = await auditee.notifications.list({});
      const notification = notifications.items.find(
        (item) => item.title === "New CAA communication" &&
          item.body.includes(input.subject),
      );
      expect(notification).toBeDefined();
      const read = await auditee.notifications.markRead({
        idempotencyKey: "IDEM-TASK8-CONTRACT-READ",
        expectedRevision: notification!.revision,
        notificationId: notification!.id,
      });
      const replayedRead = await auditee.notifications.markRead({
        idempotencyKey: "IDEM-TASK8-CONTRACT-READ",
        expectedRevision: notification!.revision,
        notificationId: notification!.id,
      });
      const inspectorCalendar = await inspector.calendar.list({});
      const inspectorItem = inspectorCalendar.items.find(
        (item) => item.auditId === "AUD-2026-001",
      );
      expect(inspectorItem).toBeDefined();
      const opened = await inspector.calendar.openItem({
        calendarItemId: inspectorItem!.id,
      });
      const managerCalendar = await manager.calendar.list({});
      const auditeeCalendar = await auditee.calendar.list({});

      expect([
        {
          event: "communication.sent",
          organizationId: sent.organizationId,
          subject: sent.subject,
          audience: sent.audience,
          direction: sent.direction,
          revision: sent.revision,
          replayExact: JSON.stringify(replayed) === JSON.stringify(sent),
        },
        {
          event: "communication.internal-separated",
          direction: internal.direction,
          visibleMessageCount: auditeeMessages.items.filter(
            (message) => message.subject === input.subject,
          ).length,
          visibleMessageDirection: visibleMessage?.direction,
          leakedInternal: JSON.stringify(auditeeMessages).includes(
            "Private enforcement deliberation.",
          ),
        },
        {
          event: "notification.read",
          subjectId: read.subjectId,
          title: read.title,
          unreadBefore: notification!.readAt === null,
          revisionBefore: notification!.revision,
          revisionAfter: read.revision,
          readRecorded: read.readAt !== null,
          replayExact: JSON.stringify(replayedRead) === JSON.stringify(read),
        },
        {
          event: "calendar.authorized-work",
          inspectorItemCount: inspectorCalendar.items.length,
          auditId: opened.auditId,
          organizationId: opened.organizationId,
          scheduledDate: opened.scheduledDate,
          dueState: opened.dueState,
          nextAction: opened.nextAction,
          managerAuditIds: managerCalendar.items.map(({ auditId }) => auditId),
          auditeeAuditIds: auditeeCalendar.items.map(({ auditId }) => auditId),
        },
      ]).toEqual([
        {
          event: "communication.sent",
          organizationId: "ORG-FLY-NAMIBIA",
          subject: "Cabin Inspection follow-up",
          audience: "AUDITEE",
          direction: "CAA_TO_AUDITEE",
          revision: 1,
          replayExact: true,
        },
        {
          event: "communication.internal-separated",
          direction: "CAA_INTERNAL",
          visibleMessageCount: 1,
          visibleMessageDirection: "CAA_TO_AUDITEE",
          leakedInternal: false,
        },
        {
          event: "notification.read",
          subjectId: "USR-AUDITEE-FLY",
          title: "New CAA communication",
          unreadBefore: true,
          revisionBefore: 1,
          revisionAfter: 2,
          readRecorded: true,
          replayExact: true,
        },
        {
          event: "calendar.authorized-work",
          inspectorItemCount: 1,
          auditId: "AUD-2026-001",
          organizationId: "ORG-FLY-NAMIBIA",
          scheduledDate: "2026-06-18",
          dueState: "DUE_SOON",
          nextAction: "Continue Cabin Inspection checklist",
          managerAuditIds: ["AUD-2026-001", "AUD-2026-099"],
          auditeeAuditIds: ["AUD-2026-001"],
        },
      ]);
    });

    it("projects all 86 screen/action contracts and the same governed Task 9 transcript", async () => {
      const harness = await createHarness();
      const uniqueScreens = new Set<string>();
      const uniqueActions = new Map<string, {
        backend: Backend;
        screenId: string;
        actionId: string;
        effect: VisibleActionEffect;
      }>();
      for (const principal of Object.values(PRINCIPALS)) {
        const backend = harness.backendFor(principal);
        if (!backend.administration) {
          throw new Error("Task 9 Administration capability is unavailable.");
        }
        const expectedRoutes = REACT_ROUTE_CONTRACTS.filter(
          (route) => route.requiredRole === null || route.requiredRole === principal.role,
        );
        const listed = await backend.administration.listScreenProjections({});
        expect(listed.map(({ screenId }) => screenId)).toEqual(
          expectedRoutes.map(({ id }) => id),
        );
        for (const route of expectedRoutes) {
          const projection = await backend.administration.getScreenProjection({
            screenId: route.id,
          });
          expect(projection.screenId).toBe(route.id);
          expect(projection.visibleActions).toEqual(SCREEN_VISIBLE_ACTIONS[route.id]);
          uniqueScreens.add(route.id);
          for (const action of projection.visibleActions) {
            const key = `${route.id}/${action.id}`;
            uniqueActions.set(key, {
              backend,
              screenId: route.id,
              actionId: action.id,
              effect: action.effect,
            });
          }
        }
      }
      expect([...uniqueScreens]).toHaveLength(86);
      expect([...uniqueActions]).toHaveLength(108);
      for (const { backend, screenId, actionId, effect } of uniqueActions.values()) {
        const invoked = await backend.administration!.invokeVisibleAction({
          screenId,
          actionId,
        });
        expect(invoked).toEqual({ screenId, actionId, effect });
      }
      await expect(
        harness.backendFor(PRINCIPALS.inspector).administration!
          .getScreenProjection({ screenId: "admin-reports" }),
      ).rejects.toThrow(/Forbidden|unavailable|role|authority/i);

      const manager = harness.backendFor(PRINCIPALS.manager);
      const otherInspector = harness.backendFor(PRINCIPALS.otherInspector);
      const admin = harness.backendFor(PRINCIPALS.admin);
      if (!manager.risk || !otherInspector.assistantDrafts || !admin.adminWorkspace) {
        throw new Error("Task 9 governed capability slice is unavailable.");
      }
      const overview = await manager.risk.getOverview({
        organizationId: "ORG-SKYCARGO",
      });
      const management = await manager.risk.getManagementProjection({});
      const guidance = await otherInspector.assistantDrafts.getGuidance({});
      const draft = await otherInspector.assistantDrafts.createDraft({
        findingId: "FND-SKYCARGO-2026-099",
        prompt: "Draft an evidence request only.",
      });
      const reportDefinitions = await admin.adminWorkspace.listReportDefinitions({
        search: "package",
      });
      const directory = await admin.adminWorkspace.listAccessDirectory({
        search: "David",
        role: "inspector",
      });
      const organizations = await admin.adminWorkspace.listOrganizations({
        search: "Fly",
        organizationType: "OPERATOR",
        status: "ACTIVE",
        scope: "CAA oversight",
      });
      const expectedDirectory =
        admin.mode === "http"
          ? [{
              subjectId: "USR-INSPECTOR-DAVID",
              displayName: "David Inspector",
              roles: ["inspector"],
              organizationId: "CAA",
              email: "david.inspector@example.test",
              mfaEnrolled: false,
              mfaState: "unenrolled",
              requiredActions: [],
              invitationState: "none",
              accountStatus: "enabled",
              applicationProfileState: "linked",
              membershipId: null,
              membershipState: "absent",
              membershipRevision: 0,
              membershipDrift: "untracked",
              lastSuccessfulSessionAt: "2026-06-15T09:00:00Z",
              providerObservedAt: "2026-06-15T09:00:00Z",
            }]
          : [{
              subjectId: "USR-INSPECTOR-DAVID",
              displayName: "David Inspector",
              roles: ["inspector"],
              organizationId: null,
              email: "Not configured in demo",
              mfaEnrolled: false,
              mfaState: "Not configured in demo",
              requiredActions: [],
              invitationState: "Not configured in demo",
              accountStatus: "Not configured in demo",
              applicationProfileState: "linked",
              membershipId: null,
              membershipState: "Not configured in demo",
              membershipRevision: 0,
              membershipDrift: "Not configured in demo",
              lastSuccessfulSessionAt: null,
              providerObservedAt: "",
            }];

      expect({
        overview: {
          organizationId: overview.organizationId,
          overdueFindingCount: overview.overdueFindingCount,
          openFindingCount: overview.openFindingCount,
          repeatFindingCount: overview.repeatFindingCount,
          revision: overview.revision,
        },
        management: {
          findings: management.findings.map((finding) => ({
            findingId: finding.findingId,
            findingNumber: finding.findingNumber,
            organizationId: finding.organizationId,
            severity: finding.severity,
            riskLevel: finding.riskLevel,
            status: finding.status,
            dueState: finding.dueState,
            capRequired: finding.capRequired,
          })),
          capEffectiveness: management.capEffectiveness.map((item) => ({
            findingId: item.findingId,
            capRevision: item.capRevision,
            capStatus: item.capStatus,
            state: item.state,
          })),
          revision: management.revision,
        },
        guidance,
        draft: {
          findingId: draft.findingId,
          prompt: draft.prompt,
          draft: draft.draft,
          advisoryOnly: draft.advisoryOnly,
          canCreateFinding: draft.canCreateFinding,
          canSetSeverity: draft.canSetSeverity,
          canCloseFinding: draft.canCloseFinding,
        },
        reportDefinitions: reportDefinitions.items,
        directory: directory.items,
        organizations: organizations.items,
      }).toEqual({
        overview: {
          organizationId: "ORG-SKYCARGO",
          overdueFindingCount: 1,
          openFindingCount: 1,
          repeatFindingCount: 0,
          revision: 1,
        },
        management: {
          findings: [{
            findingId: "FND-SKYCARGO-2026-099",
            findingNumber: "CAR-2026-099",
            organizationId: "ORG-SKYCARGO",
            severity: "LEVEL_2_MAJOR",
            riskLevel: "MEDIUM",
            status: "OPEN",
            dueState: "OVERDUE",
            capRequired: true,
          }],
          capEffectiveness: [{
            findingId: "FND-SKYCARGO-2026-099",
            capRevision: 2,
            capStatus: "MORE_INFORMATION_REQUESTED",
            state: "NOT_ELIGIBLE",
          }],
          revision: 1,
        },
        guidance: {
          advisoryOnly: true,
          prohibitedActions: [
            "create Finding", "set severity", "close Finding", "enforcement action",
          ],
        },
        draft: {
          findingId: "FND-SKYCARGO-2026-099",
          prompt: "Draft an evidence request only.",
          draft:
            "Advisory draft for CAR-2026-099: review the configured finding basis and request only the expected evidence.",
          advisoryOnly: true,
          canCreateFinding: false,
          canSetSeverity: false,
          canCloseFinding: false,
        },
        reportDefinitions: [{
          id: "ADMIN-RPT-PACKAGE-001",
          title: "Inspection package configuration preview",
          description: "Typed mock report definition; this is not a real report or PDF engine.",
          packageFields: [
            "packageId", "auditId", "organizationId", "questionIds",
            "configuredReferences", "expectedEvidence", "riskFocus",
          ],
          actionReason:
            "ADMIN-RPT-PACKAGE-001 generation is unavailable because Task 10 provides a typed browser-local preview only.",
        }],
        directory: expectedDirectory,
        organizations: [{
          id: "ORG-FLY-NAMIBIA",
          legalName: "Fly Namibia",
          organizationType: "OPERATOR",
          status: "ACTIVE",
          scope: "CAA oversight",
          detailAvailable: true,
          disabledReason: null,
        }],
      });
    });

    it("replays the same direct command and rejects operation ID payload drift", async () => {
      const harness = await createHarness();
      const inspector = harness.backendFor(PRINCIPALS.inspector);
      const input = {
        operationId: "OP-IDEMPOTENT-RESPONSE",
        responseId: "RESP-CAB-EMEQ-PBE-001",
        auditId: "AUD-2026-001",
        questionId: "CAB-EMEQ-PBE-001",
        expectedResponseRevision: null,
        answer: "NON_COMPLIANT" as const,
        comment: "Deterministic idempotent response.",
      };
      const first = await inspector.inspections.upsertChecklistResponse(input);
      const replay = await inspector.inspections.upsertChecklistResponse(input);
      expect(replay).toEqual(first);
      await expect(
        inspector.inspections.upsertChecklistResponse({ ...input, comment: "Changed payload" }),
      ).rejects.toThrow(/operation ID/i);
    });

    it("returns a deterministic sync acknowledgement and exact replay", async () => {
      const harness = await createHarness();
      const inspector = harness.backendFor(PRINCIPALS.inspector);
      const checkout = await inspector.inspections.checkout({
        operationId: "OP-SYNC-CONTRACT-CHECKOUT",
        packageId: "PKG-CAB-2026-001",
        expectedPackageVersion: 1,
        deviceInstanceId: "DEVICE-CANDIDATE-001",
      });
      const request = {
        operation: {
          operationId: "OP-SYNC-CONTRACT",
          protocolVersion: 1,
          offlineGrantId: checkout.offlineGrant.grantId,
          packageId: checkout.offlineGrant.packageId,
          packageVersion: checkout.offlineGrant.packageVersion,
          entityId: "RESP-CAB-EMEQ-PBE-001",
          commandType: "UPSERT_CHECKLIST_RESPONSE" as const,
          baseRevision: null,
          deviceInstanceId: checkout.offlineGrant.deviceInstanceId,
          clientOccurredAt: FIXED_NOW,
          payload: {
            auditId: "AUD-2026-001",
            questionId: "CAB-EMEQ-PBE-001",
            answer: "NON_COMPLIANT" as const,
            comment: "Sync envelope contract only.",
          },
        },
      };
      const first = await inspector.sync.pushOperation(request);
      const replay = await inspector.sync.pushOperation(request);
      expect(first).toMatchObject({
        operationId: "OP-SYNC-CONTRACT",
        status: "accepted",
        authoritativeEntityId: "RESP-CAB-EMEQ-PBE-001",
      });
      expect(replay).toEqual(first);
    });

    it("keeps Organization Registry projections role- and organization-scoped", async () => {
      const harness = await createHarness();
      const internal = await harness.backendFor(PRINCIPALS.manager).organizations.list({});
      expect(internal.items.map(({ id }) => id).filter((id) => id !== "ORG-SYNTHETIC-AOC")).toEqual([
        "ORG-FLY-NAMIBIA",
        "ORG-SKYCARGO",
      ]);
      expect(internal.items[0]).toMatchObject({
        legalName: "Fly Namibia",
        openFindingCount: 0,
        nextAuditDate: "2026-07-15",
      });

      const auditee = await harness.backendFor(PRINCIPALS.auditee).organizations.list({});
      expect(auditee.items).toHaveLength(1);
      expect(auditee.items[0]?.id).toBe("ORG-FLY-NAMIBIA");
      expect(JSON.stringify(auditee)).not.toMatch(
        /SkyCargo|internalRisk|inspectorWorkload|Internal CAA Note/i,
      );
    });

    it("advances the exact surveillance plan through Finance, GM, Executive Director, and GM release", async () => {
      const harness = await createHarness();
      const finance = harness.backendFor(PRINCIPALS.finance);
      const initial = (await finance.planning.list({})).items[0]!;
      expect(initial).toMatchObject({
        id: "PLAN-2026-CAB-001",
        status: "FINANCE_REVIEW",
        currentOwnerRole: "finance",
        revision: 1,
      });
      await expect(
        harness.backendFor(PRINCIPALS.manager).planning.decide({
          operationId: "OP-PLAN-WRONG-AUTHORITY",
          planningItemId: initial.id,
          expectedPlanningRevision: initial.revision,
          decision: "APPROVE_BUDGET",
          reason: "Manager cannot approve the budget gate.",
        }),
      ).rejects.toThrow(/Finance/i);

      const budgetApproved = await finance.planning.decide({
        operationId: "OP-PLAN-FINANCE-APPROVE",
        planningItemId: initial.id,
        expectedPlanningRevision: initial.revision,
        decision: "APPROVE_BUDGET",
        reason: "Budget envelope confirmed for the configured inspection.",
      });
      expect(budgetApproved).toMatchObject({
        status: "GM_REVIEW",
        currentOwnerRole: "gm",
        revision: 2,
      });

      const gm = harness.backendFor(PRINCIPALS.gm);
      const forwarded = await gm.planning.decide({
        operationId: "OP-PLAN-GM-FORWARD",
        planningItemId: initial.id,
        expectedPlanningRevision: budgetApproved.revision,
        decision: "FORWARD_FOR_FINAL_APPROVAL",
        reason: "Operational scope is ready for final approval.",
      });
      expect(forwarded).toMatchObject({
        status: "EXECUTIVE_DIRECTOR_REVIEW",
        currentOwnerRole: "executiveDirector",
        revision: 3,
      });

      const executive = harness.backendFor(PRINCIPALS.executiveDirector);
      const approved = await executive.planning.decide({
        operationId: "OP-PLAN-EXECUTIVE-APPROVE",
        planningItemId: initial.id,
        expectedPlanningRevision: forwarded.revision,
        decision: "APPROVE_PLAN",
        reason: "The annual surveillance item is approved for release.",
      });
      expect(approved).toMatchObject({
        status: "GM_RELEASE",
        currentOwnerRole: "gm",
        revision: 4,
      });

      const released = await gm.planning.decide({
        operationId: "OP-PLAN-GM-RELEASE",
        planningItemId: initial.id,
        expectedPlanningRevision: approved.revision,
        decision: "RELEASE_PLAN",
        reason: "Release the approved item to department preparation.",
      });
      expect(released).toMatchObject({
        status: "RELEASED",
        currentOwnerRole: "manager",
        revision: 5,
      });
      await expect(
        finance.planning.decide({
          operationId: "OP-PLAN-STALE",
          planningItemId: initial.id,
          expectedPlanningRevision: 1,
          decision: "APPROVE_BUDGET",
          reason: "Stale replay must fail.",
        }),
      ).rejects.toThrow(/revision/i);

      const admin = harness.backendFor(PRINCIPALS.admin);
      const [templates, reminders, auditEvents] = await Promise.all([
        admin.configuration.listChecklistTemplateVersions({}),
        admin.configuration.listReminderRules({}),
        admin.auditTrail.list({ entityType: "SURVEILLANCE_PLAN", entityId: initial.id }),
      ]);
      expect(templates.items[0]).toMatchObject({
        id: "CTV-CABIN-1",
        templateId: "CABIN",
        version: 1,
        status: "PUBLISHED",
        questionCount: 6,
      });
      expect(reminders.items.map(({ offsetDays }) => offsetDays)).toEqual([30, 15, 7, 0, -1]);
      expect(auditEvents.items.map(({ action }) => action)).toEqual([
        "PLANNING_BUDGET_APPROVED",
        "PLANNING_FORWARDED_FOR_FINAL_APPROVAL",
        "PLANNING_APPROVED",
        "PLANNING_RELEASED",
      ]);
      expect(JSON.stringify(auditEvents)).not.toMatch(/internalCaaNote/i);
    });

    it("produces the same normalized planning, package, team, and Auditee coordination transcript", async () => {
      const harness = await createHarness();
      const manager = harness.backendFor(PRINCIPALS.manager);
      if (!manager.planningIntake || !manager.packageDrafts || !manager.teams) {
        throw new Error("Task 4 manager capabilities are unavailable.");
      }
      const auditee = harness.backendFor(PRINCIPALS.auditee);
      if (!auditee.auditeeCoordination) {
        throw new Error("Task 4 Auditee coordination capability is unavailable.");
      }

      const transcript: unknown[] = [];
      const initialDraft = await manager.planningIntake.getDraft({
        draftId: "PLAN-DRAFT-2026-001",
      });
      transcript.push({
        event: "planning.draft.loaded",
        id: initialDraft.id,
        noticePolicy: initialDraft.noticePolicy,
        requestedBudget: initialDraft.requestedBudget,
        revision: initialDraft.revision,
      });
      const savedDraft = await manager.planningIntake.saveDraft({
        idempotencyKey: "IDEM-TASK4-CONTRACT-DRAFT",
        expectedRevision: initialDraft.revision,
        draftId: initialDraft.id,
        values: {
          organizationId: "ORG-FLY-NAMIBIA",
          organizationName: "Fly Namibia",
          applicationType: "Continued Surveillance",
          domain: "Cabin Safety",
          inspectionCategory: "Routine / Announced",
          noticePolicy: "ADVANCE",
          purpose: "Annual routine oversight contract transcript.",
          triggerType: "Department Manager initiated",
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
      transcript.push({
        event: "planning.draft.saved",
        noticePolicy: savedDraft.noticePolicy,
        purpose: savedDraft.purpose,
        location: savedDraft.location,
        revision: savedDraft.revision,
      });
      const submitted = await manager.planningIntake.submit({
        idempotencyKey: "IDEM-TASK4-CONTRACT-SUBMIT",
        expectedRevision: savedDraft.revision,
        draftId: savedDraft.id,
        planningItemId: "PLAN-2026-TASK4-CONTRACT",
      });
      transcript.push({
        event: "planning.intake.submitted",
        draftRevision: submitted.draft.revision,
        submittedPlanningItemId: submitted.draft.submittedPlanningItemId,
        planningStatus: submitted.planningItem.status,
        ownerRole: submitted.planningItem.currentOwnerRole,
        requestedBudget: submitted.planningItem.estimatedBudget,
      });

      const initialPackage = await manager.packageDrafts.get({
        packageDraftId: "PKG-AUD-2026-001-CABIN",
      });
      transcript.push({
        event: "package.draft.loaded",
        id: initialPackage.id,
        applicationType: initialPackage.applicationType,
        domain: initialPackage.domain,
        riskFocus: initialPackage.riskFocus,
        questionIds: initialPackage.questions.map(({ id }) => id),
        revision: initialPackage.revision,
      });
      const savedPackage = await manager.packageDrafts.save({
        idempotencyKey: "IDEM-TASK4-CONTRACT-PACKAGE",
        expectedRevision: initialPackage.revision,
        packageDraftId: initialPackage.id,
        riskFocus: ["PBE serviceability", "Cabin inspection CAP follow-up"],
      });
      transcript.push({
        event: "package.draft.saved",
        riskFocus: savedPackage.riskFocus,
        questionIds: savedPackage.questions.map(({ id }) => id),
        revision: savedPackage.revision,
      });

      const leads = await manager.teams.list({ role: "leadInspector" });
      transcript.push({
        event: "team.leads.listed",
        items: leads.items.map(({ subjectId, displayName, role }) => ({
          subjectId,
          displayName,
          role,
        })),
      });

      const coordination = await auditee.auditeeCoordination.list({});
      transcript.push({
        event: "auditee.coordination.listed",
        items: coordination.items.map((item) => ({
          auditId: item.auditId,
          organizationId: item.organizationId,
          scheduledStartDate: item.scheduledStartDate,
          status: item.status,
          revision: item.revision,
        })),
      });
      const confirmed = await auditee.auditeeCoordination.respond({
        idempotencyKey: "IDEM-TASK4-CONTRACT-COORDINATION",
        expectedRevision: coordination.items[0]!.revision,
        auditId: coordination.items[0]!.auditId,
        organizationId: coordination.items[0]!.organizationId,
        decision: "CONFIRM",
        alternativeDate: null,
      });
      transcript.push({
        event: "auditee.coordination.confirmed",
        auditId: confirmed.auditId,
        status: confirmed.status,
        alternativeDate: confirmed.alternativeDate,
        revision: confirmed.revision,
      });

      expect(transcript).toEqual([
        {
          event: "planning.draft.loaded",
          id: "PLAN-DRAFT-2026-001",
          noticePolicy: "ADVANCE",
          requestedBudget: 0,
          revision: 1,
        },
        {
          event: "planning.draft.saved",
          noticePolicy: "ADVANCE",
          purpose: "Annual routine oversight contract transcript.",
          location: "Windhoek",
          revision: 2,
        },
        {
          event: "planning.intake.submitted",
          draftRevision: 3,
          submittedPlanningItemId: "PLAN-2026-TASK4-CONTRACT",
          planningStatus: "FINANCE_REVIEW",
          ownerRole: "finance",
          requestedBudget: 0,
        },
        {
          event: "package.draft.loaded",
          id: "PKG-AUD-2026-001-CABIN",
          applicationType: "Cabin Inspection",
          domain: "Cabin Safety",
          riskFocus: [
            "Emergency equipment serviceability",
            "PBE serviceability",
            "Cabin inspection CAP follow-up",
          ],
          questionIds: ["PKG-Q-CAB-PBE", "PKG-Q-CAB-GALLEY"],
          revision: 1,
        },
        {
          event: "package.draft.saved",
          riskFocus: ["PBE serviceability", "Cabin inspection CAP follow-up"],
          questionIds: ["PKG-Q-CAB-PBE", "PKG-Q-CAB-GALLEY"],
          revision: 2,
        },
        {
          event: "team.leads.listed",
          items: [{
            subjectId: "USR-LEAD-CANER",
            displayName: "Caner Lead Inspector",
            role: "leadInspector",
          }],
        },
        {
          event: "auditee.coordination.listed",
          items: [{
            auditId: "AUD-2026-001",
            organizationId: "ORG-FLY-NAMIBIA",
            scheduledStartDate: "2026-06-15",
            status: "AWAITING_AUDITEE_CONFIRMATION",
            revision: 1,
          }],
        },
        {
          event: "auditee.coordination.confirmed",
          auditId: "AUD-2026-001",
          status: "CONFIRMED",
          alternativeDate: null,
          revision: 2,
        },
      ]);
    });

    it("produces the same normalized checklist and configuration transcript", async () => {
      const harness = await createHarness();
      const admin = harness.backendFor(PRINCIPALS.admin);
      const manager = harness.backendFor(PRINCIPALS.manager);
      if (!admin.adminWorkspace || !manager.adminWorkspace) {
        throw new Error("Task 5 Admin workspace capability is unavailable.");
      }
      await expect(
        manager.adminWorkspace.listQuestions({}),
      ).rejects.toThrow();

      const transcript: Array<Record<string, unknown>> = [];
      const references = await admin.adminWorkspace.listRegulatoryReferences({});
      transcript.push({
        event: "configuration.references.listed",
        items: references.items.map(({ id, version, status }) => ({ id, version, status })),
      });
      const masters = await admin.adminWorkspace.listTemplateMasters({});
      transcript.push({
        event: "configuration.templates.listed",
        items: masters.items.map(({
          id,
          publishedVersionId,
          owner,
          itemCount,
          previewPath,
          disabledReason,
          revision,
        }) => ({
          id,
          publishedVersionId,
          owner,
          itemCount,
          previewPath,
          disabledReason,
          revision,
        })),
      });
      const questions = await admin.adminWorkspace.listQuestions({});
      transcript.push({
        event: "configuration.questions.listed",
        ids: questions.items.map(({ id }) => id).sort(),
      });
      const question = await admin.adminWorkspace.createQuestion({
        idempotencyKey: "IDEM-TASK5-CONTRACT-QUESTION",
        expectedRevision: null,
        prompt:
          "Is the multiline emergency equipment record complete?\nDoes it identify the exact cabin position?",
        configuredReference: "Configured Cabin Inspection reference — EM EQ / PBE",
        expectedEvidence: "PBE serviceability record\nCabin position confirmation",
      });
      transcript.push({
        event: "configuration.question.created",
        id: question.id,
        prompt: question.prompt,
        expectedEvidence: question.expectedEvidence,
        revision: question.revision,
      });

      const template = await admin.adminWorkspace.getTemplate({
        templateId: "TPL-CABIN-2026",
      });
      transcript.push({
        event: "configuration.template.loaded",
        id: template.id,
        publishedVersionId: template.publishedVersionId,
        versionIds: template.versions.map(({ id }) => id),
        publishedQuestionIds: template.versions[0]!.questionIds,
        revision: template.revision,
      });
      const draft = await admin.adminWorkspace.createDraft({
        idempotencyKey: "IDEM-TASK5-CONTRACT-DRAFT",
        expectedRevision: template.revision,
        templateId: template.id,
        changeReason: "Add the multiline emergency equipment Question.",
      });
      transcript.push({
        event: "configuration.template.draft.created",
        id: draft.id,
        status: draft.status,
        owner: draft.owner,
        questionCount: draft.questionIds.length,
        revision: draft.revision,
      });
      const added = await admin.adminWorkspace.addDraftQuestion({
        idempotencyKey: "IDEM-TASK5-CONTRACT-ADD",
        expectedRevision: draft.revision,
        templateId: template.id,
        draftVersionId: draft.id,
        questionId: question.id,
      });
      transcript.push({
        event: "configuration.template.question.added",
        questionCount: added.questionIds.length,
        lastQuestionId: added.questionIds.at(-1),
        revision: added.revision,
      });
      const moved = await admin.adminWorkspace.moveDraftQuestion({
        idempotencyKey: "IDEM-TASK5-CONTRACT-MOVE",
        expectedRevision: added.revision,
        templateId: template.id,
        draftVersionId: added.id,
        questionId: question.id,
        direction: "UP",
      });
      transcript.push({
        event: "configuration.template.question.moved",
        secondLastQuestionId: moved.questionIds.at(-2),
        revision: moved.revision,
      });
      const inspectionPackage = await admin.adminWorkspace.getInspectionPackage({
        packageId: "PKG-CAB-2026-001",
      });
      transcript.push({
        event: "configuration.package.loaded",
        id: inspectionPackage.id,
        auditId: inspectionPackage.auditId,
        questionCount: inspectionPackage.questionIds.length,
        configuredReferenceCount: inspectionPackage.configuredReferences.length,
        expectedEvidenceCount: inspectionPackage.expectedEvidence.length,
        riskFocusCount: inspectionPackage.riskFocus.length,
        includesDraftQuestion: inspectionPackage.questionIds.includes(question.id),
      });

      const inspector = harness.backendFor(PRINCIPALS.inspector);
      const response = await inspector.inspections.upsertChecklistResponse({
        operationId: "OP-TASK5-CONTRACT-RESPONSE",
        responseId: "RESP-TASK5-CONTRACT-PBE",
        auditId: "AUD-2026-001",
        questionId: "CAB-EMEQ-PBE-001",
        expectedResponseRevision: null,
        answer: "COMPLIANT",
        comment: "",
      });
      const completePackage = await inspector.inspections.getPackage({
        packageId: "PKG-CAB-2026-001",
      });
      for (const packageQuestion of completePackage.questions) {
        if (packageQuestion.currentResponse) continue;
        const assignedInspector = packageQuestion.assignedInspectorUserIds.includes(PRINCIPALS.inspector.subjectId)
          || packageQuestion.assignedInspectorUserIds.includes(HTTP_CANONICAL_INSPECTOR_SUBJECT_ID)
          ? PRINCIPALS.inspector
          : packageQuestion.assignedInspectorUserIds.includes(PRINCIPALS.otherInspector.subjectId)
            ? PRINCIPALS.otherInspector
            : null;
        if (!assignedInspector) {
          throw new Error(`No canonical Inspector principal is assigned to ${packageQuestion.id}.`);
        }
        await harness.backendFor(assignedInspector).inspections.upsertChecklistResponse({
          operationId: `OP-TASK5-CONTRACT-COMPLETE-${packageQuestion.id}`,
          responseId: `RESP-TASK5-CONTRACT-${packageQuestion.id}`,
          auditId: completePackage.auditId,
          questionId: packageQuestion.id,
          expectedResponseRevision: null,
          answer: "COMPLIANT",
          comment: "",
        });
      }
      const submitted = await inspector.inspections.submitChecklist({
        operationId: "OP-TASK5-CONTRACT-SUBMIT",
        auditId: "AUD-2026-001",
        expectedChecklistRevision: 1,
      });
      const reopened = await harness.backendFor(PRINCIPALS.leadInspector)
        .inspections.reopenChecklist({
          operationId: "OP-TASK5-CONTRACT-REOPEN",
          auditId: "AUD-2026-001",
          expectedChecklistRevision: submitted.checklistRevision,
          reason: "Continue exact configured sampling.",
        });
      transcript.push({
        event: "checklist.execution.completed",
        responseRevision: response.revision,
        submittedStatus: submitted.checklistStatus,
        submittedRevision: submitted.checklistRevision,
        reopenedStatus: reopened.checklistStatus,
        reopenedRevision: reopened.checklistRevision,
      });

      expect(transcript).toEqual([
        {
          event: "configuration.references.listed",
          items: [
            { id: "NAMCARS-CAB-001", version: "2026.1", status: "ACTIVE" },
            { id: "NAMCARS-FOPS-004", version: "2025.4", status: "SUPERSEDED" },
          ],
        },
        {
          event: "configuration.templates.listed",
          items: [
            {
              id: "TPL-CABIN-2026",
              publishedVersionId: "CTV-CABIN-1",
              owner: "Department Manager",
              itemCount: 6,
              previewPath: "/admin/templates",
              disabledReason: null,
              revision: 1,
            },
            {
              id: "TPL-FOPS-2026",
              publishedVersionId: "CTV-FOPS-1",
              owner: "Department Manager",
              itemCount: 0,
              previewPath: null,
              disabledReason:
                "TPL-FOPS-2026 / CTV-FOPS-1 has no declared Template Preview route in Task 10.",
              revision: 1,
            },
          ],
        },
        {
          event: "configuration.questions.listed",
          ids: [
            "CAB-COCKPIT-GEN-001",
            "CAB-EMEQ-PBE-001",
            "CAB-GALLEY-001",
            "CAB-LAV-001",
            "CAB-PAX-SEAT-001",
            "CAB-VID-CREW-SEAT-001",
          ],
        },
        {
          event: "configuration.question.created",
          id: "Q-ADMIN-2026-007",
          prompt:
            "Is the multiline emergency equipment record complete?\nDoes it identify the exact cabin position?",
          expectedEvidence: "PBE serviceability record\nCabin position confirmation",
          revision: 1,
        },
        {
          event: "configuration.template.loaded",
          id: "TPL-CABIN-2026",
          publishedVersionId: "CTV-CABIN-1",
          versionIds: ["CTV-CABIN-1"],
          publishedQuestionIds: [
            "CAB-GALLEY-001",
            "CAB-LAV-001",
            "CAB-PAX-SEAT-001",
            "CAB-EMEQ-PBE-001",
            "CAB-VID-CREW-SEAT-001",
            "CAB-COCKPIT-GEN-001",
          ],
          revision: 1,
        },
        {
          event: "configuration.template.draft.created",
          id: "CTV-CABIN-DRAFT-2",
          status: "DRAFT",
          owner: "Admin Preview",
          questionCount: 6,
          revision: 1,
        },
        {
          event: "configuration.template.question.added",
          questionCount: 7,
          lastQuestionId: "Q-ADMIN-2026-007",
          revision: 2,
        },
        {
          event: "configuration.template.question.moved",
          secondLastQuestionId: "Q-ADMIN-2026-007",
          revision: 3,
        },
        {
          event: "configuration.package.loaded",
          id: "PKG-CAB-2026-001",
          auditId: "AUD-2026-001",
          questionCount: 6,
          configuredReferenceCount: 6,
          expectedEvidenceCount: 6,
          riskFocusCount: 3,
          includesDraftQuestion: false,
        },
        {
          event: "checklist.execution.completed",
          responseRevision: 1,
          submittedStatus: "SUBMITTED",
          submittedRevision: 2,
          reopenedStatus: "IN_PROGRESS",
          reopenedRevision: 3,
        },
      ]);
    });

    it("exposes immutable checklist template detail only to Admin", async () => {
      const harness = await createHarness();
      const admin = harness.backendFor(PRINCIPALS.admin);
      const detail = await admin.configuration.getChecklistTemplateVersion({
        templateVersionId: "CTV-CABIN-1",
      });

      expect(detail).toMatchObject({
        id: "CTV-CABIN-1",
        templateId: "CABIN",
        version: 1,
        status: "PUBLISHED",
        questionCount: 6,
      });
      expect(detail.questions).toHaveLength(6);
      expect(detail.questions.find((question) => question.id === "CAB-EMEQ-PBE-001")).toMatchObject({
        sectionId: "EM EQ / PBE",
        expectedEvidence: "PBE serviceability record and cabin position confirmation",
        allowedAnswers: [
          "COMPLIANT",
          "NON_COMPLIANT",
          "OBSERVATION",
          "NOT_APPLICABLE",
          "NOT_CHECKED",
        ],
        commentRequiredFor: ["NON_COMPLIANT", "OBSERVATION"],
      });
      expect(JSON.stringify(detail)).not.toMatch(
        /assignedInspectorUserIds|currentResponse|internalCaaNote|draft|secret/i,
      );

      for (const principal of [
        PRINCIPALS.inspector,
        PRINCIPALS.leadInspector,
        PRINCIPALS.manager,
        PRINCIPALS.finance,
        PRINCIPALS.gm,
        PRINCIPALS.executiveDirector,
        PRINCIPALS.auditee,
      ]) {
        await expect(
          harness.backendFor(principal).configuration.getChecklistTemplateVersion({
            templateVersionId: "CTV-CABIN-1",
          }),
        ).rejects.toThrow(/Admin configuration authority/i);
      }
    });
  });
}
