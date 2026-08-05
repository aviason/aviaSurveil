import { Suspense, useEffect, useRef, useState, type ReactElement } from "react";
import { Navigate, Route, Routes, useNavigate } from "react-router-dom";

import { useApplicationRuntime } from "./providers";
import { REACT_ROUTE_CONTRACT_BY_ID, REACT_ROUTE_CONTRACTS, type BuildProfileAvailability, type ReactSurfaceId, type RouteContract } from "./route-contracts";
import { SCREEN_COMPONENT_REGISTRY } from "./screen-component-registry";
import type { Role } from "../backend/backend";
import { LoginPage } from "../auth/login-page";
import { RoleGuard } from "../auth/role-guard";
import { useOptionalSession } from "../auth/session-provider";
import { RoleSelectPage, ROLE_ENTRIES, createRoleEntryPath } from "../ui/role-select-page";

export { ROLE_ENTRIES, createRoleEntryPath } from "../ui/role-select-page";

function RoleSelectRoute() {
  const { buildProfile, identityMode } = useApplicationRuntime();
  const session = useOptionalSession();
  const navigate = useNavigate();
  if (identityMode === "oidc-session" && session?.state.status === "authenticated") return <OidcLogoutRoute />;
  return <RoleSelectPage mode={identityMode ?? (buildProfile === "http" ? "canonical-test-role-switch" : "demo-role-switch")} onRoleRequest={(role) => navigate(createRoleEntryPath(role))} />;
}

function OidcLogoutRoute() {
  const runtime = useApplicationRuntime();
  const session = useOptionalSession();
  const logoutStarted = useRef(false);
  const [message, setMessage] = useState("Signing out…");
  useEffect(() => {
    if (session?.state.status !== "authenticated" || logoutStarted.current) return;
    logoutStarted.current = true;
    void (async () => {
      try {
        await runtime.beforeSubjectChange?.("LOGOUT");
        await session.logout();
        setMessage("You have signed out.");
      } catch {
        setMessage("Logout could not be completed. Your authenticated session remains active.");
      }
    })();
  }, [runtime, session]);
  return <LoginPage message={message} onLogin={() => session?.login("/inspector/inspector-assignments")} />;
}

const roleHomeSurfaceId: Record<Role, ReactSurfaceId> = {
  inspector: "inspector-home",
  leadInspector: "lead-home",
  manager: "manager-home",
  gm: "gm-home",
  finance: "finance-home",
  executiveDirector: "executive-home",
  auditee: "auditee-home",
  admin: "admin-home",
};

function nearestImplementedParent(contract: RouteContract, buildProfile: BuildProfileAvailability) {
  const candidates = [contract.parentId, contract.requiredRole ? roleHomeSurfaceId[contract.requiredRole] : null];
  for (const candidate of candidates) {
    let currentId = candidate;
    while (currentId) {
      const current = REACT_ROUTE_CONTRACT_BY_ID.get(currentId);
      if (!current) break;
      const entry = SCREEN_COMPONENT_REGISTRY[current.componentKey];
      if (entry.status === "implemented" && current.availableProfiles.includes(buildProfile)) {
        return entry.component;
      }
      currentId = current.parentId;
    }
  }
  return null;
}

function ParentSurface({ contract }: { contract: RouteContract }) {
  const { buildProfile } = useApplicationRuntime();
  const ParentScreen = nearestImplementedParent(contract, buildProfile);
  return ParentScreen ? <ParentScreen /> : null;
}

function BlockedProfileRoute({ contract }: { contract: RouteContract }) {
  return <><ParentSurface contract={contract} /><section role="alert" aria-label="Unavailable HTTP capability">{contract.blockedProfileReason}</section></>;
}

function PendingImplementationRoute({ contract }: { contract: RouteContract }) {
  return <><ParentSurface contract={contract} /><section role="alert" aria-label={`${contract.label} pending implementation`} data-testid="route-pending-implementation"><strong>{contract.label}</strong> is pending implementation in this demo route contract.</section></>;
}

function ContractRoute({ contract }: { contract: RouteContract }) {
  const { buildProfile } = useApplicationRuntime();
  const entry = SCREEN_COMPONENT_REGISTRY[contract.componentKey];
  const isAvailable = contract.availableProfiles.includes(buildProfile);
  return <RoleGuard requiredRole={contract.requiredRole}><Suspense fallback={<p data-testid="route-loading">Loading route...</p>}>{!isAvailable ? <BlockedProfileRoute contract={contract} /> : entry.status === "implemented" ? <entry.component /> : <PendingImplementationRoute contract={contract} />}</Suspense></RoleGuard>;
}

function routeElement(contract: RouteContract): ReactElement {
  return <ContractRoute contract={contract} />;
}

export function AppRouter() {
  const { supplementalRouteElements } = useApplicationRuntime();
  return <Routes>
    <Route path="/" element={<RoleSelectRoute />} />
    {supplementalRouteElements ?? null}
    {REACT_ROUTE_CONTRACTS.filter((contract) => contract.id !== "role-select").map((contract) => <Route key={contract.id} path={contract.path} element={routeElement(contract)} />)}
    <Route path="*" element={<Navigate replace to="/" />} />
  </Routes>;
}
