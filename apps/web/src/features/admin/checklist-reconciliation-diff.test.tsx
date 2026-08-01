// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { it, expect } from "vitest";
import { ChecklistReconciliationDiff } from "./checklist-reconciliation-diff";
import type { GovernedQuestionReconciliationView } from "../../backend/backend";
it("shows immutable wording and scope differences", () => {
  const items = [{ legacyQuestionId: "Q1", legacyWording: "old", currentWording: "new", wordingChanged: true, legacyApplicability: "old", currentApplicability: "new", applicabilityChanged: true, legacyScopeClassification: "old", currentScopeClassification: "new", scopeChanged: true }] as GovernedQuestionReconciliationView[];
  render(<ChecklistReconciliationDiff items={items} />);
  expect(screen.getAllByText("old → new")).toHaveLength(3);
});
