// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { AGACandidateDemoBackend } from "../../backend/aga-candidate-demo";
import { AGACandidateDemoPanel } from "./aga-candidate-demo-panel";

function backend(): AGACandidateDemoBackend {
  return {
    capability: async () => ({ available: true, labels: ["candidate-only", "release pending", "production-ready: not established"] }),
    summary: async () => ({ packageDigest: "sha256:sealed", formCount: 52, questionCount: 1310, sourceRequirements: ["EXACT_SOURCE_BYTES", "EXACT_SOURCE_BYTES_SHA256", "EFFECTIVE_DATE", "CLAUSE_OR_PAGE_LOCATOR", "APPLICABILITY", "NAMED_SOURCE_OWNER_ATTESTATION"], labels: ["candidate-only", "release pending", "production-ready: not established"] }),
    listForms: async () => ({ items: [
      { code: "FSS-AGA-FORM-001", title: "Synthetic form", questionCount: 0, questionExtractionState: "NO_PROTOCOL_QUESTION_BOUNDARY_DETECTED" },
      { code: "FSS-AGA-FORM-002", title: "Candidate form", questionCount: 3, questionExtractionState: "EXTRACTED_CANDIDATE_BOUNDARIES" },
    ], nextCursor: null }),
    getForm: async () => ({ code: "FSS-AGA-FORM-001", title: "Synthetic form", questionCount: 0, questionExtractionState: "NO_PROTOCOL_QUESTION_BOUNDARY_DETECTED" }),
    listQuestions: async () => ({ items: [
      { proposalId: "P-1", formCode: "FSS-AGA-FORM-002", ordinal: 1, text: "Synthetic candidate text", textDigest: "sha256:question", sourceGapCategory: "PROPOSAL_PRESENT_REVIEW_REQUIRED", riskBand: "PROPOSED_REVIEW_REQUIRED" },
      { proposalId: "P-2", formCode: "FSS-AGA-FORM-002", ordinal: 2, text: "Synthetic unmapped text", textDigest: "sha256:unmapped", sourceGapCategory: "UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL", riskBand: "PROPOSED_HIGH_OPERATIONAL" },
    ], nextCursor: null }),
  };
}

describe("AGA candidate demo panel", () => {
  it("does not render when the tagged capability is absent", () => {
    render(<AGACandidateDemoPanel capability={undefined} />);
    expect(screen.queryByTestId("aga-candidate-demo-panel")).not.toBeInTheDocument();
  });

  it("renders only read-only candidate boundaries from the capability", async () => {
    render(<AGACandidateDemoPanel capability={backend()} />);
    expect(await screen.findByRole("heading", { name: "AGA candidate demo" })).toBeInTheDocument();
    expect(screen.getByText("candidate-only")).toBeInTheDocument();
    expect(screen.getByText("production-ready: not established")).toBeInTheDocument();
    expect(screen.getByText("52")).toBeInTheDocument();
    expect(screen.getByText("1310")).toBeInTheDocument();
    expect(screen.getByText("EXACT SOURCE BYTES")).toBeInTheDocument();
    expect(screen.getByText(/no question-level source proposal is present/i)).toBeInTheDocument();
    expect(screen.getByText("Expert review blocker")).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("loads every sealed question page for the synthetic Department Manager demo handoff", async () => {
    const allQuestions = Array.from({ length: 1310 }, (_, index) => ({
      proposalId: `P-${index + 1}`,
      formCode: `FSS-AGA-FORM-${String((index % 52) + 1).padStart(3, "0")}`,
      ordinal: index + 1,
      text: `Candidate question ${index + 1}`,
      textDigest: `sha256:${String(index + 1).padStart(64, "0")}`,
      sourceGapCategory: "PROPOSAL_PRESENT_REVIEW_REQUIRED" as const,
      riskBand: "PROPOSED_HIGH_OPERATIONAL" as const,
    }));
    const cursors: Array<string | undefined> = [];
    const pagedBackend = {
      ...backend(),
      listQuestions: async (input: { cursor?: string; limit?: number }) => {
        cursors.push(input.cursor);
        const start = input.cursor ? Number(input.cursor) : 0;
        const items = allQuestions.slice(start, start + (input.limit ?? 100));
        const nextCursor = start + items.length < allQuestions.length ? String(start + items.length) : null;
        return { items, nextCursor };
      },
    } satisfies AGACandidateDemoBackend;

    render(<AGACandidateDemoPanel capability={pagedBackend} />);

    await waitFor(() => {
      const handoffs = screen.getAllByTestId("aga-candidate-manager-demo-handoff");
      expect(handoffs.some((candidate) => candidate.querySelectorAll('[data-testid="aga-candidate-manager-question-row"]').length === 1310)).toBe(true);
    });
    const handoffs = screen.getAllByTestId("aga-candidate-manager-demo-handoff");
    const handoff = handoffs.find((candidate) => candidate.querySelectorAll('[data-testid="aga-candidate-manager-question-row"]').length === 1310)!;
    expect(within(handoff).getByText("Synthetic Department Manager demo handoff")).toBeInTheDocument();
    expect(within(handoff).getAllByTestId("aga-candidate-manager-question-row")).toHaveLength(1310);
    expect(cursors).toHaveLength(14);
    expect(cursors).toContain("100");
    expect(cursors).toContain("1200");
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
