import { Suspense, type ReactElement } from "react";
import { Navigate, Route, Routes, useLocation, useNavigate } from "react-router-dom";

import { useApplicationRuntime } from "./providers";
import { CANONICAL_AUDIT_PREPARATION_PATH, CANONICAL_INSPECTOR_AUDIT_PATH, CANONICAL_INSPECTOR_CHECKLIST_PATH, CANONICAL_QUESTION_REVIEW_PATH, REACT_ROUTE_CONTRACT_BY_ID, REACT_ROUTE_CONTRACTS, type BuildProfileAvailability, type ReactSurfaceId, type RouteContract } from "./route-contracts";
import { SCREEN_COMPONENT_REGISTRY } from "./screen-component-registry";
import { RouteLoadBoundary, RouteLoadingState } from "./route-loader";
import type { Role } from "../backend/backend";
import { RoleGuard } from "../auth/role-guard";
import { QuestionReviewPage } from "../features/checklists/question-review-page";
import { AuditAssignmentPage } from "../features/teams/audit-assignment-page";
import { AuditDetailPage } from "../features/inspections/audit-detail-page";
import { ChecklistRunnerPage } from "../features/checklists/checklist-runner-page";
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
  return <Navigate replace to={createRoleEntryPath(role)} />;
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
  return <RoleGuard requiredRole={contract.requiredRole}>{!isAvailable ? <BlockedProfileRoute contract={contract} /> : entry.status === "implemented" ? <entry.component /> : <PendingImplementationRoute contract={contract} />}</RoleGuard>;
}

function routeElement(contract: RouteContract): ReactElement {
  return <ContractRoute contract={contract} />;
}

export function AppRouter() {
  const location = useLocation();
  const resetKey = `${location.pathname}${location.search}${location.hash}`;
  return (
    <RouteLoadBoundary resetKey={resetKey}>
      <Suspense fallback={<RouteLoadingState />}>
        <Routes>
          <Route path="/" element={<RoleSelectRoute />} />
          <Route path={CANONICAL_QUESTION_REVIEW_PATH} element={<RoleGuard requiredRole="manager"><QuestionReviewPage /></RoleGuard>} />
          <Route path={CANONICAL_AUDIT_PREPARATION_PATH} element={<RoleGuard requiredRole="leadInspector"><AuditAssignmentPage /></RoleGuard>} />
          <Route path={CANONICAL_INSPECTOR_CHECKLIST_PATH} element={<RoleGuard requiredRole="inspector"><ChecklistRunnerPage /></RoleGuard>} />
          <Route path={CANONICAL_INSPECTOR_AUDIT_PATH} element={<RoleGuard requiredRole="inspector"><AuditDetailPage /></RoleGuard>} />
          {REACT_ROUTE_CONTRACTS.filter((contract) => contract.id !== "role-select").map((contract) => <Route key={contract.id} path={contract.path} element={routeElement(contract)} />)}
          <Route path="*" element={<Navigate replace to="/" />} />
        </Routes>
      </Suspense>
    </RouteLoadBoundary>
  );
}
