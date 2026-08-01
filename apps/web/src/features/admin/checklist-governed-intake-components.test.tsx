// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ChecklistIntakePanel } from "./checklist-intake-panel";
import { ChecklistCandidateReview } from "./checklist-candidate-review";
import { ChecklistPublicationBlockers } from "./checklist-publication-blockers";

describe("governed checklist intake projections", () => {
  it("labels archive work candidate-only and preserves disabled reason", () => {
    render(<ChecklistIntakePanel batch={null} disabledReason="blocked: private parser receipt required" />);
    expect(screen.getByText(/candidate-only inventory/i)).toBeInTheDocument();
    expect(screen.getByRole("button")).toBeDisabled();
    expect(screen.getByRole("status")).toHaveTextContent("blocked");
  });

  it("does not present candidate origin as regulatory authority", () => {
    render(<ChecklistCandidateReview candidate={{ existingCandidateId: "CAND", candidateRootId: "ROOT", revision: 1, contentDigest: "sha256:c", origin: "EXISTING_CHECKLIST_CANDIDATE", questions: [{ questionId: "Q1", ordinal: 1, wording: "Question" }], candidateCurrentness: "CURRENT" }} />);
    expect(screen.getByText("EXISTING_CHECKLIST_CANDIDATE")).toBeInTheDocument();
    expect(screen.getByRole("note")).toHaveTextContent(/not regulatory source authority/i);
  });

  it("renders ordered blockers rather than enabling publication", () => {
    render(<ChecklistPublicationBlockers issues={[{ fieldPath: "regulatoryTrace", code: "SOURCE_MAPPING_REQUIRED", message: "mapping required", sourceIdentity: null, sourceHash: null, clauseId: null, locator: null }]} />);
    expect(screen.getByText(/SOURCE_MAPPING_REQUIRED/)).toBeInTheDocument();
  });
});
