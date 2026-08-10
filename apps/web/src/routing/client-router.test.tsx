// @vitest-environment jsdom
import "@testing-library/jest-dom/vitest";

import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { MemoryRouter, Route, Routes, useParams } from "./client-router";

afterEach(cleanup);

function AuditIdentityProbe() {
  const { auditId } = useParams<{ auditId: string }>();
  return <output aria-label="Decoded Audit identity">{auditId}</output>;
}

describe("client router parameters", () => {
  it("decodes an encoded server-owned Audit identity before exposing it to the route", () => {
    render(
      <MemoryRouter initialEntries={["/inspector/audits/inspection%3Aassignment%3Aplan-intake-42"]}>
        <Routes>
          <Route path="/inspector/audits/:auditId" element={<AuditIdentityProbe />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByRole("status", { name: "Decoded Audit identity" })).toHaveTextContent(
      "inspection:assignment:plan-intake-42",
    );
  });
});
