import { Fragment, useEffect, useState, type ReactElement } from "react";
import { Link, Route, useLocation } from "react-router-dom";

import { AGADemoWorkspaceGuard } from "../auth/aga-demo-workspace-guard";
import { useApplicationRuntime } from "./providers";
import type { Role } from "../backend/backend";
import { AGADemoClassificationWorkspacePage } from "../features/checklists/aga-classification-workspace-page";
import { AGADemoCAPEvidencePage } from "../features/caps/aga-demo-cap-evidence-page";
import { AGADemoPotentialFindingPage } from "../features/findings/aga-demo-potential-finding-page";
import { AGADemoInspectionPage, type AGADemoLifecycleProjection } from "../features/inspections/aga-demo-inspection-page";
import { AGADemoInspectionPackagePage } from "../features/inspections/aga-demo-inspection-package-page";

export const AGA_DEMO_WORKSPACE_ROUTES = [
  { path: "/admin/aga-demo-workspace", role: "admin", label: "Admin Preview" },
  { path: "/department-manager/aga-demo-workspace", role: "manager", label: "Department Manager" },
  { path: "/inspector/aga-demo-workspace", role: "inspector", label: "CAA Inspector" },
  { path: "/lead-inspector/aga-demo-workspace", role: "leadInspector", label: "Lead Inspector" },
  { path: "/auditee/aga-demo-workspace", role: "auditee", label: "Auditee — Fly Namibia" },
] as const satisfies readonly { path: string; role: Role; label: string }[];

export function agaDemoWorkspaceLandingPath(role: Role): string | null {
  switch (role) {
    case "admin": return "/admin/aga-demo-workspace";
    case "manager": return "/department-manager/aga-demo-workspace";
    case "inspector": return "/inspector/aga-demo-workspace/inspection";
    case "leadInspector": return "/lead-inspector/aga-demo-workspace/inspection";
    case "auditee": return "/auditee/aga-demo-workspace/caps-evidence";
    default: return null;
  }
}

function WorkspaceRoute({ path, role, label }: { path: string; role: Role; label: string }) {
  const location = useLocation();
  const [projection, setProjection] = useState<AGADemoLifecycleProjection | null>(null);
  const suffix = location.pathname.slice(path.length).replace(/\/$/u, "");
  const knownLifecycleSuffix = suffix === "" || suffix === "/inspection" || suffix === "/potential-findings" || suffix === "/caps-evidence";
  const auditeeSuffixAllowed = suffix === "" || suffix === "/caps-evidence";
  if (!knownLifecycleSuffix || (role === "auditee" && !auditeeSuffixAllowed)) {
    return <WorkspaceRouteNotFound />;
  }
  const page = role === "auditee"
    ? "caps-evidence"
    : suffix === "/inspection"
    ? "inspection"
    : suffix === "/potential-findings"
      ? "potential-findings"
      : suffix === "/caps-evidence"
        ? "caps-evidence"
        : "classification";
  return (
    <AGADemoWorkspaceGuard requiredRole={role}>
      {(capability) => (
        <>
          <nav aria-label="AGA synthetic lifecycle routes" className="aga-workspace-route-nav">
            {role === "auditee" ? <Link aria-current="page" to={`${path}/caps-evidence`}>CAP and Evidence</Link> : <>
              <Link aria-current={page === "classification" ? "page" : undefined} to={path}>Classification workspace</Link>
              {role === "manager" ? <Link className="aga-workspace-route-nav__setup" aria-current={location.pathname === "/department-manager/aga-demo-workspace/inspection-package" ? "page" : undefined} to="/department-manager/aga-demo-workspace/inspection-package">Package builder</Link> : null}
              <Link aria-current={page === "inspection" ? "page" : undefined} to={`${path}/inspection`}>Inspection lifecycle</Link>
              <Link aria-current={page === "potential-findings" ? "page" : undefined} to={`${path}/potential-findings`}>Potential Findings</Link>
              <Link aria-current={page === "caps-evidence" ? "page" : undefined} to={`${path}/caps-evidence`}>CAP and Evidence</Link>
            </>}
          </nav>
          {page === "inspection" ? <AGADemoInspectionPage capability={capability} role={role} roleLabel={label} initialProjection={projection ?? undefined} onProjectionChange={setProjection} /> : null}
          {page === "potential-findings" ? <AGADemoPotentialFindingPage capability={capability} role={role} roleLabel={label} initialProjection={projection ?? undefined} onProjectionChange={setProjection} /> : null}
          {page === "caps-evidence" ? <AGADemoCAPEvidencePage capability={capability} role={role} roleLabel={label} initialProjection={projection ?? undefined} onProjectionChange={setProjection} /> : null}
          {page === "classification" ? <AGADemoClassificationWorkspacePage capability={capability} role={role} roleLabel={label} /> : null}
        </>
      )}
    </AGADemoWorkspaceGuard>
  );
}

