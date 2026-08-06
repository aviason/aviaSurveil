import type { ReactNode } from "react";
import { useLocation } from "react-router-dom";

import { LoginPage } from "./login-page";
import { useSession } from "./session-provider";
import type { SessionState } from "./session-client";

export function HttpAuthGate({
  children,
}: {
  children(state: Extract<SessionState, { status: "authenticated" }>): ReactNode;
}) {
  const location = useLocation();
  const session = useSession();
  const returnTo =
    location.pathname === "/"
      ? "/"
      : `${location.pathname}${location.search}`;

  if (session.state.status === "authenticated") {
    return <>{children(session.state)}</>;
  }
  if (session.state.status === "unavailable") {
    return (
      <LoginPage
        message={session.state.message}
        onLogin={() => session.login(returnTo)}
      />
    );
  }
  if (session.state.status === "expired") {
    return (
      <LoginPage
        message="Your session expired. Sign in again to continue."
        onLogin={() => session.login(returnTo)}
      />
    );
  }
  return <LoginPage onLogin={() => session.login(returnTo)} />;
}
