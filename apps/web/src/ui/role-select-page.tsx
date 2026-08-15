import { Link } from "react-router-dom";

import type { Role } from "../backend/backend";
import type { IconKey, ReactSurfaceId } from "../app/route-contracts";
import { BRAND_ASSETS } from "./brand-assets";

export interface RoleEntry {
  role: Role;
  routeId: ReactSurfaceId;
  route: string;
  slug: string;
  label: string;
  purpose: string;
  iconKey: IconKey;
  loginLabel?: string;
  loginPurpose?: string;
  loginIconKey?: IconKey;
}

export const ROLE_ENTRIES: readonly RoleEntry[] = [
  {
    role: "inspector",
    routeId: "inspector-home",
    route: "inspector-assignments",
    slug: "inspector",
    label: "CAA Inspector",
    purpose: "What do I need to inspect or review today?",
    iconKey: "assignments",
    loginLabel: "Inspector Workspace",
    loginPurpose: "Daily inspections, findings and field work",
  },
  {
    role: "leadInspector",
    routeId: "lead-home",
    route: "lead-review",
    slug: "lead-inspector",
    label: "Lead Inspector",
    purpose: "Which submitted checklist or Potential Finding needs my decision?",
    iconKey: "leadReview",
    loginPurpose: "Assigned audits and report sign-off",
    loginIconKey: "dashboard",
  },
  {
    role: "manager",
    routeId: "manager-home",
    route: "dashboard",
    slug: "department-manager",
    label: "Department Manager",
    purpose: "Where is my department exposed, delayed, or overloaded?",
    iconKey: "dashboard",
    loginPurpose: "Planning, approvals and department oversight",
    loginIconKey: "organizations",
  },
  {
    role: "gm",
    routeId: "gm-home",
    route: "gm-dashboard",
    slug: "general-manager",
    label: "General Manager",
    purpose: "Which cross-department review needs an intermediate decision?",
    iconKey: "dashboard",
    loginPurpose: "Cross-department review and forwarding",
    loginIconKey: "reports",
  },
  {
    role: "finance",
    routeId: "finance-home",
    route: "finance-review",
    slug: "finance",
    label: "Finance Review",
    purpose: "Which budget-required surveillance plan needs review?",
    iconKey: "finance",
    loginPurpose: "Budget and resource review",
  },
  {
    role: "executiveDirector",
    routeId: "executive-home",
    route: "executive-dashboard",
    slug: "executive-director",
    label: "Executive Director",
    purpose: "Which eligible plan or report needs a final decision?",
    iconKey: "reports",
    loginPurpose: "Final plan and report approval",
    loginIconKey: "leadReview",
  },
  {
    role: "auditee",
    routeId: "auditee-home",
    route: "service-provider-cap",
    slug: "auditee",
    label: "Auditee",
    purpose: "What does the CAA need from my organization?",
    iconKey: "assignments",
    loginLabel: "Service Provider Portal",
    loginPurpose: "Inspection dates, CAP, Evidence and CAA requests",
    loginIconKey: "profile",
  },
  {
    role: "admin",
    routeId: "admin-home",
    route: "templates",
    slug: "admin",
    label: "Admin Preview",
    purpose: "Which configured template or rule is available for preview?",
    iconKey: "templates",
    loginLabel: "Administration",
    loginPurpose: "Templates, rules and access",
  },
] as const;

export type RoleSelectionMode = "demo-role-switch" | "canonical-test-role-switch" | "oidc-session";

export function createRoleEntryPath(role: Role): string {
  const entry = ROLE_ENTRIES.find((candidate) => candidate.role === role);
  if (!entry) throw new Error(`Unknown role: ${role}`);
  return `/${entry.slug}/${entry.route}`;
}

export function roleLabel(role: Role): string {
  return ROLE_ENTRIES.find((entry) => entry.role === role)?.label ?? role;
}

const ROLE_GROUPS = [
  { id: "operational", label: "Operational", roles: ["inspector", "leadInspector"] },
  { id: "leadership", label: "Leadership", roles: ["manager", "gm", "finance", "executiveDirector"] },
  { id: "external", label: "External & Configuration", roles: ["auditee", "admin"] },
] as const satisfies ReadonlyArray<{ id: string; label: string; roles: readonly Role[] }>;

