import type { PropsWithChildren } from "react";

import type { Role } from "../backend/backend";
import { roleLabel } from "../ui/role-select-page";
import type { IdentityMode, SessionState } from "./session-client";

export function RoleHandoff({
  identityMode,
  session,
  targetRole,
  onRoleRequest,
  children,
}: PropsWithChildren<{
  identityMode: IdentityMode;
  session: SessionState;
  targetRole: Role;
  onRoleRequest(role: Role): void;
}>) {
  const label = typeof children === "string" ? children : roleLabel(targetRole);
  if (identityMode !== "oidc-session") {
    return (
      <button
        className="secondary-button"
        data-action-boundary="navigation"
        onClick={() => onRoleRequest(targetRole)}
        type="button"
      >
        {children}
      </button>
    );
  }
  const allowed = session.status === "authenticated" && session.session.roles.includes(targetRole);
  return (
    <span>
      <button
        className="secondary-button"
        data-action-boundary="navigation"
        disabled={!allowed}
        onClick={() => allowed && onRoleRequest(targetRole)}
        type="button"
      >
        {children}
      </button>
      {!allowed ? (
        <span>
          Your session does not include {roleLabel(targetRole)} authority.
        </span>
      ) : (
        <span className="visually-hidden">{label}</span>
      )}
    </span>
  );
}
