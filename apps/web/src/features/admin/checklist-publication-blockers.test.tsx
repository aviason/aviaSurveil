// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { it, expect } from "vitest";
import { ChecklistPublicationBlockers } from "./checklist-publication-blockers";
it("renders blockers without an approval control", () => {
  render(<ChecklistPublicationBlockers issues={[{ fieldPath: "trace", code: "SOURCE_MAPPING_REQUIRED", message: "mapping required", sourceIdentity: null, sourceHash: null, clauseId: null, locator: null }]} />);
  expect(screen.getByText(/SOURCE_MAPPING_REQUIRED/)).toBeInTheDocument();
});
