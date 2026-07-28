import { describe, expect, it } from "vitest";

import { REACT_ROUTE_CONTRACTS, type ReactSurfaceId } from "../app/route-contracts";
import { BackendAuthorizationInvariantError } from "../backend/backend-contracts";
import { createMockBackend, DEMO_CAPABILITY_PERMISSION_MATRIX, DEMO_PRINCIPALS } from "./create-mock-backend";
import { MemoryMockStore } from "./memory-mock-store";
import { SCREEN_VISIBLE_ACTIONS } from "./seed-data";

const FIXED_NOW = "2026-06-15T09:00:00.000Z";

// Independent of SCREEN_VISIBLE_ACTIONS: this is the source-action acceptance matrix.
const EXPECTED_ACTION_IDS: Record<ReactSurfaceId, readonly string[]> = {
  "role-select": ["select-inspector", "select-lead-inspector", "select-manager", "select-finance", "select-gm", "select-executive", "select-auditee", "select-admin"],
  "inspector-home": ["open-assignment"], "inspector-findings": ["open-finding"], "inspector-messages": ["compose-message"], "inspector-calendar": ["open-calendar-item"], "inspector-reports": ["preview-report"], "audit-detail": ["start-checklist"], "checklist-runner": ["save-response", "submit-checklist"], "finding-detail": ["open-closure-report"], "closure-report-preview": ["download-closure-report"], "inspector-assistant": ["draft-advisory"], "inspector-profile": ["save-profile"],
  "lead-home": ["review-potential-finding"], "lead-preliminary-reports": ["open-preliminary-report"], "lead-preliminary-report-workflow": ["save-preliminary-draft"], "lead-final-reports": ["open-final-readiness"], "lead-final-report-readiness": ["prepare-final"], "lead-prepare-final-report": ["save-draft", "preview-report"], "lead-final-report-document": ["download-report"], "lead-audit-assignment": ["assign-inspector"], "lead-checklist-question-assignment": ["save-question-assignment"], "cap-review": ["accept-cap", "request-cap-information"], "lead-calendar": ["open-lead-calendar-item"], "lead-messages": ["compose-lead-message"], "lead-analytics-reports": ["download-analytics"], "lead-settings": ["save-lead-settings"],
  "manager-home": ["open-overdue-finding"], "audit-plan": ["create-audit"], "manager-audits": ["open-manager-audit"], "report-preview": ["return-report", "forward-report"], "manager-risk-dashboard": ["open-risk-profile"], "manager-inspection-team": ["open-team-member"], "manager-findings-review": ["open-evidence-review"], "manager-cap-monitoring": ["open-cap-closure"], "manager-checklist-management": ["open-checklist-template"], "manager-safety-intelligence": ["open-safety-intelligence"], "organization-risk-profile": ["open-organization-finding"], "manager-ssp-nasp": ["open-ssp-nasp"], "manager-usoap-readiness": ["open-usoap-gap"], "manager-cap-effectiveness": ["review-effectiveness"], "organization-registry": ["open-organization"], "organization-detail": ["open-organization-history"], "inspection-package-builder": ["save-package"], "evidence-review": ["accept-evidence", "request-evidence-information"], "manager-preliminary-report-review": ["return-preliminary-report", "forward-preliminary-report"], "manager-cap-closure-review": ["authorize-closure"], "new-audit-wizard-1": ["wizard-next", "wizard-save-draft"], "new-audit-wizard-2": ["wizard-back", "wizard-next"], "new-audit-wizard-3": ["wizard-back", "wizard-next"], "new-audit-wizard-4": ["wizard-back", "wizard-next"], "new-audit-wizard-5": ["wizard-back", "wizard-preview", "wizard-submit"],
  "gm-home": ["open-department-summary"], "gm-planning": ["review-plan"], "gm-report-approvals": ["forward-report", "return-report"], "gm-departments": ["open-department"], "gm-risk-dashboard": ["open-summary-risk"], "gm-settings": ["save-gm-settings"], "finance-home": ["approve-budget", "return-budget"],
  "executive-home": ["open-executive-report"], "executive-planning": ["approve-plan", "return-plan"], "executive-preliminary-reports": ["issue-preliminary-report"], "executive-final-reports": ["issue-final-report"], "executive-report-preview": ["download-executive-report"], "executive-notifications": ["open-executive-notification"], "executive-settings": ["save-executive-settings"],
  "auditee-home": ["respond-to-cap"], "auditee-inspection-coordination": ["confirm-coordination"], "auditee-preliminary-reports": ["open-auditee-preliminary-report"], "auditee-final-reports": ["open-auditee-final-report"], "auditee-report-preview": ["download-auditee-report"], "auditee-messages": ["compose-auditee-message"], "auditee-documents": ["download-document"], "auditee-settings": ["save-auditee-settings"],
  "admin-regulatory-library": ["search-regulatory-library"], "admin-template-list": ["open-template"], "admin-home": ["preview-template"], "admin-question-bank": ["add-question"], "admin-checklist-builder": ["save-checklist-builder"], "admin-version-history": ["compare-template-version"], "admin-inspection-package-builder": ["build-inspection-package"], "admin-reports": ["download-admin-report"], "admin-users-roles": ["edit-user-role"], "admin-configurations": ["save-configuration"], "admin-organization-master-data": ["open-master-organization"], "admin-organization-detail": ["save-organization"], "admin-audit-log": ["download-audit-log"],
};

