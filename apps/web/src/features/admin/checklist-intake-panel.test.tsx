// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";
import { render, screen } from "@testing-library/react";
import { it, expect } from "vitest";
import { ChecklistIntakePanel } from "./checklist-intake-panel";
it("shows a disabled reason when the external archive is unavailable", () => {
  render(<ChecklistIntakePanel batch={null} disabledReason="blocked: external archive" />);
  expect(screen.getByRole("button")).toBeDisabled();
  expect(screen.getByRole("status")).toHaveTextContent("blocked");
});
