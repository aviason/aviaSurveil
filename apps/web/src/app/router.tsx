import { Suspense, type ReactElement } from "react";
import { Navigate, Route, Routes, useNavigate } from "react-router-dom";

import { agaDemoWorkspaceLandingPath } from "./aga-demo-workspace-routes";
import { useApplicationRuntime } from "./providers";
import { CANONICAL_QUESTION_REVIEW_PATH, REACT_ROUTE_CONTRACT_BY_ID, REACT_ROUTE_CONTRACTS, type BuildProfileAvailability, type ReactSurfaceId, type RouteContract } from "./route-contracts";
import { SCREEN_COMPONENT_REGISTRY } from "./screen-component-registry";
import type { Role } from "../backend/backend";
import { RoleGuard } from "../auth/role-guard";
import { QuestionReviewPage } from "../features/checklists/question-review-page";
import { useOptionalSession } from "../auth/session-provider";
import { RoleSelectPage, ROLE_ENTRIES, createRoleEntryPath } from "../ui/role-select-page";

export { ROLE_ENTRIES, createRoleEntryPath } from "../ui/role-select-page";

function RoleSelectRoute() {
  const { buildProfile, identityMode } = useApplicationRuntime();
  const session = useOptionalSession();
  const navigate = useNavigate();
  if (identityMode === "oidc-session" && session?.state.status === "authenticated") {
    return <AuthenticatedLandingRoute role={session.state.activeRole} />;
  }
  return <RoleSelectPage mode={identityMode ?? (buildProfile === "http" ? "canonical-test-role-switch" : "demo-role-switch")} onRoleRequest={(role) => navigate(createRoleEntryPath(role))} />;
}

function AuthenticatedLandingRoute({ role }: { role: Role }) {
  const runtime = useApplicationRuntime();
  const agaLandingPath = runtime.agaDemoWorkspaceSurfaceEnabled
    ? agaDemoWorkspaceLandingPath(role)
    : null;
  return <Navigate replace to={agaLandingPath ?? createRoleEntryPath(role)} />;
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

function AgaDemoOnlyRoute({ contract }: { contract: RouteContract }) {
  const target = contract.requiredRole ? agaDemoWorkspaceLandingPath(contract.requiredRole) : null;
  if (target) return <Navigate replace to={target} />;
  return (
    <section data-testid="aga-demo-only-route" role="status">
      <h1>{contract.label}</h1>
      <p>This local candidate exposes the connected AGA demo workspace only. The canonical {contract.label.toLowerCase()} surface is not provisioned by this API profile.</p>
    </section>
  );
}

function ContractRoute({ contract }: { contract: RouteContract }) {
  const { buildProfile, agaDemoWorkspaceSurfaceEnabled } = useApplicationRuntime();
  const entry = SCREEN_COMPONENT_REGISTRY[contract.componentKey];
  const isAvailable = contract.availableProfiles.includes(buildProfile);
  const agaDemoOnly = agaDemoWorkspaceSurfaceEnabled && contract.id !== "admin-checklist-builder";
  return <RoleGuard requiredRole={contract.requiredRole}><Suspense fallback={<p data-testid="route-loading">Loading route...</p>}>{agaDemoOnly ? <AgaDemoOnlyRoute contract={contract} /> : !isAvailable ? <BlockedProfileRoute contract={contract} /> : entry.status === "implemented" ? <entry.component /> : <PendingImplementationRoute contract={contract} />}</Suspense></RoleGuard>;
}

function routeElement(contract: RouteContract): ReactElement {
  return <ContractRoute contract={contract} />;
}

export function AppRouter() {
  const { supplementalRouteElements } = useApplicationRuntime();
  return <Routes>
    <Route path="/" element={<RoleSelectRoute />} />
    <Route path={CANONICAL_QUESTION_REVIEW_PATH} element={<RoleGuard requiredRole="manager"><QuestionReviewPage /></RoleGuard>} />
    {supplementalRouteElements ?? null}
    {REACT_ROUTE_CONTRACTS.filter((contract) => contract.id !== "role-select").map((contract) => <Route key={contract.id} path={contract.path} element={routeElement(contract)} />)}
    <Route path="*" element={<Navigate replace to="/" />} />
  </Routes>;
}
