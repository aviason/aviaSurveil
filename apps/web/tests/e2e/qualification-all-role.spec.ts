import { expect, test, type Browser, type BrowserContext, type Page } from "@playwright/test";
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import fs from "node:fs";
import path from "node:path";

import { planningItemLabel, recordReference } from "../../src/features/shared/record-presentation";

type Role = "admin" | "manager" | "finance" | "gm" | "executiveDirector" | "leadInspector" | "inspector" | "auditee";
type PurposeToken = "PLATFORM-ADMIN" | "AGA-MANAGER" | "FINANCE-REVIEWER" | "GENERAL-MANAGER" | "EXECUTIVE-DIRECTOR" | "LEAD-INSPECTOR" | "INSPECTOR" | "TARGET-AUDITEE" | "CONTROL-AUDITEE";

interface RosterAccount {
  purposeToken: PurposeToken;
  displayName: string;
  email: string;
  organizationId: string;
  role: Role;
  membershipId: string;
}

interface ScenarioManifest {
  schemaVersion: number;
  target: string;
  scenarioName: string;
  catalogVersion: string;
  catalogRootDigest: string;
  providerScopeId: string;
  regulatedTargetId: string;
  applicationType: string;
  inspectionCategory: string;
  riskCategory: string;
  plannedDate: string;
  location: string;
  selectedQuestionVersionIds: string[];
  selectionDigest: string;
}

interface TeamMemberRecord {
  subjectId: string;
  displayName: string;
  role: Role;
  organizationId: string | null;
  revision: number;
}

interface TeamMemberPage {
  items: TeamMemberRecord[];
  nextCursor?: string | null;
}

interface ReportVersionApiView {
  reportVersionId: string;
  reportId: string;
  kind: "PRELIMINARY" | "FINAL";
  auditId: string;
  findingIds: string[];
  potentialFindingIds: string[];
  potentialFindingRootDigest: string;
  contentHash: string;
  version: number;
  status: string;
}

interface CanonicalQuestionCatalogEntry {
  catalogVersion: string;
  usageClass: string;
  questionVersionId: string;
  formCode: string;
  ordinal: number;
  questionDigest: string;
  sourceLocator?: string;
  sourceGapState: string;
  canSelect: boolean;
  canPublish: boolean;
  aiAdvisory?: { advisoryState: string; recommendationReasonCodes?: string[] };
  recommendation?: {
    recommendationState: string;
    classification: string;
    includedByDefault: boolean;
    canDefer: boolean;
    signalCodes?: string[];
  };
}

interface CanonicalQuestionCatalogPage {
  items: CanonicalQuestionCatalogEntry[];
  nextCursor?: string | null;
  catalogVersion: string;
  usageClass: string;
  totalCount: number;
  recommendationSummary: {
    comparableAuditCount: number;
    historyDeferredCount: number;
  };
}

interface ApprovedSourceQuestion {
  immutableQuestionId: string;
  immutableQuestionVersionId: string;
  ordinal: number;
  textDigest: string;
  sourceLocator: string;
}

interface ApprovedSourceForm {
  formCode: string;
  sourceFormSha256: string;
  sourceArchiveSha256: string;
  questionCount: number;
  questions: ApprovedSourceQuestion[];
}

interface ApprovedCatalogOracle {
  catalogRootDigest: string;
  forms: ApprovedSourceForm[];
  rows: Array<ApprovedSourceQuestion & { formCode: string }>;
}

interface EvidenceCompleteApiView {
  evidenceVersionId: string;
  version: number;
  uploadState: string;
  scanState: string;
  reviewState: string;
}

interface ApiResponse {
  status: number;
  rawBody: string;
  body: unknown;
}

const origin = process.env.AVIA_QUALIFICATION_ORIGIN ?? "https://localhost:8443";
const scenarioId = process.env.AVIA_QUALIFICATION_SCENARIO_ID ?? "qualification-local";
const scenarioManifestPath = process.env.AVIA_QUALIFICATION_SCENARIO_MANIFEST ?? "";
const rosterManifestPath = process.env.AVIA_QUALIFICATION_ROSTER_MANIFEST ?? "";
const credentialDirectory = process.env.AVIA_QUALIFICATION_CREDENTIAL_DIRECTORY ?? "";
const cloudCredentialRecovery = process.env.AVIA_QUALIFICATION_CLOUD_CREDENTIAL_RECOVERY === "1";
const workspaceRoot = process.env.AVIA_WORKSPACE_ROOT ?? "";
const resultPath = process.env.AVIA_QUALIFICATION_RESULT_PATH ?? "";
const qualificationTarget = process.env.AVIA_QUALIFICATION_TARGET ?? "";
type RoleSession = { context: BrowserContext; page: Page };
const roleSessions = new Map<PurposeToken, RoleSession>();

function readJson<T>(filePath: string): T {
  if (!filePath) throw new Error("Qualification manifest path is not configured.");
  return JSON.parse(fs.readFileSync(filePath, "utf8")) as T;
}

function readApprovedCatalogOracle(): ApprovedCatalogOracle {
  if (!workspaceRoot) throw new Error("Workspace root is required for the approved catalog oracle.");
  const archivePath = path.join(workspaceRoot, "apps", "surveil", "deliverables", "AGA_ALL_FORMS_APPROVED_SOURCE_V2.zip");
  const result = spawnSync(
    "unzip",
    ["-p", archivePath, "aga-approved-source-v2/AGA_ALL_FORMS_APPROVED_SOURCE_V2.json"],
    { cwd: workspaceRoot, env: process.env, encoding: "utf8", maxBuffer: 8 * 1024 * 1024 },
  );
  if (result.error || result.status !== 0 || !result.stdout) {
    throw new Error(`Approved catalog oracle extraction failed (exit ${result.status ?? "spawn"}).`);
  }
  const document = JSON.parse(result.stdout) as { catalogRootDigest: string; forms: ApprovedSourceForm[] };
  const rows: ApprovedCatalogOracle["rows"] = [];
  const digest = createHash("sha256");
  for (const form of document.forms) {
    digest.update(`form\u0000${form.formCode}\u0000${form.sourceFormSha256}\u0000${form.sourceArchiveSha256}\u0000${form.questionCount}\n`);
    for (const question of form.questions) {
      const row = { ...question, formCode: form.formCode };
      rows.push(row);
      digest.update(`question\u0000${form.formCode}\u0000${question.ordinal}\u0000${question.immutableQuestionId}\u0000${question.immutableQuestionVersionId}\u0000${question.textDigest}\u0000${question.sourceLocator}\n`);
    }
  }
  return { catalogRootDigest: `sha256:${digest.digest("hex")}`, forms: document.forms, rows };
}

function catalogRootDigestForTraversal(oracle: ApprovedCatalogOracle, orderedQuestionVersionIds: readonly string[]): string {
  const observed = new Set(orderedQuestionVersionIds);
  const digest = createHash("sha256");
  for (const form of oracle.forms) {
    digest.update(`form\u0000${form.formCode}\u0000${form.sourceFormSha256}\u0000${form.sourceArchiveSha256}\u0000${form.questionCount}\n`);
    for (const question of form.questions) {
      if (!observed.has(question.immutableQuestionVersionId)) continue;
      digest.update(`question\u0000${form.formCode}\u0000${question.ordinal}\u0000${question.immutableQuestionId}\u0000${question.immutableQuestionVersionId}\u0000${question.textDigest}\u0000${question.sourceLocator}\n`);
    }
  }
  return `sha256:${digest.digest("hex")}`;
}

function recoverCloudCredential(purposeToken: PurposeToken): string {
  if (!workspaceRoot || !qualificationTarget) throw new Error("Cloud qualification credential recovery is not configured.");
  const [customer, environment] = qualificationTarget.split("/", 2);
  if (!customer || environment !== "demo") throw new Error("Cloud qualification credential recovery requires the exact demo target.");
  const expectedPath = path.join(workspaceRoot, ".state", "qualification", "role-credentials", customer, environment, scenarioId, purposeToken);
  const python = process.env.AVIA_QUALIFICATION_PYTHON ?? "python3";
  const result = spawnSync(
    python,
    [
      path.join(workspaceRoot, "scripts", "workspace.py"),
      "qualification-credential",
      "--target",
      qualificationTarget,
      "--scenario-id",
      scenarioId,
      "--purpose",
      purposeToken,
      "--confirm",
      `${qualificationTarget}:${purposeToken}`,
    ],
    { cwd: workspaceRoot, env: process.env, encoding: "utf8", maxBuffer: 64 * 1024 },
  );
  if (result.error || result.status !== 0) {
    throw new Error(`Cloud credential recovery failed for ${purposeToken} (exit ${result.status ?? "spawn"}).`);
  }
  const returnedPath = result.stdout.trim().split(/\r?\n/).at(-1) ?? "";
  if (path.resolve(returnedPath) !== path.resolve(expectedPath)) {
    throw new Error(`Cloud credential recovery returned an unexpected custody path for ${purposeToken}.`);
  }
  return expectedPath;
}

function readCredential(purposeToken: PurposeToken): string {
  const filePath = cloudCredentialRecovery ? recoverCloudCredential(purposeToken) : path.join(credentialDirectory, purposeToken);
  if (!cloudCredentialRecovery && !credentialDirectory) throw new Error("Qualification credential directory is not configured.");
  const stat = fs.lstatSync(filePath);
  if (!stat.isFile() || stat.isSymbolicLink() || (stat.mode & 0o077) !== 0) throw new Error(`Qualification credential entry is not a private regular file: ${purposeToken}`);
  try {
    const value = fs.readFileSync(filePath, "utf8").trim();
    if (!value) throw new Error(`Qualification credential entry is empty: ${purposeToken}`);
    return value;
  } finally {
    if (cloudCredentialRecovery) {
      fs.unlinkSync(filePath);
      const runDirectory = path.dirname(filePath);
      if (fs.readdirSync(runDirectory).length === 0) fs.rmdirSync(runDirectory);
    }
  }
}

function potentialFindingRootDigest(ids: readonly string[]): string {
  const canonicalIds = [...ids].sort();
  return `sha256:${createHash("sha256").update(JSON.stringify(canonicalIds)).digest("hex")}`;
}

function writeEvent(event: Record<string, string | number | boolean>): void {
  if (!resultPath) return;
  fs.mkdirSync(path.dirname(resultPath), { recursive: true, mode: 0o700 });
  fs.appendFileSync(resultPath, `${JSON.stringify({ scenarioId, ...event })}\n`, { mode: 0o600 });
  fs.chmodSync(resultPath, 0o600);
}

function accountsByToken(): Map<PurposeToken, RosterAccount> {
  const accounts = readJson<{ accounts: RosterAccount[] }>(rosterManifestPath).accounts;
  if (accounts.length !== 9) throw new Error(`Expected nine prepared accounts, received ${accounts.length}.`);
  return new Map(accounts.map((account) => [account.purposeToken, account]));
}

async function signIn(browser: Browser, account: RosterAccount, viewport: { width: number; height: number } = { width: 1280, height: 800 }): Promise<{ context: BrowserContext; page: Page }> {
  const existing = roleSessions.get(account.purposeToken);
  if (existing) return existing;
  const context = await browser.newContext({ viewport });
  const page = await context.newPage();
  await page.goto(`${origin}/`, { waitUntil: "domcontentloaded" });
  await page.getByRole("button", { name: "Sign in with organization identity" }).click();
  await expect(page).toHaveURL(/\/identity\/login\?id=/);
  await page.locator('input[name="identifier"]').fill(account.email);
  await page.locator('input[name="password"]').fill(readCredential(account.purposeToken));
  await page.getByRole("button", { name: "Continue" }).click();
  await page.waitForURL((url) => url.origin === origin && !url.pathname.startsWith("/identity/"), { timeout: 30_000 });
  await expect(page.locator("main")).toHaveCount(1);
  await expect(page.getByRole("button", { name: "Logout" })).toBeVisible({ timeout: 30_000 });
  const session = { context, page };
  roleSessions.set(account.purposeToken, session);
  return session;
}

