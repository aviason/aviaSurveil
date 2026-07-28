import { useState, type FormEvent } from "react";

import { useApplicationRuntime } from "../../app/providers";
import type {
  RequestUserLifecycleInput,
  Role,
  UserLifecycleRequestView,
} from "../../backend/backend";
import {
  AdminError,
  AdminPage,
  DisabledAdminAction,
  useAdminLoad,
  useAdminWorkspace,
} from "./admin-workspace-shared";

const roles = [
  "inspector",
  "leadInspector",
  "manager",
  "gm",
  "finance",
  "executiveDirector",
  "auditee",
  "admin",
] satisfies Role[];
const caaRoles = new Set<Role>(
  roles.filter((role) => role !== "auditee"),
);

function lifecycleKey(action: string, target: string): string {
  return `user-lifecycle:${action.toLowerCase()}:${target.toLowerCase()}:${Date.now().toString(36)}`;
}

export function UsersRolesPage() {
  const backend = useAdminWorkspace();
  const { buildProfile } = useApplicationRuntime();
  const isHttp = buildProfile === "http";
  const [search, setSearch] = useState("");
  const [role, setRole] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [organizationId, setOrganizationId] = useState("");
  const [provisionRole, setProvisionRole] = useState<Role>("inspector");
  const [lifecycle, setLifecycle] = useState<UserLifecycleRequestView | null>(null);
  const [commandError, setCommandError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const directory = useAdminLoad(
    () => backend.listAccessDirectory({ search, role }),
    [backend, search, role],
  );

  async function requestLifecycle(input: RequestUserLifecycleInput) {
    setBusy(true);
    setCommandError(null);
    try {
      setLifecycle(await backend.requestUserLifecycle(input));
    } catch (cause) {
      setCommandError(
        cause instanceof Error
          ? cause.message
          : "The identity lifecycle request could not be recorded.",
      );
    } finally {
      setBusy(false);
    }
  }

  async function provision(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalizedEmail = email.trim().toLowerCase();
    const normalizedOrganizationId = organizationId.trim();
    if (caaRoles.has(provisionRole) && normalizedOrganizationId !== "CAA") {
      setCommandError("CAA roles require the exact CAA organization.");
      return;
    }
    if (provisionRole === "auditee" && normalizedOrganizationId === "CAA") {
      setCommandError("Auditee access requires a non-CAA organization.");
      return;
    }
    await requestLifecycle({
      idempotencyKey: lifecycleKey("PROVISION", normalizedEmail),
      action: "PROVISION",
      roles: [provisionRole],
      organizationId: normalizedOrganizationId,
      email: normalizedEmail,
      displayName: displayName.trim(),
    });
  }

  async function refreshLifecycle() {
    if (!lifecycle) return;
    setBusy(true);
    setCommandError(null);
    try {
      setLifecycle(await backend.getUserLifecycleRequest({
        requestId: lifecycle.id,
      }));
    } catch (cause) {
      setCommandError(
        cause instanceof Error
          ? cause.message
          : "The identity lifecycle status could not be loaded.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <AdminPage
      testId="admin-users-roles-page"
      routeLabel="Users / Roles"
      title="Users / Roles"
      description={
        isHttp
          ? "Application-authorized Keycloak provisioning and account lifecycle status."
          : "Typed demo access directory. Production identity and account administration remain outside the demo profile."
      }
      action={
        isHttp
          ? (
              <button
                onClick={() => setShowCreate((value) => !value)}
                type="button"
              >
                Create user
              </button>
            )
          : undefined
      }
    >
      <section className="admin-filter-bar" aria-label="User directory filters">
        <label>
          Search
          <input
            aria-label="Search users"
            onChange={(event) => setSearch(event.target.value)}
            value={search}
          />
        </label>
        <label>
          Role
          <select
            aria-label="User role"
            onChange={(event) => setRole(event.target.value)}
            value={role}
          >
            <option value="">All roles</option>
            {roles.map((value) => (
              <option key={value} value={value}>{value}</option>
            ))}
          </select>
        </label>
      </section>

      {isHttp && showCreate ? (
        <form className="admin-filter-bar" onSubmit={(event) => void provision(event)}>
          <label>
            Email
            <input
              aria-label="Provisioning email"
              onChange={(event) => setEmail(event.target.value)}
              required
              type="email"
              value={email}
            />
          </label>
          <label>
            Display name
            <input
              aria-label="Provisioning display name"
              onChange={(event) => setDisplayName(event.target.value)}
              required
              value={displayName}
            />
          </label>
          <label>
            Organization
            <input
              aria-label="Provisioning organization"
              onChange={(event) => setOrganizationId(event.target.value)}
              required
              value={organizationId}
            />
          </label>
          <label>
            Role
            <select
              aria-label="Provisioning role"
              onChange={(event) => setProvisionRole(event.target.value as Role)}
              value={provisionRole}
            >
              {roles.map((value) => (
                <option key={value} value={value}>{value}</option>
              ))}
            </select>
          </label>
          <button disabled={busy} type="submit">Submit provisioning</button>
        </form>
      ) : null}

      <AdminError message={directory.error ?? commandError} />
      {isHttp && lifecycle ? (
        <section
          className="admin-record-card"
          aria-live="polite"
          id="user-lifecycle-status"
        >
          <b>Provisioning status: {lifecycle.status}</b>
          <small>{lifecycle.id}</small>
          {lifecycle.failureReason ? <p>{lifecycle.failureReason}</p> : null}
          <p>First login requires TOTP MFA enrollment in Keycloak.</p>
          <button
            disabled={busy}
            onClick={() => void refreshLifecycle()}
            type="button"
          >
            Refresh provisioning status
          </button>
        </section>
      ) : null}

      <div
        className="admin-card-register admin-dense-register"
        role="list"
        aria-label={isHttp ? "Identity access directory" : "Demo access directory"}
      >
        {directory.data?.items.map((entry) => {
          const primaryRole = entry.roles[0];
          const demoReason =
            `${entry.subjectId} account provisioning and role changes are production-only and require Plan 3 Keycloak administration.`;
          const mfaReason =
            "MFA enrollment is user-controlled through the required first-login Keycloak action.";
          return (
            <article className="admin-record-card" key={entry.subjectId} role="listitem">
              <header>
                <div>
                  <b>{entry.displayName}</b>
                  <small>{entry.subjectId}</small>
                </div>
                <span>{entry.roles.join(", ")}</span>
              </header>
              <dl>
                <div><dt>Organization scope</dt><dd>{entry.organizationId ?? "CAA internal"}</dd></div>
                <div><dt>Email</dt><dd>{entry.email}</dd></div>
                <div><dt>MFA</dt><dd>{entry.mfaState}</dd></div>
                <div><dt>Invitation</dt><dd>{entry.invitationState}</dd></div>
                <div><dt>Account status</dt><dd>{entry.accountStatus}</dd></div>
              </dl>
              <div className="admin-inline-actions">
                {isHttp ? (
                  <>
                    <DisabledAdminAction label={`Manage MFA ${entry.subjectId}`} reason={mfaReason} />
                    <button
                      aria-controls="user-lifecycle-status"
                      disabled={busy}
                      onClick={() => void requestLifecycle({
                        idempotencyKey: lifecycleKey("SUSPEND", entry.subjectId),
                        subjectId: entry.subjectId,
                        action: "SUSPEND",
                        roles: primaryRole ? [primaryRole] : [],
                        organizationId: entry.organizationId ?? "CAA",
                      })}
                      type="button"
                    >
                      Deactivate {entry.subjectId}
                    </button>
                  </>
                ) : (
                  <>
                    <DisabledAdminAction label={`Invite ${entry.subjectId}`} reason={demoReason} />
                    <DisabledAdminAction label={`Change role ${entry.subjectId}`} reason={demoReason} />
                    <DisabledAdminAction label={`Manage MFA ${entry.subjectId}`} reason={demoReason} />
                    <DisabledAdminAction label={`Deactivate ${entry.subjectId}`} reason={demoReason} />
                  </>
                )}
              </div>
            </article>
          );
        })}
      </div>
      {!isHttp ? (
        <DisabledAdminAction
          label="Create user"
          reason="User creation is production-only and requires Plan 3 Keycloak administration."
        />
      ) : null}
    </AdminPage>
  );
}