export function RoleSelectPage({
  mode,
  onRoleRequest,
}: {
  mode: RoleSelectionMode;
  onRoleRequest(role: Role): void;
}) {
  return (
    <div className="role-select-page login" data-role-selection-mode={mode}>
      <button
        className="login-skip"
        onClick={() => document.querySelector<HTMLElement>("#login-workspaces")?.focus()}
        type="button"
      >
        Skip to workspace selection
      </button>
      <section className="login-hero" aria-labelledby="login-hero-title">
        <img className="login-hero__texture" src={BRAND_ASSETS.loginTexture} alt="" aria-hidden="true" />
        <div className="login-hero__brand">
          <img className="login-hero__logo" src={BRAND_ASSETS.mark} alt="" aria-hidden="true" />
          <div>
            <div className="login__title" role="heading" aria-level={2}>AviaSurveil360</div>
            <div className="login__sub">Civil Aviation Authority surveillance &amp; oversight</div>
          </div>
        </div>
        <div className="login-hero__story">
          <span className="login-demo-badge">Local workspace</span>
          <h1 id="login-hero-title">
            Safer oversight,
            <br />
            from plan to closure.
          </h1>
          <span className="login-hero__rule" aria-hidden="true" />
          <button
            className="login-hero__explore"
            onClick={() => {
              const workspaces = document.querySelector<HTMLElement>("#login-workspaces");
              workspaces?.focus();
              workspaces?.scrollIntoView?.();
            }}
            type="button"
          >
            Explore the local workspace
            <img src={BRAND_ASSETS.icons.logout} width="22" height="22" alt="" aria-hidden="true" />
          </button>
        </div>
        <div className="login-hero__foot">
          <p>Local qualification profile · the production profile uses authenticated sessions.</p>
          <button onClick={() => window.location.reload()} type="button">Reset local session</button>
        </div>
      </section>
      <main className="login-selector" id="login-workspaces" tabIndex={-1}>
        <header className="login-selector__head">
          <span className="login-selector__eyebrow">Role-based access</span>
          <h2>Choose your workspace</h2>
          <p>Enter the local qualification profile through one of eight role perspectives.</p>
        </header>
        <div className="login-selector__groups role-select-panel" data-testid="role-select-panel" aria-label="Role entries">
          {ROLE_GROUPS.map((group) => (
            <section className="role-group" aria-labelledby={`login-group-${group.id}`} key={group.id}>
              <div className="role-group__head">
                <h3 id={`login-group-${group.id}`}>{group.label}</h3>
                <span aria-hidden="true" />
              </div>
              <div className="role-group__list">
                {group.roles.map((role) => {
                  const entry = ROLE_ENTRIES.find((candidate) => candidate.role === role);
                  if (!entry) return null;
                  const recommended = entry.role === "inspector";
                  return (
                    <Link
                      aria-label={`Open ${entry.label}${entry.loginLabel ? ` — ${entry.loginLabel}` : ""}`}
                      className={`role-card${recommended ? " role-card--recommended" : ""}`}
                      data-icon-key={entry.iconKey}
                      key={entry.role}
                      onClick={() => onRoleRequest(entry.role)}
                      to={createRoleEntryPath(entry.role)}
                    >
                      <span className="role-card__icon" aria-hidden="true">
                        <img
                          data-testid="role-card-icon"
                          src={BRAND_ASSETS.icons[entry.loginIconKey ?? entry.iconKey]}
                          alt=""
                          width="24"
                          height="24"
                        />
                      </span>
                      <span className="role-card__copy">
                        <span className="role-card__name-line">
                          <span className="role-card__name">{entry.loginLabel ?? entry.label}</span>
                          {recommended ? <span className="role-card__recommended">Recommended start</span> : null}
                        </span>
                        <span className="role-card__desc">{entry.loginPurpose ?? entry.purpose}</span>
                      </span>
                      <img className="role-card__arrow" src={BRAND_ASSETS.icons.logout} width="20" height="20" alt="" aria-hidden="true" />
                    </Link>
                  );
                })}
              </div>
            </section>
          ))}
        </div>
      </main>
    </div>
  );
}
