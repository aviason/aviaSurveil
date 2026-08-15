import { useEffect, useMemo, useState } from "react";

import { useApplicationRuntime } from "../../app/providers";
import type { ProfileView, Role } from "../../backend/backend";
import { CommandError, errorMessage, WorkspaceShell } from "../shared/workspace-shell";

interface ProfileProjection { role: Role; roleLabel: string; routeLabel: string; displayName: string; workspace: string; testId: string; variant: "profile" | "settings" | "auditee-settings"; }
const inspectorProjection: ProfileProjection = { role: "inspector", roleLabel: "CAA Inspector", routeLabel: "Profile", displayName: "Aylin Sezer", workspace: "My Inspections", testId: "inspector-profile-page", variant: "profile" };
const leadProjection: ProfileProjection = { role: "leadInspector", roleLabel: "Lead Inspector", routeLabel: "Lead Settings", displayName: "Caner Yildiz", workspace: "Lead Inspector Review", testId: "lead-settings-page", variant: "settings" };
const gmProjection: ProfileProjection = { role: "gm", roleLabel: "General Manager", routeLabel: "Settings", displayName: "Omar General Manager", workspace: "General Manager Governance", testId: "gm-settings-page", variant: "settings" };
const executiveProjection: ProfileProjection = { role: "executiveDirector", roleLabel: "Executive Director", routeLabel: "Settings", displayName: "Zara Executive Director", workspace: "Executive Director Governance", testId: "executive-settings-page", variant: "settings" };
const auditeeProjection: ProfileProjection = { role: "auditee", roleLabel: "Auditee", routeLabel: "Auditee Settings", displayName: "Authorized Auditee user", workspace: "Service Provider Portal", testId: "auditee-settings-page", variant: "auditee-settings" };

export function ProfilePage({ projection }: { projection: ProfileProjection }) {
  const runtime = useApplicationRuntime();
  const backend = useMemo(() => runtime.backendForRole?.(projection.role) ?? runtime.backend, [projection.role, runtime]);
  const [profile, setProfile] = useState<ProfileView | null>(null);
  const [displayName, setDisplayName] = useState(projection.displayName);
  const [editing, setEditing] = useState(false);
  const [status, setStatus] = useState("");
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (!backend.profiles) return;
    let cancelled = false;
    void backend.profiles.getMine({}).then((loaded) => {
      if (!cancelled) {
        setProfile(loaded);
        setDisplayName(loaded.displayName);
      }
    }).catch((cause) => !cancelled && setError(errorMessage(cause)));
    return () => { cancelled = true; };
  }, [backend]);
  async function save() {
    if (!backend.profiles || !profile) return;
    try {
      const saved = await backend.profiles.updateMine({ expectedRevision: profile.revision, idempotencyKey: `PROFILE-${profile.subjectId}-${profile.revision + 1}`, displayName });
      setProfile(saved);
      setEditing(false);
      setStatus("Profile saved in the demo workspace.");
    } catch (cause) { setError(errorMessage(cause)); }
  }
  return <WorkspaceShell roleLabel={projection.roleLabel} routeLabel={projection.routeLabel}>
    <div className="inspector-secondary-page inspector-profile-page" data-testid={projection.testId}>
      <header className="inspector-secondary-head workbench-page-header"><div><h1>{projection.variant === "auditee-settings" ? "Service Provider Settings" : projection.variant === "settings" ? "Settings" : "Profile"}</h1>{projection.variant === "settings" ? <p>Configured rules and parameters (preview only).</p> : projection.variant === "auditee-settings" ? <p>Own-organization profile and read-only demo notification preferences.</p> : null}</div></header>
      <CommandError message={error} />
      {projection.variant === "auditee-settings" && profile ? <section className="auditee-settings-scope" aria-label="Organization scope"><h2>Organization scope</h2><dl><div><dt>Organization</dt><dd>{profile.organizationId ?? "Unavailable"}</dd></div><div><dt>Organization ID</dt><dd>{profile.organizationId ?? "Unavailable"}</dd></div><div><dt>Subject ID</dt><dd>{profile.subjectId}</dd></div></dl></section> : null}
      <section className="inspector-profile-card"><dl><div><dt>Name</dt><dd>{displayName}</dd></div><div><dt>Role</dt><dd>{projection.variant === "profile" ? "Inspector Workspace" : projection.roleLabel}</dd></div><div><dt>Workspace</dt><dd>{projection.workspace}</dd></div><div><dt>Scope</dt><dd>{profile?.organizationId ?? "Server-owned session scope"}</dd></div></dl>{editing ? <><label>Display name<input value={displayName} onChange={(event) => setDisplayName(event.target.value)} /></label><button className="inspector-secondary-button inspector-secondary-button--primary" disabled={!profile} onClick={() => void save()} type="button">Save profile</button></> : <button className="inspector-profile-edit" disabled={!profile} onClick={() => setEditing(true)} type="button">Edit profile</button>}{status ? <p role="status">{status}</p> : null}</section>
      {projection.variant === "settings" ? <section className="lead-settings-grid" aria-label="Configured rules"><article><h2>Configured rules</h2><p>Preview only — lifecycle and approval rules are not editable here.</p></article><article><h2>Finding lifecycle</h2><p>Finding → CAP → Evidence → CAA Review → Closure.</p></article><article><h2>Closure policy</h2><p>CAP acceptance alone does not close a Finding.</p></article><article><h2>Notifications</h2><p>Demo in-app records only; no real delivery.</p></article></section> : null}
      {projection.variant === "auditee-settings" ? <section className="auditee-settings-notifications" aria-label="Notification preferences"><h2>Notification preferences</h2><p id="auditee-notification-disabled-reason">Notification preferences are read-only because no typed configuration capability is declared.</p><label><input aria-describedby="auditee-notification-disabled-reason" aria-label="Due Date reminders" checked disabled readOnly type="checkbox" /> Due Date reminders</label><label><input aria-describedby="auditee-notification-disabled-reason" aria-label="Report release updates" checked disabled readOnly type="checkbox" /> Report release updates</label></section> : null}
    </div>
  </WorkspaceShell>;
}

export function InspectorProfilePage() { return <ProfilePage projection={inspectorProjection} />; }
export function LeadSettingsPage() { return <ProfilePage projection={leadProjection} />; }
export function GeneralManagerSettingsPage() { return <ProfilePage projection={gmProjection} />; }
export function ExecutiveDirectorSettingsPage() { return <ProfilePage projection={executiveProjection} />; }
export function AuditeeSettingsPage() { return <ProfilePage projection={auditeeProjection} />; }