async function closeRoleSessions(): Promise<void> {
  const sessions = [...roleSessions.values()];
  roleSessions.clear();
  await Promise.all(sessions.map(({ context }) => context.close()));
}

async function verifyGenericFailure(browser: Browser): Promise<void> {
  const context = await browser.newContext({ viewport: { width: 1280, height: 800 } });
  const page = await context.newPage();
  await page.goto(`${origin}/`, { waitUntil: "domcontentloaded" });
  await page.getByRole("button", { name: "Sign in with organization identity" }).click();
  await expect(page).toHaveURL(/\/identity\/login\?id=/);
  await page.locator('input[name="identifier"]').fill("nonexistent-qualification-user@example.invalid");
  await page.locator('input[name="password"]').fill("fake-password-that-must-not-match");
  await page.getByRole("button", { name: "Continue" }).click();
  await expect(page.locator("body")).toContainText("username, email, or password is incorrect");
  writeEvent({ event: "generic-login-failure", status: "verified locally" });
  await context.close();
}

async function getApiResponse(page: Page, requestPath: string): Promise<ApiResponse> {
  return page.evaluate(async (path) => {
    const result = await fetch(`/api${path}`, { credentials: "include" });
    const rawBody = await result.text();
    try {
      return { status: result.status, rawBody, body: JSON.parse(rawBody) as unknown };
    } catch {
      return { status: result.status, rawBody, body: null };
    }
  }, requestPath);
}

async function getApiJson<T>(page: Page, requestPath: string): Promise<T> {
  const response = await getApiResponse(page, requestPath);
  if (response.body === null) {
    throw new Error(`${requestPath} returned a non-JSON response (HTTP ${response.status}, ${response.rawBody.length} bytes).`);
  }
  expect(response.status, `${requestPath} must be available to the authenticated session`).toBe(200);
  return response.body as T;
}

async function assertControlApiIsolation(
  page: Page,
  requestPaths: string[],
  forbiddenValues: string[],
  options: { detailPaths?: readonly string[] } = {},
): Promise<void> {
  const detailPaths = new Set(options.detailPaths ?? []);
  for (const requestPath of requestPaths) {
    const response = await getApiResponse(page, requestPath);
    if (detailPaths.has(requestPath)) {
      expect([403, 404], `${requestPath} must be denied for Control Auditee`).toContain(response.status);
    } else {
      expect([200, 403, 404], `${requestPath} must be empty or denied for Control Auditee`).toContain(response.status);
      if (response.status === 200 && response.body && typeof response.body === "object") {
        const collections: unknown[] = [];
        const collect = (value: unknown) => {
          if (Array.isArray(value)) {
            collections.push(value);
            return;
          }
          if (!value || typeof value !== "object") return;
          for (const child of Object.values(value)) collect(child);
        };
        collect(response.body);
        for (const collection of collections) {
          expect(collection, `${requestPath} returned records to Control Auditee`).toHaveLength(0);
        }
      }
    }
    for (const forbiddenValue of forbiddenValues) {
      expect(response.rawBody, `${requestPath} leaked ${forbiddenValue} to Control Auditee`).not.toContain(forbiddenValue);
    }
  }
}

async function assertControlDomIsolation(page: Page, routePath: string, forbiddenValues: string[]): Promise<void> {
  await page.goto(`${origin}${routePath}`, { waitUntil: "domcontentloaded" });
  const emptyStatePattern = /not available|not found|no [a-z ]+(available|found|assigned|records|findings|audits|reports|evidence|items|match)/i;
  await expect.poll(async () => {
    const text = await page.locator("body").innerText();
    return !text.includes("Loading workspace") && !text.includes("Restoring the secured route") && emptyStatePattern.test(text);
  }, { timeout: 30_000 }).toBe(true);
  const bodyText = await page.locator("body").innerText();
  for (const forbiddenValue of forbiddenValues) {
    expect(bodyText, `${routePath} leaked ${forbiddenValue} to Control Auditee DOM`).not.toContain(forbiddenValue);
  }
  const recordSelectors = ["[data-audit-id]", "[data-finding-id]", "[data-evidence-id]", "[data-report-version-id]"];
  for (const selector of recordSelectors) {
    await expect(page.locator(selector), `${routePath} rendered a target record in Control Auditee DOM`).toHaveCount(0);
  }
  expect(bodyText, `${routePath} must render an explicit empty or denied state`).toMatch(emptyStatePattern);
}

async function expectResponsiveViewport(page: Page, surface: string): Promise<void> {
  const dimensions = await page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    documentWidth: document.documentElement.scrollWidth,
    bodyWidth: document.body.scrollWidth,
  }));
  expect(dimensions.documentWidth, `${surface} document must not overflow horizontally`).toBeLessThanOrEqual(dimensions.clientWidth + 1);
  expect(dimensions.bodyWidth, `${surface} body must not overflow horizontally`).toBeLessThanOrEqual(dimensions.clientWidth + 1);
}

async function findPreparedTeamMember(page: Page, role: "leadInspector" | "inspector", displayName: string): Promise<TeamMemberRecord> {
  const output = await getApiJson<TeamMemberPage>(page, `/v1/team-members?role=${encodeURIComponent(role)}`);
  const matches = output.items.filter((member) => member.displayName === displayName && member.role === role);
  if (matches.length !== 1) {
    const available = output.items.map((member) => `${member.displayName} [${member.role}]`).join(" | ");
    throw new Error(`server roster must contain exactly one ${role} named ${displayName}; available entries: ${available}`);
  }
  return matches[0];
}

interface PlanningPinView {
  id: string;
  title?: string;
  revision: number;
  organizationId?: string;
  inspectionType?: string;
  submittedScopeSnapshotId?: string;
  planningSnapshotDigest?: string;
}

async function assertPlanningPin(page: Page, planningItemId: string, expected: { snapshotId: string; digest: string; minimumRevision: number }): Promise<PlanningPinView> {
  const output = await getApiJson<{ items: PlanningPinView[] }>(page, "/v1/planning/items?limit=100");
  const item = output.items.find((candidate) => candidate.id === planningItemId);
  expect(item, `planning item ${planningItemId} must remain visible to its approval authority`).toBeTruthy();
  expect(item?.submittedScopeSnapshotId).toBe(expected.snapshotId);
  expect(item?.planningSnapshotDigest).toBe(expected.digest);
  expect(item?.revision).toBeGreaterThanOrEqual(expected.minimumRevision);
  return item as PlanningPinView;
}

async function verifyPublicPrivateAdmin404(browser: Browser): Promise<void> {
  const context = await browser.newContext({ viewport: { width: 1280, height: 800 } });
  try {
    const response = await context.request.get(`${origin}/api/private-admin`);
    expect(response.status(), "public private-admin endpoint must not exist").toBe(404);
  } finally {
    await context.close();
  }
}

async function verifyRoleBoundaryMatrix(browser: Browser, cases: Array<{ account: RosterAccount; home: string; foreign: string }>): Promise<void> {
  for (const boundary of cases) {
    const session = await signIn(browser, boundary.account);
    await session.page.goto(`${origin}${boundary.home}`, { waitUntil: "domcontentloaded" });
    await expect(session.page.getByTestId("route-forbidden")).toHaveCount(0);
    await session.page.goto(`${origin}${boundary.foreign}`, { waitUntil: "domcontentloaded" });
    await expect(session.page.getByTestId("route-forbidden")).toBeVisible();
    await session.page.goto(`${origin}${boundary.home}`, { waitUntil: "domcontentloaded" });
    const directLogout = session.page.getByRole("button", { name: "Logout" }).first();
    const profile = session.page.getByRole("button", { name: boundary.account.displayName }).first();
    await expect.poll(async () => (await directLogout.isVisible()) || (await profile.isVisible()), { timeout: 30_000 }).toBe(true);
    if (await directLogout.isVisible()) {
      await directLogout.click();
    } else {
      await profile.click();
      await session.page.getByRole("menu", { name: "Profile menu" }).getByRole("button", { name: "Logout" }).click();
    }
    await expect(session.page.getByRole("button", { name: "Sign in with organization identity" })).toBeVisible();
    const sessionAfterLogout = await getApiResponse(session.page, "/auth/session");
    expect(sessionAfterLogout.status, `${boundary.account.purposeToken} session must be unauthorized after logout`).toBe(401);
  }
}

