import { readFile } from "node:fs/promises";

import { expect, test, type APIRequestContext } from "@playwright/test";

const apiURL = process.env.AVIA_HTTP_API_URL ?? "http://127.0.0.1:58081";
const testToken = process.env.AVIA_CANONICAL_TEST_TOKEN ?? "";
const reportVersionId = "RPT-CAB-2026-001-V1";

function canonicalHeaders(subject: string, revision?: number): Record<string, string> {
  const headers: Record<string, string> = {
    "content-type": "application/json",
    "x-avia-test-token": testToken,
    "x-avia-test-subject": subject,
    "x-csrf-token": "canonical-document-test",
  };
  if (revision !== undefined) headers["if-match"] = `"rev-${revision}"`;
  return headers;
}

async function decideReport(
  request: APIRequestContext,
  subject: string,
  revision: number,
  decision: "FORWARD" | "ISSUE_AND_LOCK",
): Promise<void> {
  const operationId = `OP-NATIVE-GO-RENDERER-${revision}-${decision}`;
  const response = await request.post(
    `${apiURL}/v1/report-versions/${reportVersionId}/decisions`,
    {
      headers: {
        ...canonicalHeaders(subject, revision),
        "idempotency-key": operationId,
      },
      data: {
        operationId,
        reportVersionId,
        expectedReportVersionRevision: revision,
        decision,
        reason: `Exercise exact immutable report version through ${decision}.`,
      },
    },
  );
  expect(response.ok(), await response.text()).toBe(true);
}

test.beforeEach(async ({ request }) => {
  const response = await request.post(`${apiURL}/__test/reset`, {
    headers: { "x-avia-test-token": testToken },
  });
  expect(response.ok(), await response.text()).toBe(true);
});

test("native Go PDF is privately rendered and downloaded for the exact report version", async ({
  page,
  request,
}) => {
  test.setTimeout(90_000);
  await decideReport(request, "USR-MANAGER-NORA", 1, "FORWARD");
  await decideReport(request, "USR-GM-OMAR", 2, "FORWARD");
  await decideReport(request, "USR-ED-ZARA", 3, "ISSUE_AND_LOCK");

  await expect.poll(async () => {
    const response = await request.get(
      `${apiURL}/v1/documents/${reportVersionId}`,
      { headers: canonicalHeaders("USR-AUDITEE-FLY") },
    );
    if (!response.ok()) return `HTTP ${response.status()}`;
    return (await response.json() as { renderStatus?: string }).renderStatus;
  }, {
    timeout: 60_000,
    message: "native Go document worker should complete the render",
  }).toBe("SUCCEEDED");

  await page.goto("/auditee/documents");
  const documentsPage = page.getByTestId("auditee-documents-page");
  await expect(documentsPage).toContainText("Generated PDF");
  await expect(documentsPage).toContainText(
    "Rendering does not approve, sign, close, or confer legal validity",
  );
  await expect(documentsPage).not.toContainText("digitally signed");
  const button = documentsPage.getByRole("button", {
    name: `Download ${reportVersionId}`,
  });
  await expect(button).toBeEnabled();

  const [download] = await Promise.all([
    page.waitForEvent("download"),
    button.click(),
  ]);
  const path = await download.path();
  expect(path).not.toBeNull();
  const body = await readFile(path!);
  expect(body.subarray(0, 5).toString()).toBe("%PDF-");
  expect(download.suggestedFilename()).toMatch(/RPT-CAB-2026-001.*\.pdf$/);
  await expect(documentsPage.getByRole("status")).toContainText(
    "authorized immutable generated Document",
  );
});
