// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { it, expect } from "vitest";
import { ChecklistCandidateReview } from "./checklist-candidate-review";
import type { ExistingChecklistCandidateView } from "../../backend/backend";
it("keeps existing-candidate origin separate from source authority", () => {
  const candidate = { existingCandidateId: "C", candidateRootId: "R", revision: 1, contentDigest: "sha256:c", origin: "EXISTING_CHECKLIST_CANDIDATE", questions: [{ questionId: "Q", ordinal: 1, wording: "candidate" }], candidateCurrentness: "CURRENT" } as ExistingChecklistCandidateView;
  render(<ChecklistCandidateReview candidate={candidate} />);
  expect(screen.getByRole("note")).toHaveTextContent(/not regulatory source authority/i);
});