test.describe("prepared identity connected qualification", () => {
  test.afterEach(async () => {
    await closeRoleSessions();
  });

  test("completes the connected planning approval and canonical preparation handoff", async ({ browser }) => {
    test.setTimeout(600_000);
    const scenario = readJson<ScenarioManifest>(scenarioManifestPath);
    const catalogOracle = readApprovedCatalogOracle();
    const accounts = accountsByToken();
    expect(scenario.schemaVersion).toBe(1);
    if (!qualificationTarget) throw new Error("Qualification target is not configured.");
    expect(scenario.target).toBe(qualificationTarget);
    expect(scenario.scenarioName).toBe("all-role-e2e");
    expect(scenario.selectedQuestionVersionIds.length).toBeGreaterThan(0);
    expect(catalogOracle.catalogRootDigest).toBe(scenario.catalogRootDigest);
    expect(catalogOracle.rows).toHaveLength(1310);
    await verifyGenericFailure(browser);

    const admin = accounts.get("PLATFORM-ADMIN");
    const manager = accounts.get("AGA-MANAGER");
    const finance = accounts.get("FINANCE-REVIEWER");
    const gm = accounts.get("GENERAL-MANAGER");
    const executiveDirector = accounts.get("EXECUTIVE-DIRECTOR");
    const leadInspector = accounts.get("LEAD-INSPECTOR");
    const inspector = accounts.get("INSPECTOR");
    const targetAuditee = accounts.get("TARGET-AUDITEE");
    const controlAuditee = accounts.get("CONTROL-AUDITEE");
    if (!admin || !manager || !finance || !gm || !executiveDirector || !leadInspector || !inspector || !targetAuditee || !controlAuditee) {
      throw new Error("The approved governance and inspection accounts are incomplete.");
    }
    const adminSession = await signIn(browser, admin);
    const managerSession = await signIn(browser, manager);
    let planningItemId = "";
    let planningItemTitle = "";
    let assignmentId = "";
    let inspectionId = "";
    let packageId = "";
    let potentialFindingId = "";
    let preliminaryReportVersionId = "";
    let preliminaryPotentialRootDigest = "";
    let finalReportVersionId = "";
    let finalPotentialRootDigest = "";
    let findingId = "";
    let preliminaryContentHash = "";
    let submittedScopeSnapshotId = "";
    let planningSnapshotDigest = "";
    let planningRevision = 0;
    let supplierOrganizationId = "";
    try {
      await adminSession.page.goto(`${origin}/admin/users-roles`, { waitUntil: "domcontentloaded" });
      await expect(adminSession.page.getByTestId("admin-users-roles-page")).toBeVisible();
      const directory = adminSession.page.getByRole("list", { name: "Identity access directory" });
      for (const account of accounts.values()) {
        const card = directory.getByRole("listitem").filter({ hasText: account.email });
        await expect(card).toHaveCount(1);
        await expect(card).toContainText(account.role);
        await expect(card).toContainText(account.organizationId);
      }
      const adminDirectory = await getApiJson<{ items: Array<{
        email: string;
        roles: Role[];
        organizationId: string | null;
        membershipId: string | null;
        membershipRevision: number;
        membershipState: string;
        membershipDrift: string;
        requiredActions: string[];
        accountStatus: string;
      }> }>(adminSession.page, "/v1/admin/access-directory?limit=25");
      for (const account of accounts.values()) {
        const entry = adminDirectory.items.find((candidate) => candidate.email === account.email);
        expect(entry, `admin access directory must contain ${account.purposeToken}`).toBeTruthy();
        expect(entry?.roles).toEqual([account.role]);
        expect(entry?.organizationId).toBe(account.organizationId);
        expect(entry?.membershipId).toBe(account.membershipId);
        expect(entry?.membershipRevision).toBe(1);
        expect(entry?.membershipState).toBe("active");
        expect(entry?.membershipDrift).toBe("in-sync");
        expect(entry?.requiredActions).toEqual([]);
        expect(entry?.accountStatus).toBe("enabled");
      }
      writeEvent({ event: "admin-roster-verification", accountCount: accounts.size, status: "verified locally" });

      await managerSession.page.goto(`${origin}/department-manager/new-audit/step-1`, { waitUntil: "domcontentloaded" });
      await expect(managerSession.page.getByRole("heading", { name: "New Inspection" })).toBeVisible();
      const supplierSelector = managerSession.page.getByLabel("Supplier / organization");
      const supplierOptions = supplierSelector.locator("option");
      await expect(supplierOptions).toHaveCount(3);
      await expect(supplierSelector).toBeEnabled();
      const providerScopeSelector = managerSession.page.getByLabel("Provider scope");
      const supplierValues = await supplierOptions.evaluateAll((options) => options.map((option) => (option as HTMLOptionElement).value));
      let scenarioSupplierFound = false;
      for (const supplierValue of supplierValues) {
        await supplierSelector.selectOption(supplierValue);
        if (await providerScopeSelector.locator(`option[value="${scenario.providerScopeId}"]`).count()) {
          scenarioSupplierFound = true;
          break;
        }
      }
      expect(scenarioSupplierFound, "the server-authorized supplier cascade must expose the scenario provider scope").toBe(true);
      await expect(providerScopeSelector).toBeEnabled();
      await providerScopeSelector.selectOption(scenario.providerScopeId);
      const regulatedTargetSelector = managerSession.page.getByLabel("Regulated target");
      await expect(regulatedTargetSelector).toBeEnabled();
      await regulatedTargetSelector.selectOption(scenario.regulatedTargetId);
      const pendingApplicationType = managerSession.page.getByLabel("Inspection type");
      await expect(pendingApplicationType).toBeEnabled();
      await expect(pendingApplicationType).toHaveValue(scenario.applicationType);
      supplierOrganizationId = await supplierSelector.inputValue();
      await managerSession.page.getByRole("button", { name: "Continue", exact: true }).click();
      await expect(managerSession.page).toHaveURL(/\/department-manager\/new-audit\/step-2\?draftId=/);
      await expect(managerSession.page.getByRole("heading", { name: "Purpose", exact: true })).toBeVisible();
      const purpose = managerSession.page.getByLabel("Purpose", { exact: true });
      await purpose.fill("Prepared all-role AGA qualification through the approved source catalog.");
      await managerSession.page.getByRole("button", { name: "Continue", exact: true }).click();
      await expect(managerSession.page).toHaveURL(/\/department-manager\/new-audit\/step-3\?draftId=/);
      const plannedDate = managerSession.page.getByLabel("Planned date", { exact: true });
      const location = managerSession.page.getByLabel("Location", { exact: true });
      await plannedDate.fill(scenario.plannedDate);
      await location.fill(scenario.location);
      await managerSession.page.getByRole("button", { name: "Continue", exact: true }).click();
      await expect(managerSession.page).toHaveURL(/\/department-manager\/new-audit\/step-4\?draftId=/);
      expect(supplierOrganizationId).toBeTruthy();
      const draftId = new URL(managerSession.page.url()).searchParams.get("draftId") ?? "";
      expect(draftId).toBeTruthy();
      const draft = await getApiJson<{ organizationId?: string; organizationName?: string; applicationType?: string; providerScopeId?: string; regulatedTargetId?: string; scopeDraftId?: string; catalogVersion?: string }>(managerSession.page, `/v1/planning/intake-drafts/${encodeURIComponent(draftId)}`);
      expect(draft.organizationId).toBeTruthy();
      expect(draft.organizationName).toBeTruthy();
      expect(draft.applicationType).toBe(scenario.applicationType);
      expect(draft.providerScopeId).toBe(scenario.providerScopeId);
      expect(draft.regulatedTargetId).toBe(scenario.regulatedTargetId);
      expect(draft.organizationId).toBe(supplierOrganizationId);
      const scopeDraftId = draft.scopeDraftId ?? "";
      expect(scopeDraftId).toBeTruthy();
      expect(draft.catalogVersion).toBe(scenario.catalogVersion);
      const catalogRows = managerSession.page.locator(".planning-intake-catalog-list li");
      await expect(catalogRows).toHaveCount(25, { timeout: 30_000 });
      const advancedFilters = managerSession.page.locator(".planning-intake-advanced-filters");
      await expect(advancedFilters).not.toHaveAttribute("open", "");
      await advancedFilters.locator("> summary").click();
      await expect(managerSession.page.getByLabel("Recommendation filter")).toHaveValue("SUGGESTED_NOW");
      await expect(managerSession.page.locator(".planning-intake-catalog-pagination")).toContainText("matching", { timeout: 30_000 });
      await advancedFilters.locator("> summary").click();

      const searchQuestions = managerSession.page.getByRole("textbox", { name: "Search questions", exact: true });
      const firstPageVisibleText = await catalogRows.first().locator(".planning-intake-question-info").textContent() ?? "";
      const firstVisibleReferences = await catalogRows.locator(".planning-intake-question-info").evaluateAll((nodes) => nodes.map((node) => node.textContent?.split(" · ").slice(0, 2).join(" · ") ?? ""));
      const firstPageQuery = new URLSearchParams({ usageClass: "GOVERNED_OPERATIONAL", scopeId: scopeDraftId, applicationType: scenario.applicationType, includedByDefault: "true", limit: "25" });
      const firstPage = await getApiJson<CanonicalQuestionCatalogPage>(managerSession.page, `/v1/question-catalogs/${encodeURIComponent(scenario.catalogVersion)}/questions?${firstPageQuery.toString()}`);
      expect(firstPage.totalCount).toBeGreaterThan(0);
      expect(firstPage.items.map((item) => `${item.formCode} · item ${item.ordinal}`)).toEqual(firstVisibleReferences);
      const comparableAuditCount = firstPage.recommendationSummary.comparableAuditCount;
      expect(comparableAuditCount).toBeGreaterThanOrEqual(3);
      expect(firstPage.recommendationSummary.historyDeferredCount).toBeGreaterThan(0);
      await expect(managerSession.page.getByRole("status", { name: "Prior-Audit recommendation summary" })).toContainText(`${comparableAuditCount} comparable prior Audits`);
      await expect(managerSession.page.getByRole("status", { name: "Prior-Audit recommendation summary" })).toContainText("withheld by history");
      const fullRecommendationQuery = new URLSearchParams({ usageClass: "GOVERNED_OPERATIONAL", scopeId: scopeDraftId, applicationType: scenario.applicationType, limit: "2000", projection: "selection" });
      const fullRecommendationPage = await getApiJson<CanonicalQuestionCatalogPage>(managerSession.page, `/v1/question-catalogs/${encodeURIComponent(scenario.catalogVersion)}/questions?${fullRecommendationQuery.toString()}`);
      await expect(managerSession.page.locator(".planning-intake-catalog-pagination")).toContainText(`${firstPage.totalCount.toLocaleString("en-US")} matching`);
      expect(firstPage.totalCount).toBeLessThan(fullRecommendationPage.totalCount);
      expect(fullRecommendationPage.totalCount).toBe(1310);
      await advancedFilters.locator("> summary").click();
      await managerSession.page.getByLabel("Recommendation filter").selectOption("");
      await expect(managerSession.page.getByRole("heading", { name: "Full approved catalog", exact: true })).toBeVisible();
      await expect(managerSession.page.locator(".planning-intake-catalog-pagination")).toContainText(`${fullRecommendationPage.totalCount.toLocaleString("en-US")} matching`, { timeout: 30_000 });
      await expect(advancedFilters).toContainText("1 active");
      await managerSession.page.getByRole("button", { name: "Clear filters", exact: true }).click();
      await expect(managerSession.page.getByLabel("Recommendation filter")).toHaveValue("SUGGESTED_NOW");
      await expect(managerSession.page.getByRole("heading", { name: "Suggested questions", exact: true })).toBeVisible();
      await expect(managerSession.page.locator(".planning-intake-catalog-pagination")).toContainText(`${firstPage.totalCount.toLocaleString("en-US")} matching`, { timeout: 30_000 });
      await advancedFilters.locator("> summary").click();
      const seedHistoryQuestionId = "qv:aga-approved-source-v2:FSS-AGA-FORM-002:all-forms-preview-002-0001";
      const fullHistoryQuestion = fullRecommendationPage.items.find((item) => item.questionVersionId === seedHistoryQuestionId);
      expect(fullHistoryQuestion).toMatchObject({ questionVersionId: seedHistoryQuestionId, canSelect: true });
      const deferredRecommendationQuery = new URLSearchParams({ usageClass: "GOVERNED_OPERATIONAL", scopeId: scopeDraftId, applicationType: scenario.applicationType, recommendationState: "RECENTLY_VERIFIED", includedByDefault: "false", limit: "2000", projection: "selection" });
      const deferredRecommendationPage = await getApiJson<CanonicalQuestionCatalogPage>(managerSession.page, `/v1/question-catalogs/${encodeURIComponent(scenario.catalogVersion)}/questions?${deferredRecommendationQuery.toString()}`);
      expect(deferredRecommendationPage.totalCount).toBe(firstPage.recommendationSummary.historyDeferredCount);
      expect(deferredRecommendationPage.items.length).toBeGreaterThan(0);
      const restoredHistoryQuestionId = deferredRecommendationPage.items[0]?.questionVersionId ?? "";
      expect(restoredHistoryQuestionId).toBeTruthy();
      expect(deferredRecommendationPage.items.every((item) => item.recommendation !== undefined && item.recommendation.recommendationState === "RECENTLY_VERIFIED" && item.recommendation.classification === "DEFER_ELIGIBLE" && item.recommendation.canDefer && !item.recommendation.includedByDefault)).toBe(true);
      expect(firstPage.items.map((item) => item.questionVersionId)).not.toContain(restoredHistoryQuestionId);
      const restoreHistoryButton = managerSession.page.getByRole("button", { name: "Include all history-deferred questions" });
      await expect(restoreHistoryButton).toBeEnabled();
      await restoreHistoryButton.click();
      await expect(managerSession.page.getByText(/history-deferred questions are included and ready for selection review/)).toBeVisible();
      await managerSession.page.getByRole("button", { name: "Undo changes" }).click();
      writeEvent({ event: "manager-prior-audit-recommendation-oracle", suggestedCount: firstPage.totalCount, fullCatalogCount: fullRecommendationPage.totalCount, comparableAuditCount: firstPage.recommendationSummary.comparableAuditCount, historyDeferredCount: firstPage.recommendationSummary.historyDeferredCount, presentHistoryDeferredQuestionId: restoredHistoryQuestionId, absentFromSuggested: !firstPage.items.some((item) => item.questionVersionId === restoredHistoryQuestionId), restoredAndUndone: true, status: "verified locally" });
      const firstPageReference = firstPageVisibleText;
      const nextQuestions = managerSession.page.getByRole("button", { name: "Next questions" });
      await expect(nextQuestions).toBeEnabled();
      await nextQuestions.click();
      await expect(catalogRows.first().locator(".planning-intake-question-info")).not.toHaveText(firstPageReference, { timeout: 30_000 });
      await managerSession.page.getByRole("button", { name: "Previous questions" }).click();
      await expect(catalogRows.first().locator(".planning-intake-question-info")).toHaveText(firstPageReference, { timeout: 30_000 });

      // The complete catalog proof is an authenticated server cursor audit. The
      // browser still exercises the real paginated UI, but it does not spend ten
      // minutes rendering 1,310 rows one 25-row screen at a time.
      const seenIds: string[] = [];
      const seen = new Set<string>();
      let apiCursor: string | undefined;
      for (;;) {
        const apiQuery = new URLSearchParams({ usageClass: "GOVERNED_OPERATIONAL", scopeId: scopeDraftId, applicationType: scenario.applicationType, limit: "2000", projection: "selection" });
        if (apiCursor) apiQuery.set("cursor", apiCursor);
        const apiPage = await getApiJson<CanonicalQuestionCatalogPage>(managerSession.page, `/v1/question-catalogs/${encodeURIComponent(scenario.catalogVersion)}/questions?${apiQuery.toString()}`);
        expect(apiPage.items.length).toBeGreaterThan(0);
        for (const [pageIndex, item] of apiPage.items.entries()) {
          const oracleRow = catalogOracle.rows[seenIds.length + pageIndex];
          expect(oracleRow).toBeTruthy();
          expect(item.formCode).toBe(oracleRow?.formCode);
          expect(item.ordinal).toBe(oracleRow?.ordinal);
          expect(item.questionDigest).toBe(oracleRow?.textDigest);
          expect(item.sourceLocator).toBe(oracleRow?.sourceLocator);
          expect(item.sourceGapState).toBe("OPTIONAL_ENRICHMENT_NOT_PROVIDED");
          expect(item.canSelect).toBe(true);
          expect(item.canPublish).toBe(false);
        }
        for (const item of apiPage.items) {
          const id = item.questionVersionId;
          expect(id).toMatch(/^qv:aga-approved-source-v2:/);
          if (seen.has(id)) throw new Error(`Catalog traversal repeated a visible question row: ${id}`);
          seen.add(id);
          seenIds.push(id);
          if (scenario.selectedQuestionVersionIds.includes(id)) {
            // Selected rows are staged through the real server-backed Search
            // control below, so this audit never relies on a 53-page UI walk.
          }
        }
        apiCursor = apiPage.nextCursor ?? undefined;
        if (!apiCursor) break;
      }
      await expect(managerSession.page.locator(".planning-intake-catalog-pagination")).toContainText("matching");
      expect(seenIds).toHaveLength(1310);
      expect(new Set(seenIds).size).toBe(1310);
      expect(seenIds).toEqual(catalogOracle.rows.map((row) => row.immutableQuestionVersionId));
      expect(catalogRootDigestForTraversal(catalogOracle, seenIds)).toBe(catalogOracle.catalogRootDigest);

      let selectionSearchUnlocked = false;
      for (const selectedQuestionVersionId of scenario.selectedQuestionVersionIds) {
        const searchTerm = selectedQuestionVersionId.split(":").at(-1) ?? selectedQuestionVersionId;
        const selectedOracleRow = catalogOracle.rows.find((row) => row.immutableQuestionVersionId === selectedQuestionVersionId);
        expect(selectedOracleRow).toBeTruthy();
        await searchQuestions.fill(searchTerm);
        if (!selectionSearchUnlocked) {
          await advancedFilters.locator("> summary").click();
          await managerSession.page.getByLabel("Recommendation filter").selectOption("");
          await advancedFilters.locator("> summary").click();
          selectionSearchUnlocked = true;
        }
        const selectedRow = catalogRows.first();
        await expect(catalogRows).toHaveCount(1, { timeout: 30_000 });
        await expect(selectedRow).toContainText(`${selectedOracleRow?.formCode} · item ${selectedOracleRow?.ordinal}`);
        await selectedRow.getByRole("checkbox").check();
      }
      await managerSession.page.setViewportSize({ width: 390, height: 844 });
      try {
        await expect(managerSession.page.getByRole("button", { name: "View details" }).first()).toBeVisible();
        await managerSession.page.getByRole("button", { name: "View details" }).first().click();
        const questionDossier = managerSession.page.getByRole("dialog", { name: "Question dossier" });
        await expect(questionDossier).toBeVisible();
        await expect(questionDossier).toContainText("Question version");
        await expect(questionDossier).toContainText("Checklist focus");
        await questionDossier.getByRole("button", { name: "Close" }).click();
        await expect(questionDossier).toHaveCount(0);
        await expectResponsiveViewport(managerSession.page, "Manager New Audit catalog 390x844");
      } finally {
        await managerSession.page.setViewportSize({ width: 1280, height: 800 });
      }
      const suggestedQuery = new URLSearchParams({ usageClass: "GOVERNED_OPERATIONAL", scopeId: scopeDraftId, applicationType: scenario.applicationType, includedByDefault: "true", limit: "25" });
      const suggestedPage = await getApiJson<CanonicalQuestionCatalogPage>(managerSession.page, `/v1/question-catalogs/${encodeURIComponent(scenario.catalogVersion)}/questions?${suggestedQuery.toString()}`);
      expect(suggestedPage.totalCount).toBeGreaterThan(0);
      expect(suggestedPage.items.every((item) => item.recommendation?.includedByDefault === true)).toBe(true);
      expect(suggestedPage.items.some((item) => (item.recommendation?.signalCodes ?? []).includes("AUDIT_TYPE_FOCUS_MATCH"))).toBe(true);
      expect(suggestedPage.items.some((item) => (item.recommendation?.signalCodes ?? []).some((reason) => ["HIGH_OR_UNKNOWN_RISK", "RECURRENCE_DUE", "OPEN_WORK", "INSUFFICIENT_LONGITUDINAL_HISTORY", "NON_CLEAN_OR_MISSING_ANSWER", "OUTSIDE_SELECTED_FOCUS"].includes(reason)))).toBe(true);

      const representativeForm = catalogOracle.rows[0]?.formCode;
      expect(representativeForm).toBeTruthy();
      const formFilterQuery = new URLSearchParams({ usageClass: "GOVERNED_OPERATIONAL", scopeId: scopeDraftId, applicationType: scenario.applicationType, formCode: representativeForm ?? "", limit: "25", projection: "selection" });
      const formFilteredPage = await getApiJson<CanonicalQuestionCatalogPage>(managerSession.page, `/v1/question-catalogs/${encodeURIComponent(scenario.catalogVersion)}/questions?${formFilterQuery.toString()}`);
      expect(formFilteredPage.items.length).toBeGreaterThan(0);
      expect(formFilteredPage.items.every((item) => item.formCode === representativeForm)).toBe(true);
      writeEvent({ event: "manager-catalog-cursor-traversal", questionCount: seenIds.length, catalogRootDigest: catalogOracle.catalogRootDigest, rootMatchesManifest: catalogOracle.catalogRootDigest === scenario.catalogRootDigest, searchFilterVerified: true, formFilterVerified: true, applicationType: scenario.applicationType, typedSuggestionsVerified: true, selectionDigest: scenario.selectionDigest, status: "verified locally" });

      const selectionSummary = managerSession.page.getByRole("region", { name: "Selection summary" });
      await expect(selectionSummary).toContainText(`${scenario.selectedQuestionVersionIds.length} questions selected`);
      await selectionSummary.getByRole("button", { name: "Review selection" }).click();
      const selectionDialog = managerSession.page.getByRole("dialog", { name: "Review selection" });
      await expect(selectionDialog).toBeVisible();
      await selectionDialog.getByRole("button", { name: "Confirm selection" }).click();
      await expect(managerSession.page.locator(".planning-intake-status")).toContainText("Selection confirmed and saved to the server-owned scope.");
      const confirmedDraft = await getApiJson<{ selectionDigest?: string }>(managerSession.page, `/v1/planning/intake-drafts/${encodeURIComponent(draftId)}`);
      expect(confirmedDraft.selectionDigest).toBe(scenario.selectionDigest);
      await managerSession.page.getByRole("button", { name: "Continue", exact: true }).click();
      await managerSession.page.waitForURL(/\/department-manager\/new-audit\/step-5\?draftId=/);
      await expect(managerSession.page.getByRole("region", { name: "Planning intake form" })).toContainText(`${scenario.selectedQuestionVersionIds.length}`);
      await managerSession.page.getByRole("button", { name: "Submit to Finance" }).click();
      await managerSession.page.waitForURL(/\/department-manager\/audit-plan\?planningItemId=/);
      planningItemId = new URL(managerSession.page.url()).searchParams.get("planningItemId") ?? "";
      expect(planningItemId).toBeTruthy();
      const submittedPlanningItems = await getApiJson<{ items: PlanningPinView[] }>(managerSession.page, "/v1/planning/items?limit=100");
      const submittedPlanningItem = submittedPlanningItems.items.find((item) => item.id === planningItemId);
      expect(submittedPlanningItem, `submitted planning item ${planningItemId} must expose its immutable scope snapshot`).toBeTruthy();
      expect(submittedPlanningItem?.submittedScopeSnapshotId).toMatch(/^scope-snapshot:/);
      expect(submittedPlanningItem?.planningSnapshotDigest).toMatch(/^sha256:/);
      expect(submittedPlanningItem?.revision).toBeGreaterThan(0);
      expect(submittedPlanningItem?.organizationId).toBe(supplierOrganizationId);
      expect(submittedPlanningItem?.inspectionType).toContain(scenario.applicationType);
      planningItemTitle = submittedPlanningItem?.title ?? "";
      expect(planningItemTitle).toBeTruthy();
      submittedScopeSnapshotId = submittedPlanningItem?.submittedScopeSnapshotId ?? "";
      planningSnapshotDigest = submittedPlanningItem?.planningSnapshotDigest ?? "";
      planningRevision = submittedPlanningItem?.revision ?? 0;
      writeEvent({ event: "manager-planning-submitted", planningItemId, supplierOrganizationId, providerScopeId: scenario.providerScopeId, regulatedTargetId: scenario.regulatedTargetId, applicationType: scenario.applicationType, selectedQuestionCount: scenario.selectedQuestionVersionIds.length, selectionDigest: scenario.selectionDigest, submittedScopeSnapshotId, planningSnapshotDigest, planningRevision, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const leadMemberSession = await signIn(browser, manager);
    const leadMember = await findPreparedTeamMember(leadMemberSession.page, "leadInspector", leadInspector.displayName);
    const inspectorMember = await findPreparedTeamMember(leadMemberSession.page, "inspector", inspector.displayName);
    writeEvent({ event: "server-roster-subject-resolution", leadSubjectId: leadMember.subjectId, inspectorSubjectId: inspectorMember.subjectId, status: "verified locally" });

    const financeSession = await signIn(browser, finance);
    try {
      await financeSession.page.goto(`${origin}/finance/finance-review`, { waitUntil: "domcontentloaded" });
      const financeRow = financeSession.page.getByRole("row").filter({ hasText: recordReference("Plan", planningItemId) });
      await expect(financeRow).toHaveCount(1, { timeout: 30_000 });
      const financeAction = financeRow.getByRole("button");
      if (!(await financeAction.isDisabled())) await financeAction.click();
      await expect(financeSession.page.getByTestId("planning-status")).toHaveText("FINANCE_REVIEW");
      await financeSession.page.getByRole("button", { name: "Approve Budget" }).click();
      await financeSession.page.getByLabel("Finance decision reason").fill("Finance verified the released planning budget and exact immutable scope.");
      await financeSession.page.getByRole("button", { name: "Confirm Finance Decision" }).click();
      await expect(financeSession.page.getByTestId("planning-status")).toHaveText("GM_REVIEW");
      const financePin = await assertPlanningPin(financeSession.page, planningItemId, { snapshotId: submittedScopeSnapshotId, digest: planningSnapshotDigest, minimumRevision: planningRevision });
      planningRevision = financePin.revision;
      writeEvent({ event: "finance-approved-planning", planningItemId, submittedScopeSnapshotId, planningSnapshotDigest, planningRevision, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const gmReviewSession = await signIn(browser, gm);
    try {
      await gmReviewSession.page.goto(`${origin}/general-manager/planning`, { waitUntil: "domcontentloaded" });
      const planningRow = gmReviewSession.page.getByRole("row").filter({ hasText: planningItemId });
      await expect(planningRow).toHaveCount(1);
      await planningRow.getByRole("button", { name: `Review ${planningItemId}` }).click();
      await expect(gmReviewSession.page.getByTestId("planning-status")).toHaveText("GM_REVIEW");
      await gmReviewSession.page.getByLabel("General Manager decision reason").fill("General Manager verified the exact target scope and operational readiness.");
      await gmReviewSession.page.getByRole("button", { name: `Forward ${planningItemId} to Executive Director` }).click();
      await expect(gmReviewSession.page.getByTestId("planning-status")).toHaveText("EXECUTIVE_DIRECTOR_REVIEW");
      const gmPin = await assertPlanningPin(gmReviewSession.page, planningItemId, { snapshotId: submittedScopeSnapshotId, digest: planningSnapshotDigest, minimumRevision: planningRevision });
      planningRevision = gmPin.revision;
      writeEvent({ event: "gm-forwarded-planning", planningItemId, submittedScopeSnapshotId, planningSnapshotDigest, planningRevision, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const executiveSession = await signIn(browser, executiveDirector);
    try {
      await executiveSession.page.goto(`${origin}/executive-director/planning`, { waitUntil: "domcontentloaded" });
      const planningRow = executiveSession.page.getByRole("row").filter({ hasText: planningItemId });
      await expect(planningRow).toHaveCount(1);
      await planningRow.getByRole("button", { name: `Review ${planningItemId}` }).click();
      await expect(executiveSession.page.getByTestId("planning-status")).toHaveText("EXECUTIVE_DIRECTOR_REVIEW");
      await executiveSession.page.getByLabel("Executive Director plan decision reason").fill("Executive Director approved the governed planning record for release.");
      await executiveSession.page.getByRole("button", { name: `Approve plan ${planningItemId}` }).click();
      await expect(executiveSession.page.getByTestId("planning-status")).toHaveText("GM_RELEASE");
      const executivePin = await assertPlanningPin(executiveSession.page, planningItemId, { snapshotId: submittedScopeSnapshotId, digest: planningSnapshotDigest, minimumRevision: planningRevision });
      planningRevision = executivePin.revision;
      writeEvent({ event: "executive-approved-planning", planningItemId, submittedScopeSnapshotId, planningSnapshotDigest, planningRevision, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const gmReleaseSession = await signIn(browser, gm);
    try {
      await gmReleaseSession.page.goto(`${origin}/general-manager/planning`, { waitUntil: "domcontentloaded" });
      const planningRow = gmReleaseSession.page.getByRole("row").filter({ hasText: planningItemId });
      await expect(planningRow).toHaveCount(1);
      await planningRow.getByRole("button", { name: `Review ${planningItemId}` }).click();
      await expect(gmReleaseSession.page.getByTestId("planning-status")).toHaveText("GM_RELEASE");
      await gmReleaseSession.page.getByLabel("General Manager decision reason").fill("General Manager released the approved plan to Department Manager preparation.");
      const releaseResponse = gmReleaseSession.page.waitForResponse((response) => response.request().method() === "POST" && response.url().includes("/api/v1/planning/items/") && response.url().endsWith("/decisions"));
      await gmReleaseSession.page.getByRole("button", { name: `Release ${planningItemId} to Department Manager` }).click();
      const releaseResult = await releaseResponse;
      if (!releaseResult.ok()) {
        const problem = await releaseResult.json() as { title?: string; code?: string; detail?: string };
        throw new Error(`GM release command failed with HTTP ${releaseResult.status()} (${problem.code ?? problem.title ?? "unknown"}): ${problem.detail ?? "no detail"}.`);
      }
      await expect(gmReleaseSession.page.getByTestId("planning-status")).toHaveText("RELEASED");
      const releasePin = await assertPlanningPin(gmReleaseSession.page, planningItemId, { snapshotId: submittedScopeSnapshotId, digest: planningSnapshotDigest, minimumRevision: planningRevision });
      planningRevision = releasePin.revision;
      writeEvent({ event: "gm-released-planning", planningItemId, submittedScopeSnapshotId, planningSnapshotDigest, planningRevision, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const preparationSession = await signIn(browser, manager);
    try {
      await preparationSession.page.goto(`${origin}/department-manager/audit-plan?planningItemId=${encodeURIComponent(planningItemId)}`, { waitUntil: "domcontentloaded" });
      const preparationPanel = preparationSession.page.getByTestId("canonical-preparation-actions");
      await expect(preparationPanel).toBeVisible();
      await preparationPanel.getByRole("button", { name: "Begin preparation" }).click();
      await expect(preparationPanel).toContainText("PREPARATION");
      const preparationState = await getApiJson<{ id?: string }>(preparationSession.page, `/v1/audit-assignments/preparations/current?planningItemId=${encodeURIComponent(planningItemId)}`);
      assignmentId = preparationState.id ?? "";
      expect(assignmentId).toBeTruthy();
      await preparationPanel.getByLabel("Lead Inspector subject ID").fill(leadMember.subjectId);
      await preparationPanel.getByRole("button", { name: "Assign Lead Inspector" }).click();
      await expect(preparationPanel).toContainText("Assignment #");
      writeEvent({ event: "manager-assigned-lead", planningItemId, assignmentId, leadSubjectId: leadMember.subjectId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const leadSession = await signIn(browser, leadInspector);
    try {
      await leadSession.page.goto(`${origin}/lead-inspector/audit-preparation?assignmentId=${encodeURIComponent(assignmentId)}`, { waitUntil: "domcontentloaded" });
      const leadPreparation = leadSession.page.getByTestId("lead-audit-assignment-page");
      await expect(leadPreparation).toContainText("Pre-materialization preparation");
      await leadPreparation.getByLabel("Inspector subject IDs").fill(inspectorMember.subjectId);
      await leadPreparation.getByRole("button", { name: "Preview exact team" }).click();
      await expect(leadPreparation.getByRole("region", { name: "Team assignment preview" })).toContainText("Preview ready");
      await leadPreparation.getByRole("button", { name: "Confirm team assignment" }).click();
      await expect(leadPreparation).toContainText("TEAM_ASSIGNED");
      await leadPreparation.getByLabel("Coverage Inspector").selectOption(inspectorMember.subjectId);
      await leadPreparation.getByRole("button", { name: "Stage all released questions for Inspector" }).click();
      await expect(leadPreparation).toContainText(`${scenario.selectedQuestionVersionIds.length} released questions staged`);
      await leadPreparation.getByRole("button", { name: "Preview next coverage batch" }).click();
      await expect(leadPreparation.getByRole("region", { name: "Question coverage preview" })).toContainText("Preview ready");
      await leadPreparation.getByRole("button", { name: "Confirm question coverage batch" }).click();
      await expect(leadPreparation).toContainText(`${scenario.selectedQuestionVersionIds.length} exact question assignments committed`);
      await expect(leadPreparation).toContainText("QUESTIONS_ASSIGNED");
      writeEvent({ event: "lead-assigned-exact-coverage", assignmentId, selectedQuestionCount: scenario.selectedQuestionVersionIds.length, inspectorSubjectId: inspectorMember.subjectId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const materializeSession = await signIn(browser, manager);
    try {
      await materializeSession.page.goto(`${origin}/department-manager/audit-plan?planningItemId=${encodeURIComponent(planningItemId)}`, { waitUntil: "domcontentloaded" });
      const preparationPanel = materializeSession.page.getByTestId("canonical-preparation-actions");
      await expect(preparationPanel).toContainText("Assignment #");
      await expect(preparationPanel).toContainText("QUESTIONS ASSIGNED");
      await preparationPanel.getByRole("button", { name: "Confirm preparation" }).click();
      await expect(materializeSession.page.getByRole("status").filter({ hasText: "preparation confirmed" })).toBeVisible();
      const materializationResponse = materializeSession.page.waitForResponse((response) =>
        response.request().method() === "POST" && response.url().includes(`/api/v1/audit-assignments/${encodeURIComponent(assignmentId)}/materializations`),
      );
      await preparationPanel.getByRole("button", { name: "Create Audit" }).click();
      await expect(preparationPanel).toContainText("Inspector start is available");
      const materializationResult = await materializationResponse;
      expect(materializationResult.ok()).toBe(true);
      const materializedState = await materializationResult.json() as { inspectionId?: string; packageId?: string; status?: string };
      inspectionId = materializedState.inspectionId ?? "";
      packageId = materializedState.packageId ?? "";
      expect(inspectionId).toBeTruthy();
      expect(packageId).toBeTruthy();
      expect(materializedState.status).toBe("AWAITING_AUDITEE_CONFIRMATION");
      writeEvent({ event: "manager-materialized-canonical-audit", planningItemId, assignmentId, inspectionId, packageId, selectedQuestionCount: scenario.selectedQuestionVersionIds.length, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const targetAuditeeSession = await signIn(browser, targetAuditee);
    try {
      await targetAuditeeSession.page.goto(`${origin}/auditee/inspection-coordination`, { waitUntil: "domcontentloaded" });
      const targetCoordination = targetAuditeeSession.page.locator(`[data-audit-id="${inspectionId}"]`);
      await expect(targetCoordination).toHaveCount(1);
      await expect(targetCoordination).toContainText(targetAuditee.organizationId);
      await expect(targetCoordination).toContainText("AWAITING_AUDITEE_CONFIRMATION");
      await targetCoordination.getByRole("button", { name: "Confirm Proposed Date" }).click();
      await expect(targetCoordination).toContainText("CONFIRMED");
      writeEvent({ event: "target-auditee-coordination-confirmed", inspectionId, organizationId: targetAuditee.organizationId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const controlAuditeeSession = await signIn(browser, controlAuditee);
    try {
      await assertControlDomIsolation(controlAuditeeSession.page, "/auditee/inspection-coordination", [inspectionId, targetAuditee.organizationId]);
      await expect(controlAuditeeSession.page.locator(`[data-audit-id="${inspectionId}"]`)).toHaveCount(0);
      await expect(controlAuditeeSession.page.locator("[data-audit-id]")).toHaveCount(0);
      await assertControlApiIsolation(controlAuditeeSession.page, [
        "/v1/auditee/coordination",
        `/v1/inspection-packages/${encodeURIComponent(packageId)}`,
      ], [inspectionId, targetAuditee.organizationId, packageId], {
        detailPaths: [`/v1/inspection-packages/${encodeURIComponent(packageId)}`],
      });
      writeEvent({ event: "control-auditee-tenant-isolation", inspectionId, targetVisibleInApi: false, targetVisibleInDom: false, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const inspectorSession = await signIn(browser, inspector);
    try {
      await inspectorSession.page.goto(`${origin}/inspector/inspector-assignments`, { waitUntil: "domcontentloaded" });
      const assignmentRow = inspectorSession.page.locator(`[data-audit-id="${inspectionId}"]`).locator("xpath=ancestor::tr");
      await expect(assignmentRow).toHaveCount(1);
      await expect(assignmentRow).not.toContainText("inspection:assignment:");
      await expect(assignmentRow).toContainText("SCHEDULED");
      await assignmentRow.getByRole("button", { name: "Start inspection" }).click();
      await expect(assignmentRow).toContainText("IN PROGRESS");
      await expect(assignmentRow.getByRole("link", { name: /Open / })).toBeVisible();
      writeEvent({ event: "inspector-started-audit", inspectionId, packageId, status: "verified locally" });

      await inspectorSession.page.goto(`${origin}/inspector/audits/${encodeURIComponent(inspectionId)}/checklist?packageId=${encodeURIComponent(packageId)}`, { waitUntil: "domcontentloaded" });
      const checklist = inspectorSession.page.getByTestId("checklist-response-panel");
      await expect(checklist).toBeVisible();
      await inspectorSession.page.setViewportSize({ width: 390, height: 844 });
      try {
        await inspectorSession.page.goto(`${origin}/inspector/audits/${encodeURIComponent(inspectionId)}/checklist?packageId=${encodeURIComponent(packageId)}`, { waitUntil: "domcontentloaded" });
        const mobileChecklist = inspectorSession.page.getByTestId("checklist-response-panel");
        await expect(mobileChecklist).toBeVisible();
        const mobileQuestion = inspectorSession.page.getByTestId(`question-${scenario.selectedQuestionVersionIds[0]}`);
        await expect(mobileQuestion).toHaveCount(1);
        await mobileQuestion.locator("xpath=ancestor::tr").getByRole("button", { name: /Open question/ }).click();
        await expect(mobileChecklist.locator("b", { hasText: "Current owner" })).toBeVisible();
        await expect(mobileChecklist.locator("b", { hasText: "Next action" })).toBeVisible();
        await expectResponsiveViewport(inspectorSession.page, "Inspector Checklist 390x844");
        writeEvent({ event: "inspector-responsive-surface", surface: "checklist", viewport: "390x844", status: "verified locally" });
      } finally {
        await inspectorSession.page.setViewportSize({ width: 1280, height: 800 });
      }
      for (const [index, questionId] of scenario.selectedQuestionVersionIds.entries()) {
        const question = inspectorSession.page.getByTestId(`question-${questionId}`);
        await expect(question).toHaveCount(1);
        const row = question.locator("xpath=ancestor::tr");
        await row.getByRole("button", { name: /Open question/ }).click();
        const answer = index === 0 ? "NON_COMPLIANT" : "COMPLIANT";
        await inspectorSession.page.getByLabel("Checklist answer").selectOption(answer);
        await inspectorSession.page.getByLabel("Inspector comment").fill(index === 0 ? "Observed a serviceability exception requiring corrective action." : `Question ${index + 1} verified against the released package.`);
        const responseSave = inspectorSession.page.waitForResponse((response) =>
          response.request().method() === "PUT" && response.url().includes("/api/v1/checklist-responses/"),
        );
        await checklist.getByRole("button", { name: "Save response" }).click();
        const responseSaveResult = await responseSave;
        if (!responseSaveResult.ok()) {
          const problem = await responseSaveResult.json() as { title?: string; code?: string; detail?: string };
          throw new Error(`Checklist response save failed with HTTP ${responseSaveResult.status()} (${problem.code ?? problem.title ?? "unknown"}): ${problem.detail ?? "no detail"}.`);
        }
        await expect(checklist.getByTestId("response-status")).toContainText(answer.replaceAll("_", " "));
        if (index === 0) {
          await checklist.getByRole("button", { name: "Create Potential Finding" }).click();
          potentialFindingId = (await checklist.getByTestId("potential-finding-id").textContent())?.trim() ?? "";
          expect(potentialFindingId).toBeTruthy();
          await expect(checklist.getByTestId("potential-finding-status")).toHaveText("PENDING_LEAD_REVIEW");
        }
      }
      const checklistWorkflowFooter = inspectorSession.page.locator(".inspector-workflow-footer");
      const checklistSubmitResponse = inspectorSession.page.waitForResponse((response) =>
        response.request().method() === "POST" && response.url().includes(`/api/v1/checklists/${encodeURIComponent(inspectionId)}/submit`),
      );
      await checklistWorkflowFooter.getByRole("button", { name: "Submit checklist to Lead Inspector" }).click({ timeout: 10_000 });
      const checklistSubmitResult = await checklistSubmitResponse;
      if (!checklistSubmitResult.ok()) {
        const problem = await checklistSubmitResult.json() as { title?: string; code?: string; detail?: string };
        throw new Error(`Checklist submission failed with HTTP ${checklistSubmitResult.status()} (${problem.code ?? problem.title ?? "unknown"}): ${problem.detail ?? "no detail"}.`);
      }
      await expect(inspectorSession.page.getByTestId("checklist-status")).toHaveText("SUBMITTED");
      writeEvent({ event: "inspector-executed-checklist", inspectionId, questionCount: scenario.selectedQuestionVersionIds.length, nonCompliantQuestionId: scenario.selectedQuestionVersionIds[0], potentialFindingId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const leadFindingSession = await signIn(browser, leadInspector);
    try {
      await leadFindingSession.page.goto(`${origin}/lead-inspector/lead-review`, { waitUntil: "domcontentloaded" });
      const potentialFinding = leadFindingSession.page.getByRole("button", { name: `${potentialFindingId} · PENDING LEAD REVIEW` });
      await expect(potentialFinding).toHaveCount(1);
      await potentialFinding.click();
      const dossier = leadFindingSession.page.getByTestId("potential-finding-dossier");
      await expect(dossier).toBeVisible();
      const preliminaryCreateResponse = leadFindingSession.page.waitForResponse((response) =>
        response.request().method() === "POST" && response.url().endsWith("/api/v1/report-versions"),
      );
      await dossier.getByRole("button", { name: "Create Preliminary Report" }).click();
      const preliminaryCreateResult = await preliminaryCreateResponse;
      if (!preliminaryCreateResult.ok()) {
        const problem = await preliminaryCreateResult.json() as { title?: string; code?: string; detail?: string };
        throw new Error(`Preliminary Report creation failed with HTTP ${preliminaryCreateResult.status()} (${problem.code ?? problem.title ?? "unknown"}): ${problem.detail ?? "no detail"}.`);
      }
      const preliminaryReport = await preliminaryCreateResult.json() as ReportVersionApiView;
      expect(preliminaryReport.kind).toBe("PRELIMINARY");
      expect(preliminaryReport.auditId).toBe(inspectionId);
      expect(preliminaryReport.findingIds).toEqual([]);
      expect(preliminaryReport.potentialFindingIds).toEqual([potentialFindingId]);
      preliminaryPotentialRootDigest = potentialFindingRootDigest(preliminaryReport.potentialFindingIds);
      expect(preliminaryReport.potentialFindingRootDigest).toBe(preliminaryPotentialRootDigest);
      preliminaryReportVersionId = preliminaryReport.reportVersionId;
      preliminaryContentHash = preliminaryReport.contentHash;
      const persistedPreliminaryReport = await getApiJson<ReportVersionApiView>(leadFindingSession.page, `/v1/report-versions/${encodeURIComponent(preliminaryReportVersionId)}`);
      expect(persistedPreliminaryReport.findingIds).toEqual([]);
      expect(persistedPreliminaryReport.potentialFindingIds).toEqual([potentialFindingId]);
      expect(persistedPreliminaryReport.potentialFindingRootDigest).toBe(preliminaryPotentialRootDigest);
      expect(persistedPreliminaryReport.contentHash).toBe(preliminaryContentHash);
      expect(await dossier.getByTestId("preliminary-report-version-id").textContent()).toBe(preliminaryReportVersionId);
      expect(preliminaryReportVersionId).toBeTruthy();
      await expect(dossier.getByTestId("preliminary-report-status")).toHaveText("DEPARTMENT_REVIEW");
      writeEvent({ event: "lead-created-preliminary-report", inspectionId, preliminaryReportVersionId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const managerPreliminarySession = await signIn(browser, manager);
    try {
      await managerPreliminarySession.page.goto(`${origin}/department-manager/preliminary-reports/${encodeURIComponent(preliminaryReportVersionId)}`, { waitUntil: "domcontentloaded" });
      const managerReview = managerPreliminarySession.page.getByTestId("manager-preliminary-review-page");
      await expect(managerReview).toBeVisible();
      await expect(managerReview.getByTestId("manager-preliminary-status")).toHaveText("DEPARTMENT_REVIEW");
      await managerReview.getByLabel("Department Manager decision reason").fill("Department Manager verified the immutable Preliminary Report execution boundary.");
      await managerReview.getByRole("button", { name: `Forward ${preliminaryReportVersionId} to General Manager` }).click();
      await expect(managerReview.getByTestId("manager-preliminary-status")).toHaveText("GM_REVIEW");
      writeEvent({ event: "manager-forwarded-preliminary-report", inspectionId, preliminaryReportVersionId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const gmPreliminarySession = await signIn(browser, gm);
    try {
      await gmPreliminarySession.page.goto(`${origin}/general-manager/report-approvals`, { waitUntil: "domcontentloaded" });
      const gmRow = gmPreliminarySession.page.getByRole("row").filter({ hasText: preliminaryReportVersionId });
      await expect(gmRow).toHaveCount(1);
      await gmRow.getByRole("button", { name: `Open ${preliminaryReportVersionId}` }).click();
      const gmDossier = gmPreliminarySession.page.getByRole("region", { name: `Selected report ${preliminaryReportVersionId}` });
      await expect(gmDossier).toBeVisible();
      await gmDossier.getByLabel("General Manager report decision reason").fill("General Manager verified the exact Preliminary Report version and review history.");
      await gmDossier.getByRole("button", { name: `Forward ${preliminaryReportVersionId} to Executive Director` }).click();
      await expect(gmDossier.getByTestId("report-status")).toHaveText("EXECUTIVE_DIRECTOR_REVIEW");
      writeEvent({ event: "gm-forwarded-preliminary-report", inspectionId, preliminaryReportVersionId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const executivePreliminarySession = await signIn(browser, executiveDirector);
    try {
      await executivePreliminarySession.page.goto(`${origin}/executive-director/preliminary-reports`, { waitUntil: "domcontentloaded" });
      const executiveRow = executivePreliminarySession.page.getByRole("row").filter({ hasText: preliminaryReportVersionId });
      await expect(executiveRow).toHaveCount(1);
      await executiveRow.getByRole("button", { name: `Open ${preliminaryReportVersionId}` }).click();
      const executiveDossier = executivePreliminarySession.page.getByRole("region", { name: `Selected Preliminary Report ${preliminaryReportVersionId}` });
      await expect(executiveDossier).toBeVisible();
      await executiveDossier.getByLabel("Executive Director report decision reason").fill("Executive Director issued and locked the exact Preliminary Report version.");
      await executiveDossier.getByRole("button", { name: `Issue and lock ${preliminaryReportVersionId}` }).click();
      await expect(executiveDossier.getByTestId("report-status")).toHaveText("LOCKED");
      writeEvent({ event: "executive-issued-preliminary-report", inspectionId, preliminaryReportVersionId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const leadFindingConversionSession = await signIn(browser, leadInspector);
    try {
      await leadFindingConversionSession.page.goto(`${origin}/lead-inspector/lead-review`, { waitUntil: "domcontentloaded" });
      const potentialFinding = leadFindingConversionSession.page.getByRole("button", { name: `${potentialFindingId} · PENDING LEAD REVIEW` });
      await expect(potentialFinding).toHaveCount(1);
      await potentialFinding.click();
      const dossier = leadFindingConversionSession.page.getByTestId("potential-finding-dossier");
      await expect(dossier).toBeVisible();
      await dossier.getByLabel("Finding severity").selectOption("LEVEL_3_MINOR");
      await dossier.getByRole("button", { name: "Convert to Finding" }).click();
      await expect(leadFindingConversionSession.page.getByTestId("lead-decision-result")).toBeVisible();
      const findingLink = leadFindingConversionSession.page.getByTestId("lead-decision-result").getByRole("link", { name: "Open Lead CAP review" });
      const findingHref = await findingLink.getAttribute("href");
      findingId = findingHref?.split("/").pop() ?? "";
      expect(findingId).toBeTruthy();
      await expect(leadFindingConversionSession.page.getByTestId("finding-status")).toHaveText("WAITING_FOR_CAP");
      const preliminaryAfterConversion = await getApiJson<ReportVersionApiView>(leadFindingConversionSession.page, `/v1/report-versions/${encodeURIComponent(preliminaryReportVersionId)}`);
      expect(preliminaryAfterConversion.findingIds).toEqual([]);
      expect(preliminaryAfterConversion.contentHash).toBe(preliminaryContentHash);
      writeEvent({ event: "lead-converted-potential-finding", inspectionId, potentialFindingId, findingId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const controlFindingSession = await signIn(browser, controlAuditee);
    try {
      await assertControlDomIsolation(controlFindingSession.page, "/auditee/service-provider-cap", [inspectionId, targetAuditee.organizationId, potentialFindingId, findingId]);
      await assertControlApiIsolation(controlFindingSession.page, [
        "/v1/findings",
        `/v1/findings/${encodeURIComponent(findingId)}`,
        `/v1/potential-findings/${encodeURIComponent(potentialFindingId)}`,
      ], [inspectionId, targetAuditee.organizationId, potentialFindingId, findingId], {
        detailPaths: [
          `/v1/findings/${encodeURIComponent(findingId)}`,
          `/v1/potential-findings/${encodeURIComponent(potentialFindingId)}`,
        ],
      });
      writeEvent({ event: "control-auditee-finding-isolation", inspectionId, potentialFindingId, findingId, targetVisibleInApi: false, targetVisibleInDom: false, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const targetCapSession = await signIn(browser, targetAuditee);
    let findingNumber = "";
    try {
      await targetCapSession.page.goto(`${origin}/auditee/service-provider-cap`, { waitUntil: "domcontentloaded" });
      const auditeePage = targetCapSession.page.getByTestId("auditee-page");
      await expect(auditeePage).toBeVisible();
      const auditeeFindings = await getApiJson<{ items: Array<{ id: string; findingNumber: string; status: string }> }>(targetCapSession.page, "/v1/findings");
      const targetFinding = auditeeFindings.items.find((item) => item.id === findingId);
      expect(targetFinding).toBeTruthy();
      findingNumber = targetFinding?.findingNumber ?? "";
      const findingRow = auditeePage.getByRole("row").filter({ hasText: findingNumber });
      await expect(findingRow).toHaveCount(1);
      await findingRow.getByRole("button", { name: findingNumber, exact: true }).click();
      const selectedFinding = auditeePage.getByTestId("auditee-selected-finding");
      await expect(selectedFinding).toBeVisible();
      await expect(selectedFinding.getByTestId("finding-status")).toContainText("Waiting for CAP");
      await auditeePage.getByRole("textbox", { name: "Root cause", exact: true }).fill("The configured serviceability control was not consistently documented.");
      await auditeePage.getByRole("textbox", { name: "Corrective action", exact: true }).fill("Restore the documented serviceability inspection and record each result.");
      await auditeePage.getByRole("textbox", { name: "Preventive action", exact: true }).fill("Add a recurring supervisor verification to the operating checklist.");
      await auditeePage.getByRole("textbox", { name: "Responsible person", exact: true }).fill("Target Auditee Safety Manager");
      await auditeePage.getByRole("textbox", { name: "Target completion date", exact: true }).fill("2026-09-15");
      await auditeePage.getByRole("textbox", { name: "Comment to CAA", exact: true }).fill("CAP submitted through the connected Auditee workflow.");
      await auditeePage.getByRole("button", { name: "Submit CAP" }).click();
      await expect(selectedFinding.getByTestId("finding-status")).toContainText("CAP Submitted");
      writeEvent({ event: "target-auditee-submitted-cap", inspectionId, findingId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const leadCapSession = await signIn(browser, leadInspector);
    try {
      await leadCapSession.page.goto(`${origin}/lead-inspector/cap-review/${encodeURIComponent(findingId)}`, { waitUntil: "domcontentloaded" });
      const capReview = leadCapSession.page.getByTestId("cap-review-target");
      await expect(capReview).toBeVisible();
      await leadCapSession.page.getByLabel("Comment to Auditee").fill("The CAP addresses the Finding and may proceed to Evidence.");
      await leadCapSession.page.getByLabel("Internal CAA Note").fill("Lead review accepted the CAP as a separate lifecycle decision.");
      await leadCapSession.page.getByRole("button", { name: "Accept CAP" }).click();
      await expect(leadCapSession.page.getByTestId("finding-status")).toHaveText("EVIDENCE_REQUIRED");
      writeEvent({ event: "lead-accepted-cap", inspectionId, findingId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const targetEvidenceSession = await signIn(browser, targetAuditee);
    let evidenceVersionId = "";
    let evidenceVersion = 0;
    try {
      await targetEvidenceSession.page.goto(`${origin}/auditee/service-provider-cap`, { waitUntil: "domcontentloaded" });
      const auditeePage = targetEvidenceSession.page.getByTestId("auditee-page");
      await expect(auditeePage).toBeVisible();
      const findingRow = auditeePage.getByRole("row").filter({ hasText: findingNumber });
      await expect(findingRow).toHaveCount(1);
      await findingRow.getByRole("button", { name: findingNumber, exact: true }).click();
      const selectedFinding = auditeePage.getByTestId("auditee-selected-finding");
      await expect(selectedFinding.getByTestId("finding-status")).toContainText("Evidence Required");
      const evidenceFile = auditeePage.getByTestId("evidence-file");
      await evidenceFile.setInputFiles({
        name: "qualification-evidence.pdf",
        mimeType: "application/pdf",
        buffer: Buffer.from("%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\n%%EOF\n"),
      });
      await expect(auditeePage.getByTestId("selected-evidence-file")).toHaveText("qualification-evidence.pdf");
      const submitEvidenceButton = auditeePage.getByRole("button", { name: "Submit Evidence version" });
      writeEvent({
        event: "target-auditee-evidence-submit-preflight",
        url: targetEvidenceSession.page.url(),
        online: await targetEvidenceSession.page.evaluate(() => navigator.onLine),
        disabled: await submitEvidenceButton.isDisabled(),
        status: "verified locally",
      });
      await expect(submitEvidenceButton).toBeEnabled();
      const evidenceBeginRequest = targetEvidenceSession.page.waitForRequest((request) =>
        request.method() === "POST" && request.url().endsWith("/api/v1/evidence/uploads"),
        { timeout: 30_000 },
      );
      const evidenceBeginResponse = targetEvidenceSession.page.waitForResponse((response) =>
        response.request().method() === "POST" && response.url().endsWith("/api/v1/evidence/uploads"),
        { timeout: 30_000 },
      );
      const evidenceCompleteResponse = targetEvidenceSession.page.waitForResponse((response) =>
        response.request().method() === "POST" && /\/api\/v1\/evidence\/uploads\/[^/]+\/complete$/.test(response.url()),
        { timeout: 120_000 },
      );
      await submitEvidenceButton.click();
      const evidenceBeginRequestResult = await evidenceBeginRequest;
      writeEvent({
        event: "target-auditee-evidence-begin-requested",
        url: evidenceBeginRequestResult.url(),
        status: "verified locally",
      });
      const evidenceBeginResult = await evidenceBeginResponse;
      expect(evidenceBeginResult.ok(), `Evidence Begin failed with HTTP ${evidenceBeginResult.status()}`).toBe(true);
      const evidenceCompleteResult = await evidenceCompleteResponse;
      writeEvent({
        event: "target-auditee-evidence-complete-response",
        statusCode: evidenceCompleteResult.status(),
        ok: evidenceCompleteResult.ok(),
        url: evidenceCompleteResult.url(),
        status: "verified locally",
      });
      if (!evidenceCompleteResult.ok()) {
        const problem = await evidenceCompleteResult.json() as { code?: string; title?: string };
        writeEvent({
          event: "target-auditee-evidence-complete-problem",
          problemCode: problem.code ?? "",
          problemTitle: problem.title ?? "",
          status: "verified locally",
        });
      }
      expect(evidenceCompleteResult.ok()).toBe(true);
      const completedEvidence = await evidenceCompleteResult.json() as EvidenceCompleteApiView;
      evidenceVersionId = completedEvidence.evidenceVersionId;
      evidenceVersion = completedEvidence.version;
      expect(evidenceVersionId).toBeTruthy();
      expect(evidenceVersion).toBeGreaterThan(0);
      // The API completion response precedes the UI's bounded CLEAN poll.
      // Wait for that connected state refresh instead of racing the worker.
      await expect(auditeePage.getByTestId("evidence-version-count")).toHaveText("1", { timeout: 45_000 });
      writeEvent({ event: "target-auditee-evidence-ui-refreshed", evidenceVersionId, evidenceVersion, status: "verified locally" });
      const evidence = await getApiJson<{ items: Array<{ id: string; version: number; scanState: string; reviewState: string }> }>(targetEvidenceSession.page, `/v1/findings/${encodeURIComponent(findingId)}/evidence`);
      const uploadedEvidence = evidence.items.find((item) => item.id === evidenceVersionId);
      expect(uploadedEvidence).toBeTruthy();
      expect(uploadedEvidence?.version).toBe(evidenceVersion);
      expect(uploadedEvidence?.scanState).toBe("CLEAN");
      expect(uploadedEvidence?.reviewState).toBe("PENDING_CAA_REVIEW");
      writeEvent({ event: "target-auditee-uploaded-evidence", inspectionId, findingId, evidenceVersionId, evidenceVersion, scanState: uploadedEvidence?.scanState ?? "", reviewState: uploadedEvidence?.reviewState ?? "", status: "verified locally" });
      await targetEvidenceSession.page.setViewportSize({ width: 390, height: 844 });
      try {
        await targetEvidenceSession.page.goto(`${origin}/auditee/service-provider-cap`, { waitUntil: "domcontentloaded" });
        const mobileAuditeePage = targetEvidenceSession.page.getByTestId("auditee-page");
        await expect(mobileAuditeePage).toBeVisible();
        const mobileFindingRow = mobileAuditeePage.getByRole("row").filter({ hasText: findingNumber });
        await expect(mobileFindingRow).toHaveCount(1);
        await mobileFindingRow.getByRole("button", { name: findingNumber, exact: true }).click();
        const mobileFinding = mobileAuditeePage.getByTestId("auditee-selected-finding");
        await expect(mobileFinding).toBeVisible();
        await expect(mobileFinding.locator("dt", { hasText: "Current owner" })).toBeVisible();
        await expect(mobileFinding.locator("dt", { hasText: "Next action" })).toBeVisible();
        await expectResponsiveViewport(targetEvidenceSession.page, "Auditee CAP 390x844");
        writeEvent({ event: "auditee-responsive-surface", surface: "cap", viewport: "390x844", status: "verified locally" });
      } finally {
        await targetEvidenceSession.page.setViewportSize({ width: 1280, height: 800 });
      }
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const controlEvidenceSession = await signIn(browser, controlAuditee);
    try {
      await assertControlDomIsolation(controlEvidenceSession.page, "/auditee/service-provider-cap", [inspectionId, targetAuditee.organizationId, potentialFindingId, findingId, evidenceVersionId]);
      await assertControlApiIsolation(controlEvidenceSession.page, [
        `/v1/findings/${encodeURIComponent(findingId)}/evidence`,
        `/v1/findings/${encodeURIComponent(findingId)}/cap-revisions`,
      ], [inspectionId, targetAuditee.organizationId, potentialFindingId, findingId, evidenceVersionId], {
        detailPaths: [
          `/v1/findings/${encodeURIComponent(findingId)}/evidence`,
          `/v1/findings/${encodeURIComponent(findingId)}/cap-revisions`,
        ],
      });
      writeEvent({ event: "control-auditee-cap-evidence-isolation", inspectionId, findingId, evidenceVersionId, targetVisibleInApi: false, targetVisibleInDom: false, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const leadEvidenceSession = await signIn(browser, leadInspector);
    try {
      await leadEvidenceSession.page.goto(`${origin}/department-manager/evidence/${encodeURIComponent(findingId)}`, { waitUntil: "domcontentloaded" });
      const leadFinding = await getApiJson<{ id: string }>(leadEvidenceSession.page, `/v1/findings/${encodeURIComponent(findingId)}`);
      expect(leadFinding.id).toBe(findingId);
      await getApiJson(leadEvidenceSession.page, `/v1/findings/${encodeURIComponent(findingId)}/evidence`);
      await getApiJson(leadEvidenceSession.page, `/v1/findings/${encodeURIComponent(findingId)}/cap-revisions`);
      const evidenceReview = leadEvidenceSession.page.getByTestId("evidence-review-target");
      await expect(evidenceReview).toBeVisible();
      await expect(evidenceReview.getByTestId("reviewing-evidence-version")).toHaveText(`Version ${evidenceVersion}`);
      await expect(evidenceReview.getByTestId("reviewing-evidence-id")).toHaveText(evidenceVersionId);
      await evidenceReview.getByLabel("Evidence review decision").selectOption("CLOSE");
      await evidenceReview.getByLabel("Comment to Auditee").fill("Evidence was scanned clean and verifies the corrective action.");
      await evidenceReview.getByLabel("Internal CAA Note").fill("Lead Inspector verified the exact latest Evidence version before closure.");
      await evidenceReview.getByRole("button", { name: "Record Evidence review" }).click();
      await expect(leadEvidenceSession.page.getByTestId("finding-status")).toHaveText("CLOSED");
      await expect(leadEvidenceSession.page.getByTestId("closure-basis")).toHaveText("EVIDENCE_VERIFIED");
      writeEvent({ event: "lead-closed-finding-after-latest-evidence", inspectionId, findingId, evidenceVersionId, closureBasis: "EVIDENCE_VERIFIED", status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const finalCreationSession = await signIn(browser, leadInspector);
    try {
      const preliminaryBeforeFinal = await getApiJson<ReportVersionApiView>(finalCreationSession.page, `/v1/report-versions/${encodeURIComponent(preliminaryReportVersionId)}`);
      expect(preliminaryBeforeFinal.findingIds).toEqual([]);
      expect(preliminaryBeforeFinal.contentHash).toBe(preliminaryContentHash);
      await finalCreationSession.page.goto(`${origin}/lead-inspector/final-reports`, { waitUntil: "domcontentloaded" });
      const finalReports = finalCreationSession.page.getByTestId("lead-final-reports-page");
      const finalCreator = finalReports.getByTestId("final-report-creator");
      await expect(finalCreator).toBeVisible();
      await finalCreator.getByLabel("Final Report audit").selectOption(inspectionId);
      await expect(finalCreator).toContainText("1 closed Finding selected");
      const finalCreateResponse = finalCreationSession.page.waitForResponse((response) =>
        response.request().method() === "POST" && response.url().endsWith("/api/v1/report-versions"),
      );
      await finalCreator.getByRole("button", { name: "Create Final Report" }).click();
      const finalCreateResult = await finalCreateResponse;
      expect(finalCreateResult.ok()).toBe(true);
      const finalReport = await finalCreateResult.json() as ReportVersionApiView;
      expect(finalReport.kind).toBe("FINAL");
      expect(finalReport.auditId).toBe(inspectionId);
      expect(finalReport.findingIds).toEqual([findingId]);
      expect(finalReport.potentialFindingIds).toEqual([potentialFindingId]);
      finalPotentialRootDigest = potentialFindingRootDigest(finalReport.potentialFindingIds);
      expect(finalPotentialRootDigest).toBe(preliminaryPotentialRootDigest);
      expect(finalReport.potentialFindingRootDigest).toBe(finalPotentialRootDigest);
      await expect(finalCreator.getByTestId("final-report-status")).toHaveText("DEPARTMENT_REVIEW");
      finalReportVersionId = finalReport.reportVersionId;
      expect(await finalCreator.getByTestId("final-report-version-id").textContent()).toBe(finalReportVersionId);
      expect(finalReportVersionId).toBeTruthy();
      const persistedFinalReport = await getApiJson<ReportVersionApiView>(finalCreationSession.page, `/v1/report-versions/${encodeURIComponent(finalReportVersionId)}`);
      expect(persistedFinalReport.findingIds).toEqual([findingId]);
      writeEvent({ event: "lead-created-final-report", inspectionId, findingId, finalReportVersionId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const managerFinalSession = await signIn(browser, manager);
    try {
      await managerFinalSession.page.goto(`${origin}/department-manager/reports/${encodeURIComponent(finalReportVersionId)}`, { waitUntil: "domcontentloaded" });
      const finalDossier = managerFinalSession.page.getByTestId("report-version-dossier");
      await expect(finalDossier).toBeVisible();
      await expect(finalDossier.getByTestId("report-status")).toHaveText("DEPARTMENT_REVIEW");
      await finalDossier.getByLabel("Department Manager decision reason").fill("Department Manager verified that every linked Finding is closed with Evidence verification.");
      await finalDossier.getByRole("button", { name: "Forward to General Manager" }).click();
      await expect(finalDossier.getByTestId("report-status")).toHaveText("GM_REVIEW");
      writeEvent({ event: "manager-forwarded-final-report", inspectionId, finalReportVersionId, status: "verified locally" });
      await managerFinalSession.page.setViewportSize({ width: 390, height: 844 });
      try {
        await managerFinalSession.page.goto(`${origin}/department-manager/findings-review?findingId=${encodeURIComponent(findingId)}`, { waitUntil: "domcontentloaded" });
        const mobileManagerPage = managerFinalSession.page.getByTestId("manager-findings-review-page");
        await expect(mobileManagerPage).toBeVisible();
        await expect(mobileManagerPage.locator("dt", { hasText: "Current owner" })).toBeVisible();
        await expect(mobileManagerPage.locator("dt", { hasText: "Next action" })).toBeVisible();
        await expectResponsiveViewport(managerFinalSession.page, "Manager Findings Review 390x844");
        writeEvent({ event: "manager-responsive-surface", surface: "findings-review", viewport: "390x844", status: "verified locally" });
      } finally {
        await managerFinalSession.page.setViewportSize({ width: 1280, height: 800 });
      }
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const gmFinalSession = await signIn(browser, gm);
    try {
      await gmFinalSession.page.goto(`${origin}/general-manager/report-approvals`, { waitUntil: "domcontentloaded" });
      const finalRow = gmFinalSession.page.getByRole("row").filter({ hasText: finalReportVersionId });
      await expect(finalRow).toHaveCount(1);
      await finalRow.getByRole("button", { name: `Open ${finalReportVersionId}` }).click();
      const finalDossier = gmFinalSession.page.getByRole("region", { name: `Selected report ${finalReportVersionId}` });
      await expect(finalDossier).toBeVisible();
      await finalDossier.getByLabel("General Manager report decision reason").fill("General Manager verified the exact closed-Finding Final Report version.");
      await finalDossier.getByRole("button", { name: `Forward ${finalReportVersionId} to Executive Director` }).click();
      await expect(finalDossier.getByTestId("report-status")).toHaveText("EXECUTIVE_DIRECTOR_REVIEW");
      writeEvent({ event: "gm-forwarded-final-report", inspectionId, finalReportVersionId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const executiveFinalSession = await signIn(browser, executiveDirector);
    try {
      await executiveFinalSession.page.goto(`${origin}/executive-director/final-reports`, { waitUntil: "domcontentloaded" });
      const finalRow = executiveFinalSession.page.getByRole("row").filter({ hasText: finalReportVersionId });
      await expect(finalRow).toHaveCount(1);
      await finalRow.getByRole("button", { name: `Open ${finalReportVersionId}` }).click();
      const finalDossier = executiveFinalSession.page.getByRole("region", { name: `Selected Final Report ${finalReportVersionId}` });
      await expect(finalDossier).toBeVisible();
      await finalDossier.getByLabel("Executive Director report decision reason").fill("Executive Director issued and locked the exact Final Report version.");
      await finalDossier.getByRole("button", { name: `Issue and lock ${finalReportVersionId}` }).click();
      await expect(finalDossier.getByTestId("report-status")).toHaveText("LOCKED");
      writeEvent({ event: "executive-issued-final-report", inspectionId, finalReportVersionId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const targetFinalSession = await signIn(browser, targetAuditee);
    try {
      await targetFinalSession.page.goto(`${origin}/auditee/final-reports`, { waitUntil: "domcontentloaded" });
      const finalPage = targetFinalSession.page.getByTestId("auditee-final-reports-page");
      await expect(finalPage).toBeVisible();
      const finalRow = finalPage.getByRole("row").filter({ hasText: finalReportVersionId });
      await expect(finalRow).toHaveCount(1);
      await finalRow.getByRole("link", { name: "Open report" }).click();
      await targetFinalSession.page.waitForURL(`${origin}/auditee/reports/${encodeURIComponent(finalReportVersionId)}`);
      await expect(targetFinalSession.page.getByTestId("auditee-report-preview-page")).toBeVisible();
      writeEvent({ event: "target-auditee-viewed-final-report", inspectionId, finalReportVersionId, organizationId: targetAuditee.organizationId, status: "verified locally" });
      await targetFinalSession.page.setViewportSize({ width: 390, height: 844 });
      try {
        await targetFinalSession.page.goto(`${origin}/auditee/final-reports`, { waitUntil: "domcontentloaded" });
        await expect(targetFinalSession.page.getByTestId("auditee-final-reports-page")).toBeVisible();
        await expect(targetFinalSession.page.getByRole("row").filter({ hasText: finalReportVersionId })).toHaveCount(1);
        await expectResponsiveViewport(targetFinalSession.page, "Auditee Final Reports 390x844");
        writeEvent({ event: "auditee-responsive-surface", surface: "final-reports", viewport: "390x844", status: "verified locally" });
      } finally {
        await targetFinalSession.page.setViewportSize({ width: 1280, height: 800 });
      }
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const controlReportSession = await signIn(browser, controlAuditee);
    try {
      for (const routePath of ["/auditee/preliminary-reports", "/auditee/final-reports", `/auditee/reports/${encodeURIComponent(finalReportVersionId)}`]) {
        await assertControlDomIsolation(controlReportSession.page, routePath, [inspectionId, targetAuditee.organizationId, preliminaryReportVersionId, finalReportVersionId, findingId, evidenceVersionId]);
      }
      await assertControlApiIsolation(controlReportSession.page, [
        "/v1/auditee/report-versions?kind=PRELIMINARY",
        "/v1/auditee/report-versions?kind=FINAL",
        `/v1/auditee/report-versions/${encodeURIComponent(finalReportVersionId)}`,
        `/v1/report-versions/${encodeURIComponent(finalReportVersionId)}`,
      ], [inspectionId, targetAuditee.organizationId, preliminaryReportVersionId, finalReportVersionId, findingId, evidenceVersionId], {
        detailPaths: [
          `/v1/auditee/report-versions/${encodeURIComponent(finalReportVersionId)}`,
          `/v1/report-versions/${encodeURIComponent(finalReportVersionId)}`,
        ],
      });
      writeEvent({ event: "control-auditee-report-isolation", inspectionId, preliminaryReportVersionId, finalReportVersionId, targetVisibleInApi: false, targetVisibleInDom: false, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    const adminEvidenceSession = await signIn(browser, admin);
    try {
      await adminEvidenceSession.page.goto(`${origin}/admin/audit-log`, { waitUntil: "domcontentloaded" });
      const adminAuditLog = adminEvidenceSession.page.getByTestId("admin-audit-log-page");
      await expect(adminAuditLog).toBeVisible();
      await adminAuditLog.getByLabel("Audit entity").fill(evidenceVersionId);
      const evidenceAuditEvents = adminAuditLog.getByRole("listitem");
      await expect(evidenceAuditEvents).toHaveCount(2);
      await expect(evidenceAuditEvents.filter({ hasText: "evidence.uploaded" })).toHaveCount(1);
      await expect(evidenceAuditEvents.filter({ hasText: "evidence.scan_completed" })).toHaveCount(1);
      await expect(adminAuditLog.getByText(evidenceVersionId)).toHaveCount(2);
      await adminAuditLog.getByLabel("Audit action").fill("evidence.reviewed");
      await adminAuditLog.getByLabel("Audit entity").fill(findingId);
      const findingReviewEvents = adminAuditLog.getByRole("listitem");
      await expect(findingReviewEvents).toHaveCount(1);
      await expect(findingReviewEvents).toContainText(findingId);
      await expect(findingReviewEvents).toContainText("CLOSED");
      writeEvent({ event: "admin-verified-evidence-record", inspectionId, findingId, evidenceVersionId, finalReportVersionId, status: "verified locally" });
    } finally {
      // Shared role sessions are closed by the test-level cleanup hook.
    }

    await verifyPublicPrivateAdmin404(browser);
    await verifyRoleBoundaryMatrix(browser, [
      { account: admin, home: "/admin/users-roles", foreign: "/department-manager/audit-plan" },
      { account: manager, home: "/department-manager/audit-plan", foreign: "/finance/finance-review" },
      { account: finance, home: "/finance/finance-review", foreign: "/admin/users-roles" },
      { account: gm, home: "/general-manager/planning", foreign: "/finance/finance-review" },
      { account: executiveDirector, home: "/executive-director/planning", foreign: "/general-manager/planning" },
      { account: leadInspector, home: "/lead-inspector/lead-review", foreign: "/inspector/inspector-assignments" },
      { account: inspector, home: "/inspector/inspector-assignments", foreign: "/lead-inspector/lead-review" },
      { account: targetAuditee, home: "/auditee/service-provider-cap", foreign: "/department-manager/audit-plan" },
      { account: controlAuditee, home: "/auditee/service-provider-cap", foreign: "/admin/users-roles" },
    ]);
    writeEvent({ event: "all-role-boundary-matrix", accountCount: accounts.size, publicPrivateAdminStatus: 404, logoutSessionStatus: 401, status: "verified locally" });
  });
});
