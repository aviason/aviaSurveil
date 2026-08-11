import { expect, test } from "@playwright/test";

const apiURL = process.env.AVIA_HTTP_API_URL ?? "http://127.0.0.1:58081";
const testToken = process.env.AVIA_CANONICAL_TEST_TOKEN ?? "";

for (const viewport of [{ width: 1440, height: 900 }, { width: 390, height: 844 }]) {
  test(`HTTP governed checklist synthetic lifecycle is accessible and overflow-free at ${viewport.width}x${viewport.height}`, async ({ page, request }) => {
    const issues: string[] = [];
    const commands: string[] = [];
    page.on("console", (message) => { if (["error", "warning"].includes(message.type())) issues.push(message.text()); });
    page.on("pageerror", (error) => issues.push(error.message));
    page.on("request", (outgoing) => {
      if (outgoing.method() === "POST" && outgoing.url().includes("/v1/")) {
        commands.push(new URL(outgoing.url()).pathname.replace(/^\/api(?=\/v1\/)/, ""));
      }
    });
    await page.setViewportSize(viewport);

    const reset = await request.post(`${apiURL}/__test/reset`, {
      headers: { "x-avia-test-token": testToken },
    });
    expect(reset.ok()).toBe(true);

    await page.goto("/department-manager/checklist-management");
    await expect(page.getByTestId("governed-checklist-review")).toBeVisible();
    await expect(page.getByRole("heading", { name: "No current governed candidates" })).toBeVisible();
    await expect(page.getByRole("link", { name: "Question Review" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Technically approve" })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Publish checklist version" })).toHaveCount(0);

    await page.getByRole("combobox", { name: "Experience" }).selectOption("admin");
    if (viewport.width < 600) await page.getByRole("button", { name: "Open navigation" }).click();
    await page.getByRole("link", { name: "Checklist Builder", exact: true }).click();
    await expect(page.getByTestId("admin-checklist-builder-page")).toBeVisible();
    await page.getByRole("button", { name: "Import synthetic governed candidate" }).click();
    await expect(page.getByText("SYNTHETIC-OPS-AOC").first()).toBeVisible();
    await page.getByRole("button", { name: "Inspect governed generation run" }).click();
    await expect(page.getByRole("button", { name: "Submit exact candidate for department review" })).toBeEnabled();
    await page.getByRole("button", { name: "Submit exact candidate for department review" }).click();
    await expect(page.getByText("Submitted for department review. Admin has no technical approval or publication action.")).toBeVisible();

    await page.getByRole("combobox", { name: "Experience" }).selectOption("manager");
    if (viewport.width < 600) await page.getByRole("button", { name: "Open navigation" }).click();
    await page.getByRole("link", { name: "Checklist Management", exact: true }).click();
    const review = page.getByTestId("governed-checklist-review");
    await expect(review).toContainText("DEPARTMENT_REVIEW");
    await review.getByLabel("Decision reason").fill("Technically approve only the exact synthetic candidate.");
    await review.getByRole("button", { name: "Technically approve" }).click();
    await expect(review).toContainText("TECHNICALLY_APPROVED");
    await review.getByLabel("Decision reason").fill("Publish only the separately approved synthetic revision.");
    await review.getByRole("button", { name: "Publish checklist version" }).click();
    await expect(review).toContainText("PUBLISHED");
    await expect(review.getByText(/Published as immutable Checklist Template Version/i)).toBeVisible();

    const materialized = await request.post(`${apiURL}/__test/governed-checklist/materialize-synthetic`, {
      headers: { "x-avia-test-token": testToken },
    });
    expect(materialized.status()).toBe(201);
    await expect(materialized.json()).resolves.toMatchObject({
      inspectionId: "AUD-SYNTHETIC-OPS-AOC-001", packageId: "PKG-SYNTHETIC-OPS-AOC-001",
      templateVersionId: expect.any(String), packageDigest: expect.stringMatching(/^sha256:/),
    });

    await page.getByRole("combobox", { name: "Experience" }).selectOption("inspector");
    await page.evaluate(() => {
      window.history.pushState({}, "", "/inspector/audits/AUD-2026-001/checklist?packageId=PKG-SYNTHETIC-OPS-AOC-001");
      window.dispatchEvent(new PopStateEvent("popstate"));
    });
    await page.getByTestId("question-Q-SYNTHETIC-OPS-AOC-001").click();
    await page.getByLabel("Checklist answer").selectOption("NON_COMPLIANT");
    await page.getByLabel("Inspector comment").fill("Synthetic test-profile evidence could not be reconciled.");
    await page.getByRole("button", { name: "Save response" }).click();
    await expect(page.getByTestId("response-status")).toContainText("NON COMPLIANT");
    await page.getByLabel("Attachment file").setInputFiles({
      name: "synthetic-controlled-record.pdf", mimeType: "application/pdf",
      buffer: Buffer.from("%PDF-1.4\nsynthetic governed evidence\n%%EOF\n"),
    });
    await page.getByRole("button", { name: "Upload Inspection Attachment" }).click();
    await expect(page.getByTestId("inspection-attachment-uploaded")).toContainText("synthetic-controlled-record.pdf");
    await page.getByRole("button", { name: "Create Potential Finding" }).click();
    await expect(page.getByTestId("potential-finding-status")).toHaveText("PENDING_LEAD_REVIEW");
    await page.getByRole("button", { name: "Submit checklist to Lead Inspector" }).click();
    await expect(page.getByTestId("checklist-status")).toHaveText("SUBMITTED");

    expect(commands).toEqual(expect.arrayContaining([
      "/v1/admin/governed-checklist/generation-runs",
      "/v1/admin/governed-checklist/candidates/CAND-SYNTHETIC-OPS-AOC-0001/submissions",
      "/v1/department-manager/governed-checklist/candidates/CAND-SYNTHETIC-OPS-AOC-0001/technical-approvals",
      "/v1/department-manager/governed-checklist/candidates/CAND-SYNTHETIC-OPS-AOC-0001/publications",
    ]));
    expect(commands.some((path) => path.startsWith("/v1/inspection-attachments/"))).toBe(true);
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
    expect(issues).toEqual([]);
  });
}
