import { describe, expect, it } from "vitest";

import {
  mapCapRevision,
  mapCapRevisions,
  mapChecklistTemplateVersionDetail,
  mapFinding,
  mapAdminRegulatoryReference,
  mapInspectionPackage,
  mapManagerDashboard,
  mapRiskManagementProjection,
} from "./transport-mappers";

describe("transport mappers", () => {
  it("deep-maps regulatory trace arrays and proposed Evidence examples", () => {
    const transport = {
      id: "NAMCARS-CAB-001",
      title: "Cabin regulatory references",
      version: "2026.1",
      status: "ACTIVE" as const,
      effectiveDate: "2026-01-01",
      configuredRules: ["Rule"],
      changeHistory: ["Change"],
      mappings: [{
        id: "RMAP-OPS-AOC-CABIN-RAMP-001",
        auditArea: "OPS",
        serviceProviderTypes: ["Air Operator (AOC)"],
        applicableRegulations: ["Part 121"],
        criticalElement: "CE-7",
        protocolQuestionId: "4.450",
        protocolQuestion: "Does the surveillance programme include ramp inspections?",
        annexReferences: ["Annex 6 Part I 4.2.2.2"],
        nationalReferences: ["NAMCAR 121.07.6"],
        caaImplementationReference: "Controlled procedure not supplied",
        requirement: "Execute risk-based ramp surveillance.",
        verificationObjective: "Verify selected cabin safety controls.",
        expectedEvidence: ["Inspector observation"],
        whyIncluded: "Candidate decomposition.",
        reviewStatus: "EXPERT_REVIEW_REQUIRED" as const,
        sourceGap: "Controlled procedure not supplied.",
        refreshPolicy: {
          sourceCollectionId: "NCAA-NAMCATS-ALL-PAGES",
          lastCheckedAt: "2026-07-28T00:00:00Z",
          nextReconciliationDate: "2027-01-28",
          nextExpertValidationDate: "2027-07-28",
          eventDrivenReview: true as const,
          reconciliationIntervalMonths: 6 as const,
          expertValidationIntervalMonths: 12 as const,
          sourceChangeState: "BASELINE_CAPTURED" as const,
          updateMode: "PROPOSE_DRAFT_ONLY" as const,
          documentCount: 58,
          manifestPath: "docs/regulatory-sources/ncaa-namcats-manifest.json",
          guardrails: ["Draft only"],
        },
        scopeRecommendation: {
          id: "SCOPE-REC-001",
          status: "ADVISORY_ONLY" as const,
          historyState: "INSUFFICIENT_FOR_DEFERRAL" as const,
          generatedAt: "2026-07-28T00:00:00Z",
          signals: ["Open Finding"],
          guardrails: ["No automatic omission"],
          questionRecommendations: [{
            questionId: "CAB-EMEQ-PBE-001",
            classification: "FOCUSED_FULL" as const,
            rationale: "Open PBE Finding.",
            historyBasis: "FND-CAB-2026-001",
            requiresManagerApproval: true as const,
          }],
        },
        sources: [{
          id: "ICAO-PQ-OPS-2024-R1.1",
          title: "2024 OPS PQs",
          sourceType: "ICAO_PQ" as const,
          version: "September 2024 Revision 1.1",
          status: "SUPPLIED_WORKING_COPY" as const,
          locator: "PQ 4.450",
          url: null,
        }],
        proposedQuestions: [{
          id: "CAB-EMEQ-PBE-001",
          prompt: "Is the PBE serviceable?",
          verificationMethod: "Observe and reconcile records.",
          evidenceExamples: ["Observation", "Serviceability record"],
          whyIncluded: "Cabin safety coverage.",
        }],
      }],
    };

    const mapped = mapAdminRegulatoryReference(transport);

    expect(mapped).toEqual(transport);
    expect(mapped.mappings).not.toBe(transport.mappings);
    expect(mapped.mappings[0]?.sources).not.toBe(transport.mappings[0]?.sources);
    expect(mapped.mappings[0]?.refreshPolicy.guardrails).not.toBe(
      transport.mappings[0]?.refreshPolicy.guardrails,
    );
    expect(mapped.mappings[0]?.scopeRecommendation.questionRecommendations).not.toBe(
      transport.mappings[0]?.scopeRecommendation.questionRecommendations,
    );
    expect(mapped.mappings[0]?.proposedQuestions[0]?.evidenceExamples).not.toBe(
      transport.mappings[0]?.proposedQuestions[0]?.evidenceExamples,
    );
  });

  it("maps generated transport Finding values into independent domain values", () => {
    const transport = {
      id: "FND-CAB-2026-001",
      findingNumber: "CAB-2026-001",
      auditId: "AUD-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      organizationName: "Fly Namibia",
      title: "PBE serviceability and accessibility not confirmed",
      description: "Configured check exception.",
      regulatoryReference: "Configured Cabin Inspection reference — EM EQ / PBE",
      findingBasis: "Exact response and comment",
      severity: "LEVEL_1_CRITICAL" as const,
      status: "EVIDENCE_REQUIRED" as const,
      dueDate: "2026-07-15",
      dueState: "NOT_DUE" as const,
      currentOwnerType: "AUDITEE" as const,
      currentOwnerId: "ORG-FLY-NAMIBIA",
      currentOwnerRole: "auditee" as const,
      nextAction: "Submit Evidence",
      capRequired: true,
      evidenceRequired: true,
      repeatFinding: false,
      createdAt: "2026-06-15T09:00:00.000Z",
      issuedAt: "2026-06-15T09:10:00.000Z",
      closedAt: null,
      closureBasis: null,
      revision: 3,
    };
    const domain = mapFinding(transport);
    expect(domain).toEqual(transport);
    expect(domain).not.toBe(transport);
  });

  it("deep-maps inspection questions and dashboard arrays", () => {
    const packageTransport = {
      id: "PKG-CAB-2026-001",
      auditId: "AUD-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      organizationName: "Fly Namibia",
      title: "2026 Cabin Inspection - Fly Namibia",
      packageVersion: 1,
      schemaVersion: 1,
      protocolVersion: 1,
      templateVersionId: "CTV-CABIN-1",
      packageDigest: "sha256:test",
      expiresAt: "2026-07-15T23:59:59.000Z",
      checklistStatus: "IN_PROGRESS" as const,
      checklistRevision: 1,
      questions: [
        {
          id: "CAB-EMEQ-PBE-001",
          sectionId: "EM EQ / PBE",
          prompt: "PBE question",
          regulatoryReference: null,
          expectedEvidence: null,
          allowedAnswers: ["COMPLIANT" as const],
          commentRequiredFor: ["NON_COMPLIANT" as const],
          assignedInspectorUserIds: ["USR-INSPECTOR-AMINA"],
          currentResponse: null,
        },
      ],
    };
    const mappedPackage = mapInspectionPackage(packageTransport);
    expect(mappedPackage.questions).not.toBe(packageTransport.questions);
    expect(mappedPackage.questions[0]?.assignedInspectorUserIds).not.toBe(
      packageTransport.questions[0]?.assignedInspectorUserIds,
    );

    const transportDashboard = {
      generatedAt: "2026-06-15T09:00:00.000Z",
      openFindings: 1,
      closedFindings: 0,
      overdueFindings: 0,
      pendingCapReviews: 0,
      pendingEvidenceReviews: 0,
      pendingReportReviews: 0,
      recentFindingNumbers: ["CAB-2026-001"],
    };
    expect(mapManagerDashboard(transportDashboard)).toEqual(transportDashboard);
  });

  it("maps role-shaped CAP revisions without leaking Auditee internal CAA notes", () => {
    const caa = mapCapRevision({
      audience: "CAA" as const,
      id: "CAP-CAB-2026-001-R1",
      capId: "CAP-CAB-2026-001",
      findingId: "FND-CAB-2026-001",
      organizationId: "ORG-FLY-NAMIBIA",
      revision: 1,
      status: "ACCEPTED" as const,
      rootCause: "Root cause",
      correctiveAction: "Corrective action",
      preventiveAction: "Preventive action",
      responsiblePerson: "Fly Namibia Cabin Safety Manager",
      targetCompletionDate: "2026-07-15",
      commentToCaa: "CAP submitted for CAA review.",
      submittedAt: "2026-06-15T09:00:00.000Z",
      latestReview: {
        decision: "ACCEPT" as const,
        commentToAuditee: "CAP accepted.",
        internalCaaNote: "Internal note.",
        decidedAt: "2026-06-15T09:00:00.000Z",
      },
    });
    expect(caa.audience).toBe("CAA");
    if (caa.audience !== "CAA") throw new Error("Expected CAA CAP revision view.");
    expect(caa.latestReview?.internalCaaNote).toBe("Internal note.");

    const auditee = mapCapRevision({
      ...caa,
      audience: "AUDITEE" as const,
      latestReview: {
        decision: "ACCEPT" as const,
        commentToAuditee: "CAP accepted.",
        decidedAt: "2026-06-15T09:00:00.000Z",
      },
    });
    expect(JSON.stringify(auditee)).not.toMatch(/internalCaaNote/i);
    expect(mapCapRevisions({ items: [caa, auditee], nextCursor: null }).items).toHaveLength(2);
  });

  it("deep-maps checklist template detail questions without inspection-only fields", () => {
    const transport = {
      id: "CTV-CABIN-1",
      templateId: "CABIN",
      title: "Cabin Inspection checklist",
      version: 1,
      status: "PUBLISHED" as const,
      publishedAt: "2026-06-15T09:00:00.000Z",
      questionCount: 1,
      questions: [
        {
          id: "CAB-EMEQ-PBE-001",
          sectionId: "EM EQ / PBE",
          prompt: "PBE question",
          regulatoryReference: "Configured Cabin Inspection reference - EM EQ / PBE",
          expectedEvidence: "PBE serviceability record and cabin position confirmation",
          allowedAnswers: [
            "COMPLIANT" as const,
            "NON_COMPLIANT" as const,
            "OBSERVATION" as const,
            "NOT_APPLICABLE" as const,
            "NOT_CHECKED" as const,
          ],
          commentRequiredFor: ["NON_COMPLIANT" as const, "OBSERVATION" as const],
        },
      ],
    };
    const detail = mapChecklistTemplateVersionDetail(transport);
    expect(detail).toEqual(transport);
    expect(detail).not.toBe(transport);
    expect(detail.questions).not.toBe(transport.questions);
    expect(detail.questions[0]?.allowedAnswers).not.toBe(transport.questions[0]?.allowedAnswers);
    expect(JSON.stringify(detail)).not.toMatch(/assignedInspectorUserIds|currentResponse/i);
  });

  it("drops unexpected private fields from the governed risk projection", () => {
    const transport = {
      findings: [{
        findingId: "FND-CAB-2026-001",
        findingNumber: "CAB-2026-001",
        organizationId: "ORG-FLY-NAMIBIA",
        organizationName: "Fly Namibia",
        inspectionId: "AUD-2026-001",
        inspectionTitle: "2026 Cabin Inspection - Fly Namibia",
        department: null,
        title: "PBE serviceability and accessibility not confirmed",
        severity: "LEVEL_1_CRITICAL",
        riskLevel: "HIGH",
        status: "OPEN",
        issuedAt: "2026-06-15T09:00:00.000Z",
        dueState: "OVERDUE",
        capRequired: true,
        internalCaaNote: "PRIVATE_RISK_NOTE",
      }],
      capEffectiveness: [],
      generatedAt: "2026-06-15T09:00:00.000Z",
      revision: 1,
      privateEnforcementScore: 99,
    } as unknown as Parameters<typeof mapRiskManagementProjection>[0];

    const mapped = mapRiskManagementProjection(transport);

    expect(JSON.stringify(mapped)).not.toMatch(
      /internalCaaNote|PRIVATE_RISK_NOTE|privateEnforcementScore/i,
    );
  });
});
