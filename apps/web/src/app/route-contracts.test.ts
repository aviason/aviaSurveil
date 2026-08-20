import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

import auditSource from "../parity/legacy-screen-source.json";
import { REACT_ROUTE_CONTRACTS } from "./route-contracts";
import { SCREEN_COMPONENT_REGISTRY } from "./screen-component-registry";

const roleByAuditRole = {
  Global: null,
  "CAA Inspector": "inspector",
  "Lead Inspector": "leadInspector",
  "Department Manager": "manager",
  "General Manager": "gm",
  Finance: "finance",
  "Executive Director": "executiveDirector",
  Auditee: "auditee",
  "Admin Preview": "admin",
} as const;

describe("React route contracts", () => {
  it("keeps the 86 active audit surfaces ordered after adding post-release checklist preparation", () => {
    expect(REACT_ROUTE_CONTRACTS).toHaveLength(86);
    const expectedAuditIds = auditSource.filter(({ auditId }) => auditId !== "ui-audit-043").map(({ auditId }) => auditId);
    expectedAuditIds.splice(expectedAuditIds.indexOf("ui-audit-052"), 0, "ui-audit-087");
    expect(REACT_ROUTE_CONTRACTS.map(({ auditId }) => auditId)).toEqual(expectedAuditIds);
    expect(REACT_ROUTE_CONTRACTS.find(({ auditId }) => auditId === "ui-audit-087")).toMatchObject({ id: "post-release-checklist-selection" });
    expect(new Set(REACT_ROUTE_CONTRACTS.map(({ id }) => id)).size).toBe(86);
    expect(new Set(REACT_ROUTE_CONTRACTS.map(({ path }) => path)).size).toBe(86);
  });

  it("declares a role, parent, placement, data boundary, component, and profile availability for every route", () => {
    for (const contract of REACT_ROUTE_CONTRACTS) {
      expect(contract).toMatchObject({
        auditId: expect.stringMatching(/^ui-audit-\d{3}$/),
        path: expect.stringMatching(/^\//),
        placement: expect.any(String),
        dataBoundary: expect.any(String),
        componentKey: expect.any(String),
        availableProfiles: expect.any(Array),
      });
      expect(contract.availableProfiles.length).toBeGreaterThan(0);
      if (contract.placement === "contextual") expect(contract.parentId).not.toBeNull();
    }
  });

  it("activates all 86 current contracts in both demo and HTTP profiles", () => {
    const dualProfile = REACT_ROUTE_CONTRACTS.filter(
      ({ availableProfiles }) => availableProfiles.join(":") === "demo:http",
    );
    const demoOnly = REACT_ROUTE_CONTRACTS.filter(
      ({ availableProfiles }) => availableProfiles.join(":") === "demo",
    );

    expect(dualProfile).toHaveLength(86);
    expect(demoOnly).toHaveLength(0);
    expect(REACT_ROUTE_CONTRACTS.every(({ blockedProfileReason }) => blockedProfileReason === undefined)).toBe(true);
  });

  it("keeps every role-owned audit row aligned with its normalized required role", () => {
    const contractByAuditId = new Map<string, (typeof REACT_ROUTE_CONTRACTS)[number]>(REACT_ROUTE_CONTRACTS.map((contract) => [contract.auditId, contract]));
    for (const row of auditSource.filter(({ auditId }) => auditId !== "ui-audit-043")) {
      expect(contractByAuditId.get(row.auditId)?.requiredRole).toBe(roleByAuditRole[row.role as keyof typeof roleByAuditRole]);
    }
  });

  it("re-homes the inherited crossed Finding and Evidence routes to their source roles", () => {
    const finding = REACT_ROUTE_CONTRACTS.find(({ auditId }) => auditId === "ui-audit-009");
    const evidence = REACT_ROUTE_CONTRACTS.find(({ auditId }) => auditId === "ui-audit-044");

    expect(finding).toMatchObject({
      id: "finding-detail",
      path: "/inspector/findings/:findingId",
      requiredRole: "inspector",
      parentId: "inspector-findings",
    });
    expect(evidence).toMatchObject({
      id: "evidence-review",
      path: "/department-manager/evidence/:findingId",
      requiredRole: "manager",
      parentId: "manager-findings-review",
    });
    expect(evidence?.additionalRoles).toEqual(["leadInspector", "inspector"]);
  });

  it("freezes root-navigation screens as primary routes", () => {
    const primaryAuditIds = REACT_ROUTE_CONTRACTS
      .filter(({ placement }) => placement === "primary")
      .map(({ auditId }) => auditId);

    expect(primaryAuditIds).toEqual([
      "ui-audit-002", "ui-audit-003", "ui-audit-004", "ui-audit-005", "ui-audit-006",
      "ui-audit-013", "ui-audit-014", "ui-audit-016", "ui-audit-023", "ui-audit-024", "ui-audit-025", "ui-audit-026",
      "ui-audit-027", "ui-audit-028", "ui-audit-029", "ui-audit-030", "ui-audit-031", "ui-audit-032", "ui-audit-033", "ui-audit-034", "ui-audit-035",
      "ui-audit-041",
      "ui-audit-052", "ui-audit-053", "ui-audit-054", "ui-audit-055", "ui-audit-056", "ui-audit-057",
      "ui-audit-058",
      "ui-audit-059", "ui-audit-060", "ui-audit-061", "ui-audit-062", "ui-audit-064", "ui-audit-065",
      "ui-audit-066", "ui-audit-067", "ui-audit-068", "ui-audit-069", "ui-audit-071", "ui-audit-072", "ui-audit-073",
      "ui-audit-074", "ui-audit-075", "ui-audit-077", "ui-audit-078", "ui-audit-079", "ui-audit-081", "ui-audit-082", "ui-audit-083", "ui-audit-084", "ui-audit-086",
    ]);
    expect(REACT_ROUTE_CONTRACTS.filter(({ auditId }) => primaryAuditIds.includes(auditId)).every(({ parentId }) => parentId === null)).toBe(true);
  });

  it("freezes every contextual route to its accepted immediate workspace parent", () => {
    const expectedContextualParentById = {
      "audit-detail": "inspector-home",
      "checklist-runner": "audit-detail",
      "finding-detail": "inspector-findings",
      "closure-report-preview": "finding-detail",
      "inspector-assistant": "finding-detail",
      "inspector-profile": "inspector-home",
      "lead-preliminary-report-workflow": "lead-preliminary-reports",
      "lead-final-report-readiness": "lead-final-reports",
      "lead-prepare-final-report": "lead-final-reports",
      "lead-final-report-document": "lead-final-reports",
      "lead-audit-assignment": "lead-home",
      "lead-checklist-question-assignment": "lead-home",
      "cap-review": "lead-home",
      "manager-safety-intelligence": "manager-risk-dashboard",
      "organization-risk-profile": "organization-registry",
      "manager-ssp-nasp": "manager-risk-dashboard",
      "manager-usoap-readiness": "manager-risk-dashboard",
      "manager-cap-effectiveness": "manager-cap-monitoring",
      "organization-detail": "organization-registry",
      "evidence-review": "manager-findings-review",
      "manager-preliminary-report-review": "report-preview",
      "manager-cap-closure-review": "manager-cap-monitoring",
      "new-audit-wizard-1": "audit-plan",
      "new-audit-wizard-2": "audit-plan",
      "new-audit-wizard-3": "audit-plan",
      "new-audit-wizard-4": "audit-plan",
      "new-audit-wizard-5": "audit-plan",
      "post-release-checklist-selection": "audit-plan",
      "executive-report-preview": "executive-final-reports",
      "auditee-report-preview": "auditee-final-reports",
      "admin-home": "admin-template-list",
      "admin-inspection-package-builder": "admin-checklist-builder",
      "admin-organization-detail": "admin-organization-master-data",
    };
    const actualContextualParentById = Object.fromEntries(
      REACT_ROUTE_CONTRACTS
        .filter(({ placement }) => placement === "contextual")
        .map(({ id, parentId }) => [id, parentId]),
    );
    const routeIds = new Set(REACT_ROUTE_CONTRACTS.map(({ id }) => id));

    expect(actualContextualParentById).toEqual(expectedContextualParentById);
    expect(Object.values(actualContextualParentById).every((parentId) => typeof parentId === "string" && routeIds.has(parentId))).toBe(true);
  });

  it("keeps all 85 routed screen components implemented without hidden generic deferred entries", () => {
    const registrySource = readFileSync(resolve(import.meta.dirname, "screen-component-registry.tsx"), "utf8");
    expect(registrySource).not.toMatch(/deferredDemoScreen|DeferredDemoScreen|deferred-demo-screen/i);
    expect(Object.entries(SCREEN_COMPONENT_REGISTRY).filter(([, { status }]) => status === "implemented")).toHaveLength(85);
    expect(Object.entries(SCREEN_COMPONENT_REGISTRY).filter(([, { status }]) => status === "router-owned")).toEqual([
      ["role-select", { status: "router-owned" }],
    ]);
    expect(Object.values(SCREEN_COMPONENT_REGISTRY).filter(({ status }) => status === "pending")).toHaveLength(0);
    expect(Object.values(SCREEN_COMPONENT_REGISTRY).filter(({ status, component }) => status === "pending" && component === undefined)).toHaveLength(0);
  });

  it("keeps the router free of undeclared literal paths and wildcard placeholder renders", () => {
    const router = readFileSync(resolve(import.meta.dirname, "router.tsx"), "utf8");
    expect(router).not.toMatch(/RoleEntryPlaceholder|candidate React entry route|coming soon|generic placeholder/i);
    expect(router).toMatch(/<Route path="\*" element={<Navigate/);
  });
});
