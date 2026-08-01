// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { it, expect } from "vitest";
import { ChecklistReviewerQueue } from "./checklist-reviewer-queue";
import type { GovernedReviewerQueuePage } from "../../backend/backend";
it("labels reviewer comments as non-binding", () => {
  render(<ChecklistReviewerQueue page={{ items: [], nextCursor: null } as GovernedReviewerQueuePage} />);
  expect(screen.getByText(/never become technical approval/i)).toBeInTheDocument();
});
