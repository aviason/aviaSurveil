import { useState, type FormEvent } from "react";

import { useApplicationRuntime } from "../../app/providers";
import type {
  RequestUserLifecycleInput,
  Role,
  UserLifecycleAction,
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
const lifecycleActions = [
  "UPDATE_ROLES",
  "SUSPEND",
  "REACTIVATE",
  "DEACTIVATE",
  "TRANSFER_ORGANIZATION",
  "RESEND_INVITATION",
  "RESET_PASSWORD",
  "RESET_MFA",
  "FORCE_LOGOUT",
] satisfies UserLifecycleAction[];
const lifecycleActionLabels: Record<UserLifecycleAction, string> = {
  PROVISION: "Provision",
  UPDATE_ROLES: "Update role",
  SUSPEND: "Suspend",
  REACTIVATE: "Reactivate",
  DEACTIVATE: "Deactivate",
  TRANSFER_ORGANIZATION: "Transfer organization",
  RESEND_INVITATION: "Resend invitation",
  RESET_PASSWORD: "Reset password",
  RESET_MFA: "Reset MFA",
  FORCE_LOGOUT: "Force logout",
};

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
  const [reason, setReason] = useState("");
  const [actionReason, setActionReason] = useState("");
  const [lifecycleAction, setLifecycleAction] =
    useState<UserLifecycleAction>("DEACTIVATE");
  const [lifecycleRole, setLifecycleRole] = useState<Role>("inspector");
  const [transferOrganizationId, setTransferOrganizationId] = useState("");
  const [transferEffectiveAt, setTransferEffectiveAt] = useState("");
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
      reason: reason.trim(),
      expectedMembershipRevision: 0,
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

  async function applyLifecycleAction(
    entry: NonNullable<typeof directory.data>["items"][number],
  ) {
    const primaryRole = entry.roles[0];
    if (!primaryRole) {
      setCommandError("The provider account has no approved application role.");
      return;
    }
    const input: RequestUserLifecycleInput = {
      idempotencyKey: lifecycleKey(lifecycleAction, entry.subjectId),
      subjectId: entry.subjectId,
      action: lifecycleAction,
      roles: [
        lifecycleAction === "UPDATE_ROLES" ? lifecycleRole : primaryRole,
      ],
      organizationId:
        lifecycleAction === "TRANSFER_ORGANIZATION"
          ? transferOrganizationId.trim()
          : (entry.organizationId ?? "CAA"),
      reason: actionReason.trim(),
      expectedMembershipRevision: entry.membershipRevision,
    };
    if (lifecycleAction === "TRANSFER_ORGANIZATION") {
      const effectiveAt = new Date(transferEffectiveAt);
      if (!transferOrganizationId.trim() || Number.isNaN(effectiveAt.getTime())) {
        setCommandError(
          "Organization transfer requires a destination and effective time.",
        );
        return;
      }
      input.effectiveAt = effectiveAt.toISOString();
    }
    await requestLifecycle(input);
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
            Reason
            <input
              aria-label="Provisioning reason"
              onChange={(event) => setReason(event.target.value)}
              required
              value={reason}
            />
          </label>
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
          <p>TOTP MFA is optional and self-enrolled in Keycloak.</p>
          <button
            disabled={busy}
            onClick={() => void refreshLifecycle()}
            type="button"
          >
            Refresh provisioning status
          </button>
        </section>
      ) : null}

      {isHttp ? (
        <section
          className="admin-filter-bar"
          aria-label="Account lifecycle action"
        >
          <label>
            Lifecycle action
            <select
              aria-label="Lifecycle action"
              onChange={(event) => setLifecycleAction(
                event.target.value as UserLifecycleAction,
              )}
              value={lifecycleAction}
            >
              {lifecycleActions.map((action) => (
                <option key={action} value={action}>
                  {lifecycleActionLabels[action]}
                </option>
              ))}
            </select>
          </label>
          <label>
            Lifecycle action reason
            <input
              aria-label="Lifecycle action reason"
              onChange={(event) => setActionReason(event.target.value)}
              required
              value={actionReason}
            />
          </label>
          {lifecycleAction === "UPDATE_ROLES" ? (
            <label>
              New role
              <select
                aria-label="Lifecycle target role"
                onChange={(event) => setLifecycleRole(event.target.value as Role)}
                value={lifecycleRole}
              >
                {roles.map((value) => (
                  <option key={value} value={value}>{value}</option>
                ))}
              </select>
            </label>
          ) : null}
          {lifecycleAction === "TRANSFER_ORGANIZATION" ? (
            <>
              <label>
                Destination organization
                <input
                  aria-label="Transfer destination organization"
                  onChange={(event) => setTransferOrganizationId(event.target.value)}
                  required
                  value={transferOrganizationId}
                />
              </label>
              <label>
                Effective time
                <input
                  aria-label="Transfer effective time"
                  onChange={(event) => setTransferEffectiveAt(event.target.value)}
                  required
                  type="datetime-local"
                  value={transferEffectiveAt}
                />
              </label>
            </>
          ) : null}
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
                  <button
                    aria-controls="user-lifecycle-status"
                    disabled={busy || !actionReason.trim() || !primaryRole}
                    onClick={() => void applyLifecycleAction(entry)}
                    type="button"
                  >
                    {lifecycleActionLabels[lifecycleAction]} {entry.subjectId}
                  </button>
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
