// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { it, expect } from "vitest";
import { ChecklistDraftEditor } from "./checklist-draft-editor";
import type { GovernedCandidateView } from "../../backend/backend";
it("renders source-mapping blockers as non-approval state", () => {
  const draft = { candidateId: "C", candidateRootId: "R", revision: 1, contentDigest: "sha256:c", status: "GENERATED_DRAFT", questions: [{ questionId: "Q", prompt: "question", regulatoryTrace: { state: "SOURCE_MAPPING_REQUIRED" } }], lineage: { lineageType: "EXISTING_CANDIDATE" } } as unknown as GovernedCandidateView;
  render(<ChecklistDraftEditor draft={draft} />);
  expect(screen.getByRole("alert")).toHaveTextContent("SOURCE_MAPPING_REQUIRED");
});
