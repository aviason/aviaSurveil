import type { RoleSelectionMode } from "./role-select-page";

export function CandidateBoundary({
  mode,
  environmentLabel,
}: {
  mode: RoleSelectionMode;
  environmentLabel: string;
}) {
  if (mode === "oidc-session") {
    return <p className="release-boundary"><span>Release-bound authenticated session</span><span aria-hidden="true">·</span><span>{environmentLabel}</span></p>;
  }
  const modeLabel = "Local qualification data";
  return (
    <p className="qualification-boundary">
      <span>Qualification profile</span>
      <span>{modeLabel}</span>
      <span>Release evidence required</span>
    </p>
  );
}
