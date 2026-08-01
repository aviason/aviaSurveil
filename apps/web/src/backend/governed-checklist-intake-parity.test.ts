import { describe, expect, it, vi } from "vitest";
import { createHttpBackend } from "./http-backend";

describe("governed checklist intake transport", () => {
  it("uses one multipart archive and one JSON receipt without leaking a durable object key", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(new Response(JSON.stringify({
      batch: { importBatchId: "IMPORT-1", expectedArchiveSha256: "sha256:archive", status: "RECEIVED", manifestDigest: null, fileCount: 0, registerCount: 0, blockingIssues: ["PARSER_PENDING"] }, replayed: false,
    }), { status: 201, headers: { "content-type": "application/json" } }));
    const backend = createHttpBackend({ apiBaseUrl: "/", environmentLabel: "test" }, { fetchImplementation, csrfToken: () => "csrf" });
    await backend.governedChecklistIntake.receiveBatch({ operationId: "OP-1", idempotencyKey: "IDEM-1", expectedArchiveSha256: "sha256:archive", reason: "candidate-only", archive: new Uint8Array([80, 75, 3, 4]) });
    const [url, init] = fetchImplementation.mock.calls[0]!;
    expect(url).toBe("/v1/admin/governed-checklist/import-batches");
    expect(init?.method).toBe("POST");
    expect(init?.body).toBeInstanceOf(FormData);
    const form = init?.body as FormData;
    expect(form.get("archive")).toBeInstanceOf(Blob);
    expect(JSON.parse(await (form.get("receipt") as Blob).text())).toEqual({ operationId: "OP-1", idempotencyKey: "IDEM-1", expectedArchiveSha256: "sha256:archive", reason: "candidate-only" });
    expect(new Headers(init?.headers).get("x-csrf-token")).toBe("csrf");
  });

  it("keeps scoped reviewer queue path distinct from Admin inventory", async () => {
    const fetchImplementation = vi.fn<typeof fetch>().mockResolvedValue(new Response(JSON.stringify({ items: [], nextCursor: null }), { status: 200, headers: { "content-type": "application/json" } }));
    const backend = createHttpBackend({ apiBaseUrl: "/", environmentLabel: "test" }, { fetchImplementation });
    await backend.governedChecklistIntake.listReviewerQueue({});
    expect(fetchImplementation.mock.calls[0]![0]).toBe("/v1/governed-checklist/reviewer-queue");
  });
});