function WorkspaceRouteNotFound() {
  return (
    <main className="aga-lifecycle-page" data-testid="aga-demo-workspace-not-found">
      <h1>AGA demo workspace route unavailable</h1>
      <p>This lifecycle route is not available for the selected role.</p>
    </main>
  );
}

export const agaDemoWorkspaceRouteElements: readonly ReactElement[] = AGA_DEMO_WORKSPACE_ROUTES.map(
  ({ path, role, label }) => <Route key={path} path={`${path}/:lifecycleView?`} element={<WorkspaceRoute path={path} role={role} label={label} />} />,
);

const managerPackageRoute = (
  <Route
    key="/department-manager/aga-demo-workspace/inspection-package"
    path="/department-manager/aga-demo-workspace/inspection-package"
    element={(
      <AGADemoWorkspaceGuard requiredRole="manager">
        {(capability) => <AGADemoInspectionPackagePage capability={capability} role="manager" roleLabel="Department Manager" />}
      </AGADemoWorkspaceGuard>
    )}
  />
);

// The connected fixture intentionally reuses the Lead Inspector session for
// its separately bound CAA_REVIEWER membership. Keep the presentation route
// explicit while preserving the server-side binding distinction.
const caaReviewerRoute = (
  <Route
    key="/caa-reviewer/aga-demo-workspace"
    path="/caa-reviewer/aga-demo-workspace/:lifecycleView?"
    element={<WorkspaceRoute path="/caa-reviewer/aga-demo-workspace" role="leadInspector" label="CAA Reviewer" />}
  />
);

export const agaDemoWorkspaceRouteElementsWithManagerPackage: readonly ReactElement[] = [
  managerPackageRoute,
  ...agaDemoWorkspaceRouteElements,
  caaReviewerRoute,
  ...[...AGA_DEMO_WORKSPACE_ROUTES.map(({ path }) => path), "/caa-reviewer/aga-demo-workspace"].map(
    (path) => <Route key={`${path}:not-found`} path={`${path}/*`} element={<WorkspaceRouteNotFound />} />,
  ),
];

export function AGADemoWorkspaceNavigation({
  activeRole,
  onNavigate,
}: {
  activeRole: Role;
  onNavigate?: () => void;
}) {
  const runtime = useApplicationRuntime();
  const location = useLocation();
  const client = runtime.backend.agaDemoWorkspace;
  const route = AGA_DEMO_WORKSPACE_ROUTES.find((candidate) => candidate.role === activeRole);
  const [available, setAvailable] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    if (!client || !route) {
      setAvailable(false);
      return () => controller.abort();
    }
    void client.capability({ signal: controller.signal })
      .then((capability) => {
        if (!controller.signal.aborted) setAvailable(capability.available);
      })
      .catch(() => {
        if (!controller.signal.aborted) setAvailable(false);
      });
    return () => controller.abort();
  }, [client, route]);

  if (!route || !available) return null;
  const active = location.pathname === route.path || location.pathname.startsWith(`${route.path}/`);
  return (
    <Link
      aria-current={active ? "page" : undefined}
      aria-label="AGA demo workspace"
      className={`nav-item${active ? " active" : ""}`}
      onClick={onNavigate}
      to={route.path}
    >
      <span className="nav-item__icon" aria-hidden="true"><svg viewBox="0 0 24 24"><path d="M4 5h16v14H4z" /><path d="M8 9h8" /><path d="M8 13h5" /></svg></span>
      <span>AGA demo workspace</span>
    </Link>
  );
}

/** Exported for focused route tests; HTTP bootstrap passes the route elements directly. */
export function AGADemoWorkspaceRoutes() {
  return <Fragment>{agaDemoWorkspaceRouteElementsWithManagerPackage}</Fragment>;
}
