// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";

import { AppProviders } from "../../app/providers";
import type { DocumentMetadataView } from "../../backend/backend";
import { createMockBackendRuntime } from "../../mock/create-mock-backend";
import { AuditeeDocumentsPage } from "./auditee-documents-page";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

function renderHTTPDocuments(documents: DocumentMetadataView[]) {
  const runtime = createMockBackendRuntime();
  const auditee = runtime.backendForRole("auditee");
  vi.spyOn(auditee.documents, "list").mockResolvedValue({
    items: documents,
    nextCursor: null,
  });
  vi.spyOn(auditee.documents, "open").mockImplementation(async ({ documentId }) => {
    const document = documents.find((candidate) => candidate.id === documentId);
    if (!document) throw new Error("Document not found");
    return structuredClone(document);
  });
  render(
    <AppProviders runtime={{
      backend: runtime.backend,
      backendForRole: runtime.backendForRole,
      buildProfile: "http",
      environmentLabel: "test",
      identityMode: "canonical-test-role-switch",
      subjectId: "USR-AUDITEE-FLY",
    }}>
      <MemoryRouter>
        <AuditeeDocumentsPage />
      </MemoryRouter>
    </AppProviders>,
  );
  return auditee;
}

describe("Auditee generated Document downloads", () => {
  it("uses only the authorized signed URL for a succeeded HTTP render", async () => {
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(
      function capture(this: HTMLAnchorElement) {
        expect(this.href).toBe(
          "https://localhost:8443/generated-documents/org/report.pdf?X-Amz-Signature=exact",
        );
        expect(this.download).toBe("RPT-CAB-2026-001-v1.pdf");
      },
    );
    const createObjectURL = vi.fn();
    Object.defineProperty(URL, "createObjectURL", {
      configurable: true,
      value: createObjectURL,
    });
    renderHTTPDocuments([{
      id: "RPT-CAB-2026-001-V1",
      organizationId: "ORG-FLY-NAMIBIA",
      title: "Report RPT-CAB-2026-001",
      kind: "REPORT",
      version: 1,
      revision: 4,
      createdAt: "2026-07-21T12:00:00Z",
      publicReviewResult: "RELEASED",
      downloadFileName: "RPT-CAB-2026-001-v1.pdf",
      renderStatus: "SUCCEEDED",
      documentVersionId: "render-job-001-version",
      downloadUrl:
        "https://localhost:8443/generated-documents/org/report.pdf?X-Amz-Signature=exact",
      downloadExpiresAt: "2026-07-21T12:05:00Z",
      sha256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      rendererHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
      templateHash: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
      sourceHash: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
    }]);

    const page = await screen.findByTestId("auditee-documents-page");
    expect(page).toHaveTextContent("Generated Report");
    expect(page).toHaveTextContent("does not approve, sign, close, or confer legal validity");
    await userEvent.click(
      within(page).getByRole("button", {
        name: "Download RPT-CAB-2026-001-V1",
      }),
    );
    expect(click).toHaveBeenCalledOnce();
    expect(createObjectURL).not.toHaveBeenCalled();
    expect(within(page).getByRole("status")).toHaveTextContent(
      "authorized immutable generated Document",
    );
  });

  it("keeps pending HTTP renders disabled without manufacturing a PDF", async () => {
    renderHTTPDocuments([{
      id: "RPT-PRELIMINARY-001-V1",
      organizationId: "ORG-FLY-NAMIBIA",
      title: "Preliminary Report",
      kind: "REPORT",
      version: 1,
      revision: 1,
      createdAt: "2026-07-21T12:00:00Z",
      publicReviewResult: "RELEASED",
      renderStatus: "PENDING",
    }]);

    const page = await screen.findByTestId("auditee-documents-page");
    const unavailable = within(page).getByRole("button", {
      name: "Download RPT-PRELIMINARY-001-V1 unavailable",
    });
    expect(unavailable).toBeDisabled();
    expect(unavailable).toHaveAttribute("title", expect.stringMatching(/pending/i));
    expect(page).not.toHaveTextContent("digitally signed");
  });
});