const HIGH_RISK_BINDINGS: Record<string, string> = {
  "cap-review/accept-cap": "caps.review", "cap-review/request-cap-information": "caps.review",
  "evidence-review/accept-evidence": "evidence.review", "evidence-review/request-evidence-information": "evidence.review",
  "manager-cap-closure-review/authorize-closure": "findings.authorizedClose",
  "finance-home/approve-budget": "planning.decide", "finance-home/return-budget": "planning.decide",
  "gm-report-approvals/forward-report": "reports.decide", "gm-report-approvals/return-report": "reports.decide",
  "executive-planning/approve-plan": "planning.decide", "executive-planning/return-plan": "planning.decide",
  "executive-preliminary-reports/issue-preliminary-report": "reports.decide", "executive-final-reports/issue-final-report": "reports.decide",
};

describe("full-screen deterministic demo scenario", () => {
  it("keeps an exhaustive source-faithful action map with non-inert high-risk effects", () => {
    expect(Object.keys(EXPECTED_ACTION_IDS)).toEqual(REACT_ROUTE_CONTRACTS.map((route) => route.id));
    expect(Object.keys(SCREEN_VISIBLE_ACTIONS)).toEqual(REACT_ROUTE_CONTRACTS.map((route) => route.id));
    for (const route of REACT_ROUTE_CONTRACTS) {
      expect(SCREEN_VISIBLE_ACTIONS[route.id].map((action) => action.id)).toEqual(EXPECTED_ACTION_IDS[route.id]);
      for (const action of SCREEN_VISIBLE_ACTIONS[route.id]) {
        const effect = action.effect;
        if (effect.type === "navigation") {
          const target = REACT_ROUTE_CONTRACTS.find((candidate) => candidate.path === effect.target);
          expect(target, `${route.id}/${action.id} target`).toBeDefined();
          expect(target!.id, `${route.id}/${action.id} self target`).not.toBe(route.id);
          if (route.id !== "role-select") expect(target!.requiredRole).toBe(route.requiredRole);
        }
        if (action.effect.type === "capabilityDispatch") {
          expect(DEMO_CAPABILITY_PERMISSION_MATRIX[route.requiredRole!]).toContain(action.effect.capability.split(".")[0]);
        }
      }
    }
    for (const [key, owner] of Object.entries(HIGH_RISK_BINDINGS)) {
      const [screenId, actionId] = key.split("/") as [ReactSurfaceId, string];
      const effect = SCREEN_VISIBLE_ACTIONS[screenId].find((action) => action.id === actionId)!.effect;
      expect(effect).toMatchObject({ type: "modal", confirmCommand: { owner, requiresRevision: true, requiresOperationMetadata: true } });
    }
    expect(SCREEN_VISIBLE_ACTIONS["finance-home"]).toEqual([
      expect.objectContaining({ id: "approve-budget", effect: expect.objectContaining({ type: "modal", dialog: "approve-budget", confirmCommand: expect.objectContaining({ owner: "planning.decide", requiresRevision: true }) }) }),
      expect.objectContaining({ id: "return-budget", effect: expect.objectContaining({ type: "modal", dialog: "return-budget", confirmCommand: expect.objectContaining({ owner: "planning.decide", requiresRevision: true }) }) }),
    ]);
    expect(SCREEN_VISIBLE_ACTIONS["new-audit-wizard-5"].map((action) => action.id)).toEqual(["wizard-back", "wizard-preview", "wizard-submit"]);
    expect(SCREEN_VISIBLE_ACTIONS["evidence-review"].map((action) => action.effect.type)).toEqual(["modal", "modal"]);
    expect(SCREEN_VISIBLE_ACTIONS["evidence-review"][0]!.effect).toMatchObject({ confirmCommand: { owner: "evidence.review", requiresRevision: true } });
    expect(SCREEN_VISIBLE_ACTIONS["executive-final-reports"][0]!.effect).toMatchObject({ type: "modal", dialog: "issue-final-report", confirmCommand: { owner: "reports.decide" } });
  });

  it("keeps canonical creation identity separate from the full-screen fixture and filters withheld Auditee calendar records", async () => {
    const canonical = MemoryMockStore.createCanonical({ clock: () => FIXED_NOW });
    const inspector = createMockBackend({ store: canonical, principal: DEMO_PRINCIPALS.inspector });
    await expect(inspector.findings.get({ findingId: "FND-CAB-2026-001" })).rejects.toThrow(/not found/i);
    const auditee = createMockBackend({ store: canonical, principal: DEMO_PRINCIPALS.auditee });
    expect((await auditee.calendar.list({})).items.map((item) => item.auditId)).toEqual(["AUD-2026-001"]);
    await expect(auditee.calendar.openItem({ calendarItemId: "CAL-AUD-2026-099" })).rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
    await expect(auditee.calendar.openItem({ calendarItemId: "CAL-AUD-2026-001" })).resolves.toMatchObject({ auditId: "AUD-2026-001" });
    expect(await inspector.assistantDrafts.getGuidance({})).toMatchObject({ advisoryOnly: true });
  });

  it("keeps canonical references absent until creation and full-screen fixture references identity-consistent", () => {
    const canonical = MemoryMockStore.createCanonical({ clock: () => FIXED_NOW });
    expect(canonical.read((state) => state.findings["FND-CAB-2026-001"])).toBeUndefined();
    expect(canonical.read((state) => Object.values(state.reportVersions).flatMap((report) => report.findingIds))).not.toContain("FND-CAB-2026-001");
    const fixture = MemoryMockStore.createFullScreenScenario({ clock: () => FIXED_NOW });
    fixture.read((state) => {
      const finding = state.findings["FND-CAB-2026-001"]!;
      expect(finding).toMatchObject({ auditId: "AUD-2026-001", organizationId: "ORG-FLY-NAMIBIA" });
      for (const cap of state.capRevisions.filter((item) => item.findingId === finding.id)) expect(cap.organizationId).toBe(finding.organizationId);
      for (const evidence of state.evidenceVersions.filter((item) => item.findingId === finding.id)) expect(evidence.organizationId).toBe(finding.organizationId);
      for (const report of Object.values(state.reportVersions).filter((item) => item.findingIds.includes(finding.id))) expect(report).toMatchObject({ auditId: finding.auditId, organizationId: finding.organizationId });
    });
  });
  it("loads all 86 role-safe projections from one seed and exposes every declared visible command", async () => {
    const store = MemoryMockStore.createFullScreenScenario({ clock: () => FIXED_NOW });
    const sessions = Object.fromEntries(
      Object.entries(DEMO_PRINCIPALS).map(([role, principal]) => [
        role,
        createMockBackend({ store, principal }),
      ]),
    );

    const projections = await Promise.all(
      REACT_ROUTE_CONTRACTS.map((route) =>
        sessions[(route.requiredRole ?? "inspector") as keyof typeof sessions].administration.getScreenProjection({
          screenId: route.id,
        }),
      ),
    );

    expect(projections).toHaveLength(86);
    expect(projections.map((projection) => projection.screenId)).toEqual(
      REACT_ROUTE_CONTRACTS.map((route) => route.id),
    );
    expect(projections.every((projection) => projection.visibleActions.length > 0)).toBe(true);
    const invoked = await Promise.all(
      REACT_ROUTE_CONTRACTS.map(async (route) => {
        const backend = sessions[(route.requiredRole ?? "inspector") as keyof typeof sessions];
        const projection = projections.find((candidate) => candidate.screenId === route.id)!;
        return Promise.all(projection.visibleActions.map((action) =>
          backend.administration.invokeVisibleAction({ screenId: route.id, actionId: action.id }),
        ));
      }),
    );
    expect(invoked.flat().every((result) => result.effect.type !== undefined)).toBe(true);
    expect(projections.find((projection) => projection.screenId === "finding-detail")).toMatchObject({
      directRecordId: "FND-CAB-2026-001",
      state: "returned",
      overdue: true,
      versionHistory: true,
    });
    const auditeeDocuments = projections.find((projection) => projection.screenId === "auditee-documents");
    expect(auditeeDocuments).toMatchObject({ organizationId: "ORG-FLY-NAMIBIA" });
    expect(auditeeDocuments).not.toHaveProperty("internalCaaFields");
  });

  it("uses revision/idempotency guards while preserving empty, denied, returned, overdue, and version history states", async () => {
    const store = MemoryMockStore.createCanonical({ clock: () => FIXED_NOW });
    const inspector = createMockBackend({ store, principal: DEMO_PRINCIPALS.inspector });
    const auditee = createMockBackend({ store, principal: DEMO_PRINCIPALS.auditee });
    const manager = createMockBackend({ store, principal: DEMO_PRINCIPALS.manager });

    await expect(auditee.administration.getScreenProjection({ screenId: "manager-home" })).rejects.toBeInstanceOf(
      BackendAuthorizationInvariantError,
    );
    expect((await inspector.calendar.list({})).items.map((item) => item.auditId)).toEqual(["AUD-2026-001"]);
    await expect(inspector.calendar.openItem({ calendarItemId: "CAL-AUD-2026-099" })).rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
    expect((await inspector.communications.list({})).items).toEqual([]);
    expect((await manager.risk.getOverview({})).overdueFindingCount).toBe(1);
    expect(await manager.risk.getOverview({ organizationId: "ORG-FLY-NAMIBIA" })).toMatchObject({
      advisoryHealth: {
        band: "Needs Attention",
        basis: "CONFIGURED_DEMO_SCENARIO",
        score: 74,
      },
    });
    expect((await manager.teams.list({ role: "inspector" })).items.map((member) => member.subjectId)).toContain(
      "USR-INSPECTOR-AMINA",
    );
    expect((await auditee.documents.list({})).items.every((document) => document.organizationId === "ORG-FLY-NAMIBIA")).toBe(true);
    await expect(auditee.risk.getOverview({ organizationId: "ORG-SKYCARGO" })).rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
    await expect(auditee.teams.list({})).rejects.toBeInstanceOf(BackendAuthorizationInvariantError);

    const sent = await auditee.communications.send({
      expectedRevision: null,
      idempotencyKey: "auditee-message-v1",
      organizationId: "ORG-FLY-NAMIBIA",
      subject: "CAP evidence question",
      body: "Please confirm the configured expected evidence.",
      audience: "CAA",
    });
    expect(sent.id).toBe("auditee-message-v1");
    expect((await auditee.communications.list({})).items).toEqual([
      expect.objectContaining({ id: "auditee-message-v1", direction: "AUDITEE_TO_CAA" }),
    ]);
    expect((await inspector.communications.list({ organizationId: "ORG-FLY-NAMIBIA" })).items).toHaveLength(1);
    expect((await inspector.administration.listScreenProjections({})).every((screen) =>
      screen.screenId === "role-select" || screen.screenId.startsWith("inspector-") || ["audit-detail", "checklist-runner", "finding-detail", "closure-report-preview"].includes(screen.screenId),
    )).toBe(true);

    const profile = await inspector.profiles.getMine({});
    const first = await inspector.profiles.updateMine({
      expectedRevision: profile.revision,
      idempotencyKey: "profile-name-v1",
      displayName: "Amina Inspector",
    });
    const replay = await inspector.profiles.updateMine({
      expectedRevision: profile.revision,
      idempotencyKey: "profile-name-v1",
      displayName: "Amina Inspector",
    });
    expect(replay).toEqual(first);
    await expect(
      inspector.profiles.updateMine({
        expectedRevision: profile.revision,
        idempotencyKey: "profile-name-conflict",
        displayName: "Different name",
      }),
    ).rejects.toThrow(/revision conflict/i);

    const notification = await inspector.notifications.list({});
    const read = await inspector.notifications.markRead({
      notificationId: notification.items[0]!.id,
      expectedRevision: notification.items[0]!.revision,
      idempotencyKey: "notification-read-v1",
    });
    expect(read.readAt).toBe(FIXED_NOW);

    const draft = await inspector.assistantDrafts.createDraft({
      findingId: "FND-SKYCARGO-2026-099",
      prompt: "Summarize the configured finding basis.",
    });
    expect(draft.advisoryOnly).toBe(true);
    expect(draft.canCreateFinding).toBe(false);
  });

  it("enforces the explicit capability permission matrix and omits sensitive CAA fields from every Auditee-readable projection", async () => {
    const store = MemoryMockStore.createCanonical({ clock: () => FIXED_NOW });
    const auditee = createMockBackend({ store, principal: DEMO_PRINCIPALS.auditee });
    expect(DEMO_CAPABILITY_PERMISSION_MATRIX.auditee).not.toContain("risk");
    expect(DEMO_CAPABILITY_PERMISSION_MATRIX.auditee).not.toContain("teams");
    expect(DEMO_CAPABILITY_PERMISSION_MATRIX.auditee).not.toContain("assistantDrafts");

    await expect(auditee.risk.getOverview({})).rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
    await expect(auditee.teams.list({})).rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
    await expect(auditee.assistantDrafts.createDraft({ findingId: "FND-SKYCARGO-2026-099", prompt: "draft" })).rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
    await expect(auditee.administration.getScreenProjection({ screenId: "manager-home" })).rejects.toBeInstanceOf(BackendAuthorizationInvariantError);

    const auditeeOutputs = {
      communications: await auditee.communications.list({}),
      calendar: await auditee.calendar.list({}),
      profile: await auditee.profiles.getMine({}),
      coordination: await auditee.auditeeCoordination.list({}),
      reports: await auditee.auditeeReports.listReleased({}),
      documents: await auditee.documents.list({}),
      notifications: await auditee.notifications.list({}),
      screens: await auditee.administration.listScreenProjections({}),
    };
    const serialized = JSON.stringify(auditeeOutputs);
    expect(serialized).not.toMatch(/USR-INSPECTOR|SkyCargo|Internal CAA|risk/i);
    expect(auditeeOutputs.calendar.items.every((item) => item.organizationId === "ORG-FLY-NAMIBIA")).toBe(true);
    expect(auditeeOutputs.coordination.items.every((item) => item.organizationId === "ORG-FLY-NAMIBIA")).toBe(true);
    expect(auditeeOutputs.reports.items.every((item) => item.organizationId === "ORG-FLY-NAMIBIA")).toBe(true);
    expect(auditeeOutputs.documents.items.every((item) => item.organizationId === "ORG-FLY-NAMIBIA")).toBe(true);
    expect(auditeeOutputs.notifications.items.every((item) => item.subjectId === "USR-AUDITEE-FLY")).toBe(true);
    expect(auditeeOutputs.screens.every((screen) => screen.screenId === "role-select" || screen.screenId.startsWith("auditee-"))).toBe(true);
  });

  it("applies every role/capability entry in the explicit permission matrix", async () => {
    const store = MemoryMockStore.createCanonical({ clock: () => FIXED_NOW });
    const capabilityNames = Object.keys(DEMO_CAPABILITY_PERMISSION_MATRIX) as Array<keyof typeof DEMO_PRINCIPALS>;
    for (const role of capabilityNames) {
      const backend = createMockBackend({ store, principal: DEMO_PRINCIPALS[role] });
      const queries = {
        communications: () => backend.communications.list({}),
        calendar: () => backend.calendar.list({}),
        profiles: () => backend.profiles.getMine({}),
        teams: () => backend.teams.list({}),
        risk: () => backend.risk.getOverview({}),
        documents: () => backend.documents.list({}),
        notifications: () => backend.notifications.list({}),
        administration: () => backend.administration.listScreenProjections({}),
        assistantDrafts: () => backend.assistantDrafts.createDraft({ findingId: "FND-SKYCARGO-2026-099", prompt: "Draft." }),
        auditeeCoordination: () => backend.auditeeCoordination.list({}),
        auditeeReports: () => backend.auditeeReports.listReleased({}),
        planningIntake: () => backend.planningIntake.getDraft({ draftId: "PLAN-DRAFT-2026-001" }),
        packageDrafts: () => backend.packageDrafts.get({ packageDraftId: "PKG-AUD-2026-001-CABIN" }),
      } as const;
      for (const [capability, query] of Object.entries(queries) as Array<[keyof typeof queries, () => Promise<unknown>]>) {
        if (DEMO_CAPABILITY_PERMISSION_MATRIX[role].includes(capability)) {
          await expect(query()).resolves.toBeDefined();
        } else {
          await expect(query()).rejects.toBeInstanceOf(BackendAuthorizationInvariantError);
        }
      }
    }
  });

  it("returns immutable values from every composed capability, including nested action and version collections", async () => {
    const store = MemoryMockStore.createCanonical({ clock: () => FIXED_NOW });
    const inspector = createMockBackend({ store, principal: DEMO_PRINCIPALS.inspector });
    const manager = createMockBackend({ store, principal: DEMO_PRINCIPALS.manager });
    await inspector.communications.send({ expectedRevision: null, idempotencyKey: "immutable-message", organizationId: null, subject: "Subject", body: "Body", audience: "CAA" });
    const communication = await inspector.communications.list({});
    const calendar = await inspector.calendar.list({});
    const profile = await inspector.profiles.getMine({});
    const teams = await manager.teams.list({});
    const risk = await manager.risk.getOverview({});
    const documents = await inspector.documents.list({});
    const notifications = await inspector.notifications.list({});
    const screen = await inspector.administration.getScreenProjection({ screenId: "finding-detail" });
    const draft = await inspector.assistantDrafts.createDraft({ findingId: "FND-SKYCARGO-2026-099", prompt: "Summarize." });

    communication.items[0]!.body = "tampered";
    calendar.items[0]!.title = "tampered";
    profile.displayName = "tampered";
    teams.items[0]!.displayName = "tampered";
    risk.overdueFindingCount = 0;
    documents.items[0]!.title = "tampered";
    notifications.items[0]!.title = "tampered";
    [...screen.visibleActions][0]!.label = "tampered";
    draft.draft = "tampered";

    expect((await inspector.communications.list({})).items[0]!.body).toBe("Body");
    expect((await inspector.calendar.list({})).items[0]!.title).not.toBe("tampered");
    expect((await inspector.profiles.getMine({})).displayName).toBe("Amina Inspector");
    expect((await manager.teams.list({})).items[0]!.displayName).not.toBe("tampered");
    expect((await manager.risk.getOverview({})).overdueFindingCount).toBe(1);
    expect((await inspector.documents.list({})).items[0]!.title).not.toBe("tampered");
    expect((await inspector.notifications.list({})).items[0]!.title).not.toBe("tampered");
    expect((await inspector.administration.getScreenProjection({ screenId: "finding-detail" })).visibleActions[0]!.label).not.toBe("tampered");
    expect((await inspector.assistantDrafts.createDraft({ findingId: "FND-SKYCARGO-2026-099", prompt: "Summarize." })).draft).not.toBe("tampered");
    expect((await inspector.caps.listRevisions({ findingId: "FND-SKYCARGO-2026-099" })).items).toHaveLength(2);
    expect((await inspector.evidence.listVersions({ findingId: "FND-SKYCARGO-2026-099" })).map((version) => version.version)).toEqual([1, 2]);
  });
});
