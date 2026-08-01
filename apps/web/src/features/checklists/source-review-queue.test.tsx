// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { it, expect } from "vitest";
import { SourceReviewQueue } from "./source-review-queue";
import type { GovernedSourceReviewQueuePage } from "../../backend/backend";
it("labels source review as assignment scoped", () => {
  render(<SourceReviewQueue page={{ items: [], nextCursor: null } as GovernedSourceReviewQueuePage} />);
  expect(screen.getByText(/current source-owner assignment scope/i)).toBeInTheDocument();
});
