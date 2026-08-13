import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type MouseEvent,
} from "react";

import { useApplicationRuntime } from "../../app/providers";
import type {
  AdminAccessDirectoryEntryView,
  RequestUserLifecycleInput,
  Role,
  UserLifecycleAction,
  UserLifecycleRequestView,
} from "../../backend/backend";
import { useDialogFocus } from "../../ui/dialog-focus";
import {
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

type Confirmation =
  | {
      kind: "provision";
      input: RequestUserLifecycleInput;
      target: string;
    }
  | {
      kind: "lifecycle";
      action: Exclude<UserLifecycleAction, "PROVISION">;
      entry: AdminAccessDirectoryEntryView;
      target: string;
    };

function lifecycleKey(action: string, target: string): string {
  return `user-lifecycle:${action.toLowerCase()}:${target.toLowerCase()}:${Date.now().toString(36)}`;
}

function normalizedState(value: string): string {
  return value.trim().toLowerCase();
}

function formatInstant(value: string): string {
  const instant = new Date(value);
  if (Number.isNaN(instant.getTime())) return value;
  return `${new Intl.DateTimeFormat("en-GB", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
    timeZone: "UTC",
  }).format(instant)} UTC`;
}

function lifecycleStatusLabel(status: UserLifecycleRequestView["status"]): string {
  switch (status) {
    case "PENDING":
      return "Pending";
    case "RUNNING":
      return "Running";
    case "SUCCEEDED":
      return "Succeeded";
    case "FAILED_RETRYABLE":
      return "Failed — retry available";
    case "FAILED_PERMANENT":
      return "Failed — permanent";
    case "MANUAL_REVIEW":
      return "Manual review required";
    case "FAILED":
      return "Failed";
  }
}

function unavailableReason(
  entry: AdminAccessDirectoryEntryView,
  action: Exclude<UserLifecycleAction, "PROVISION">,
  busy: boolean,
): string | null {
  if (busy) return "Another lifecycle request is being submitted.";
  if (
    !entry.membershipId ||
    entry.membershipRevision < 1 ||
    entry.roles.length !== 1 ||
    !entry.organizationId
  ) {
    return "No exact desired application membership is linked to this provider account.";
  }

  const membershipState = normalizedState(entry.membershipState);
  switch (action) {
    case "SUSPEND":
      if (membershipState === "suspended") {
        return "Membership is already suspended.";
      }
      if (membershipState === "deactivated") {
        return "Deactivated membership must be reactivated before suspension.";
      }
      if (membershipState !== "active") {
        return "Suspension is available only for active memberships.";
      }
      return null;
    case "REACTIVATE":
      if (
        membershipState !== "suspended" &&
        membershipState !== "deactivated"
      ) {
        return "Reactivation is available only for suspended or deactivated memberships.";
      }
      return null;
    case "DEACTIVATE":
      return membershipState === "deactivated"
        ? "Membership is already deactivated."
        : null;
    case "TRANSFER_ORGANIZATION":
      return entry.roles[0] === "auditee"
        ? null
        : "CAA roles require the exact CAA organization and cannot transfer.";
    case "RESEND_INVITATION":
      return membershipState === "invited"
        ? null
        : "Invitation resend is available only for invited memberships.";
    case "RESET_MFA":
      return entry.mfaEnrolled
        ? null
        : "MFA reset requires an enrolled provider authenticator.";
    case "UPDATE_ROLES":
    case "RESET_PASSWORD":
    case "FORCE_LOGOUT":
      return null;
  }
}

function lifecycleRetryable(status: UserLifecycleRequestView["status"]): boolean {
  return status === "FAILED_RETRYABLE";
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
  const [organizationId, setOrganizationId] = useState("");
  const [provisionRole, setProvisionRole] = useState<Role>("inspector");
  const [confirmation, setConfirmation] = useState<Confirmation | null>(null);
  const [actionReason, setActionReason] = useState("");
  const [lifecycleRole, setLifecycleRole] = useState<Role>("inspector");
  const [transferOrganizationId, setTransferOrganizationId] = useState("");
  const [transferEffectiveAt, setTransferEffectiveAt] = useState("");
  const [lifecycle, setLifecycle] = useState<UserLifecycleRequestView | null>(null);
  const [commandError, setCommandError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [submittingLabel, setSubmittingLabel] = useState<string | null>(null);
  const [directoryRefreshCount, setDirectoryRefreshCount] = useState(0);
  const dialogContainerRef = useRef<HTMLDivElement | null>(null);
  const dialogInitialFocusRef = useRef<HTMLButtonElement | null>(null);
  const dialogOpenerRef = useRef<HTMLButtonElement | null>(null);
  const commandErrorRef = useRef<HTMLDivElement | null>(null);
  const lifecycleRefreshInFlightRef = useRef(false);
  const directory = useAdminLoad(
    () => backend.listAccessDirectory({ search, role }),
    [backend, search, role],
  );

  function closeConfirmation(returnFocus = true) {
    setConfirmation(null);
    setActionReason("");
    setTransferOrganizationId("");
    setTransferEffectiveAt("");
    if (returnFocus) dialogOpenerRef.current?.focus();
  }

  useDialogFocus({
    containerRef: dialogContainerRef,
    initialFocusRef: dialogInitialFocusRef,
    onClose: closeConfirmation,
    open: confirmation !== null,
  });

  useEffect(() => {
    if (commandError) commandErrorRef.current?.focus();
  }, [commandError]);

  async function requestLifecycle(
    input: RequestUserLifecycleInput,
  ): Promise<boolean> {
    setBusy(true);
    setSubmittingLabel(lifecycleActionLabels[input.action]);
    setCommandError(null);
    try {
      const result = await backend.requestUserLifecycle(input);
      setLifecycle(result);
      if (result.status === "SUCCEEDED") directory.reload();
      return true;
    } catch (cause) {
      setCommandError(
        cause instanceof Error
          ? cause.message
          : "The identity lifecycle request could not be recorded.",
      );
      return false;
    } finally {
      setBusy(false);
      setSubmittingLabel(null);
    }
  }

  function reviewProvision(
    event: FormEvent<HTMLFormElement>,
  ) {
    event.preventDefault();
    const normalizedEmail = email.trim().toLowerCase();
    const normalizedOrganizationId = organizationId.trim();
    const normalizedReason = reason.trim();
    if (caaRoles.has(provisionRole) && normalizedOrganizationId !== "CAA") {
      setCommandError("CAA roles require the exact CAA organization.");
      return;
    }
    if (provisionRole === "auditee" && normalizedOrganizationId === "CAA") {
      setCommandError("Auditee access requires a non-CAA organization.");
      return;
    }
    const submitter = (event.nativeEvent as SubmitEvent).submitter;
    dialogOpenerRef.current =
      submitter instanceof HTMLButtonElement
        ? submitter
        : null;
    setCommandError(null);
    setConfirmation({
      kind: "provision",
      target: normalizedEmail,
      input: {
        idempotencyKey: lifecycleKey("PROVISION", normalizedEmail),
        action: "PROVISION",
        roles: [provisionRole],
        organizationId: normalizedOrganizationId,
        email: normalizedEmail,
        displayName: displayName.trim(),
        reason: normalizedReason,
        expectedMembershipRevision: 0,
      },
    });
  }

  function openLifecycleConfirmation(
    entry: AdminAccessDirectoryEntryView,
    action: Exclude<UserLifecycleAction, "PROVISION">,
    event: MouseEvent<HTMLButtonElement>,
  ) {
    dialogOpenerRef.current = event.currentTarget;
    setCommandError(null);
    setActionReason("");
    setLifecycleRole(entry.roles[0] ?? "inspector");
    setTransferOrganizationId("");
    setTransferEffectiveAt("");
    setConfirmation({
      kind: "lifecycle",
      action,
      entry,
      target: entry.subjectId,
    });
  }

  async function confirmRequest(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!confirmation) return;
    if (confirmation.kind === "provision") {
      const input = confirmation.input;
      closeConfirmation(false);
      const succeeded = await requestLifecycle(input);
      if (succeeded) {
        setShowCreate(false);
        setEmail("");
        setDisplayName("");
        setReason("");
        setOrganizationId("");
        setProvisionRole("inspector");
      }
      return;
    }

    const primaryRole = confirmation.entry.roles[0];
    if (!primaryRole) {
      setCommandError("The provider account has no approved application role.");
      closeConfirmation(false);
      return;
    }
    const input: RequestUserLifecycleInput = {
      idempotencyKey: lifecycleKey(
        confirmation.action,
        confirmation.entry.subjectId,
      ),
      subjectId: confirmation.entry.subjectId,
      action: confirmation.action,
      roles: [
        confirmation.action === "UPDATE_ROLES"
          ? lifecycleRole
          : primaryRole,
      ],
      organizationId:
        confirmation.action === "TRANSFER_ORGANIZATION"
          ? transferOrganizationId.trim()
          : confirmation.entry.organizationId!,
      reason: actionReason.trim(),
      expectedMembershipRevision: confirmation.entry.membershipRevision,
    };
    if (confirmation.action === "TRANSFER_ORGANIZATION") {
      const effectiveAt = new Date(transferEffectiveAt);
      if (
        !transferOrganizationId.trim() ||
        transferOrganizationId.trim() === confirmation.entry.organizationId ||
        Number.isNaN(effectiveAt.getTime())
      ) {
        setCommandError(
          "Organization transfer requires a different destination and valid effective time.",
        );
        return;
      }
      input.effectiveAt = effectiveAt.toISOString();
    }
    closeConfirmation(false);
    await requestLifecycle(input);
  }

  async function refreshLifecycle() {
    if (!lifecycle || lifecycleRefreshInFlightRef.current) return;
    lifecycleRefreshInFlightRef.current = true;
    setBusy(true);
    setSubmittingLabel("Lifecycle status");
    setCommandError(null);
    try {
      const result = await backend.getUserLifecycleRequest({
        requestId: lifecycle.id,
      });
      setLifecycle(result);
      if (result.status === "SUCCEEDED") directory.reload();
    } catch (cause) {
      setCommandError(
        cause instanceof Error
          ? cause.message
          : "The identity lifecycle status could not be loaded.",
      );
    } finally {
      lifecycleRefreshInFlightRef.current = false;
      setBusy(false);
      setSubmittingLabel(null);
    }
  }

  function refreshDirectory() {
    setDirectoryRefreshCount((count) => count + 1);
    directory.reload();
  }

  const confirmationAction =
    confirmation?.kind === "provision" ? "PROVISION" : confirmation?.action;
  const confirmationLabel = confirmationAction
    ? lifecycleActionLabels[confirmationAction]
    : "";
  const transferInvalid =
    confirmation?.kind === "lifecycle" &&
    confirmation.action === "TRANSFER_ORGANIZATION" &&
    (
      !transferOrganizationId.trim() ||
      transferOrganizationId.trim() === confirmation.entry.organizationId ||
      Number.isNaN(new Date(transferEffectiveAt).getTime())
    );
  const roleUnchanged =
    confirmation?.kind === "lifecycle" &&
    confirmation.action === "UPDATE_ROLES" &&
    lifecycleRole === confirmation.entry.roles[0];
  const confirmationDisabled =
    busy ||
    (confirmation?.kind === "lifecycle" &&
      (!actionReason.trim() || transferInvalid || roleUnchanged));

  return (
    <AdminPage
      testId="admin-users-roles-page"
      routeLabel="Users / Roles"
      title="Users / Roles"
      description={
        isHttp
          ? "Application-authorized identity directory and reasoned account lifecycle controls."
          : "Typed demo access directory. Production identity and account administration remain outside the demo profile."
      }
      action={
        isHttp ? (
          <button
            aria-expanded={showCreate}
            onClick={() => setShowCreate((value) => !value)}
            type="button"
          >
            Create user
          </button>
        ) : undefined
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
        <button
          disabled={busy}
          onClick={refreshDirectory}
          type="button"
        >
          Refresh user directory
        </button>
      </section>

      {isHttp && showCreate ? (
        <form
          className="admin-filter-bar admin-lifecycle-create"
          onSubmit={reviewProvision}
        >
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
          <button disabled={busy} type="submit">Review provisioning</button>
        </form>
      ) : null}

      {commandError ? (
        <div
          className="command-error admin-command-error"
          ref={commandErrorRef}
          role="alert"
          tabIndex={-1}
        >
          <strong>Action could not be completed</strong>
          <span>{commandError}</span>
        </div>
      ) : null}

      {busy && submittingLabel ? (
        <p className="admin-directory-state" role="status">
          Submitting {submittingLabel.toLowerCase()}…
        </p>
      ) : null}

      {directoryRefreshCount > 0 ? (
        <p className="admin-directory-state" data-durable-outcome role="status">
          User directory refresh requested ({directoryRefreshCount}).
        </p>
      ) : null}

      {isHttp && lifecycle ? (
        <section
          aria-label="Lifecycle request status"
          className="admin-record-card admin-lifecycle-status"
          id="user-lifecycle-status"
          role="status"
        >
          <header>
            <div>
              <b>{lifecycleActionLabels[lifecycle.action]} request</b>
              <small>{lifecycle.id}</small>
            </div>
            <span>{lifecycle.status}</span>
          </header>
          {lifecycle.action === "PROVISION" ? (
            <b>Provisioning status: {lifecycle.status}</b>
          ) : null}
          <p className="admin-lifecycle-status__summary">
            {lifecycleStatusLabel(lifecycle.status)}
          </p>
          {lifecycle.failureReason ? <p>{lifecycle.failureReason}</p> : null}
          <dl>
            <div><dt>Requested by</dt><dd>{lifecycle.requestedBySubjectId}</dd></div>
            <div><dt>Expected revision</dt><dd>{lifecycle.expectedMembershipRevision}</dd></div>
            <div><dt>Resulting revision</dt><dd>{lifecycle.resultingMembershipRevision}</dd></div>
            <div><dt>Provider acknowledgement</dt><dd>{lifecycle.providerAcknowledgedAt ? formatInstant(lifecycle.providerAcknowledgedAt) : "Pending"}</dd></div>
            <div><dt>Attempts</dt><dd>{lifecycle.attemptCount}</dd></div>
          </dl>
          <p>TOTP MFA is optional and self-enrolled with the configured identity provider.</p>
          <button
            disabled={busy}
            onClick={() => void refreshLifecycle()}
            type="button"
          >
            {lifecycleRetryable(lifecycle.status)
              ? "Retry lifecycle status"
              : lifecycle.action === "PROVISION"
                ? "Refresh provisioning status"
                : "Refresh lifecycle status"}
          </button>
        </section>
      ) : null}

      {directory.data === null && directory.error === null ? (
        <p className="admin-directory-state" role="status">
          Loading user directory…
        </p>
      ) : null}
      {directory.data === null && directory.error ? (
        <div className="admin-directory-state admin-directory-state--error">
          <p role="alert">{directory.error}</p>
          <button onClick={refreshDirectory} type="button">
            Retry user directory
          </button>
        </div>
      ) : null}
      {directory.data && directory.error ? (
        <div className="admin-directory-state admin-directory-state--stale">
          <p role="status">
            Showing the last successful directory because {directory.error}
          </p>
          <button onClick={refreshDirectory} type="button">
            Retry user directory
          </button>
        </div>
      ) : null}
      {directory.data?.items.length === 0 ? (
        <p className="admin-directory-state" role="status">
          No accounts match the current filters.
        </p>
      ) : null}

      <div
        className="admin-card-register admin-dense-register"
        role="list"
        aria-label={isHttp ? "Identity access directory" : "Demo access directory"}
      >
        {directory.data?.items.map((entry) => {
          const demoReason =
            `${entry.subjectId} account provisioning and role changes are production-only and require configured identity-provider administration.`;
          return (
            <article
              aria-label={`${entry.displayName} ${entry.subjectId}`}
              className="admin-record-card admin-identity-card"
              key={entry.subjectId}
              role="listitem"
            >
              <header>
                <div>
                  <b>{entry.displayName}</b>
                  <small>{entry.subjectId}</small>
                </div>
                <span>{entry.accountStatus}</span>
              </header>
              <dl>
                <div><dt>Provider account</dt><dd>{entry.accountStatus}</dd></div>
                <div><dt>Application profile</dt><dd>{entry.applicationProfileState}</dd></div>
                <div><dt>Role</dt><dd>{entry.roles.join(", ") || "No application role"}</dd></div>
                <div><dt>Organization</dt><dd>{entry.organizationId ?? "No application organization"}</dd></div>
                <div><dt>Email</dt><dd>{entry.email}</dd></div>
                <div><dt>Invitation</dt><dd>{entry.invitationState}</dd></div>
                <div><dt>MFA</dt><dd>{entry.mfaState}</dd></div>
                <div><dt>Required actions</dt><dd>{entry.requiredActions.join(", ") || "None"}</dd></div>
                <div>
                  <dt>Desired membership</dt>
                  <dd>
                    {entry.membershipId
                      ? `${entry.membershipState} · ${entry.membershipId} · revision ${entry.membershipRevision}`
                      : "No desired membership"}
                  </dd>
                </div>
                <div><dt>Authority alignment</dt><dd>{entry.membershipDrift}</dd></div>
                <div>
                  <dt>Last successful session</dt>
                  <dd>
                    {entry.lastSuccessfulSessionAt ? (
                      <time dateTime={entry.lastSuccessfulSessionAt}>
                        {formatInstant(entry.lastSuccessfulSessionAt)}
                      </time>
                    ) : "No successful application session recorded"}
                  </dd>
                </div>
                <div>
                  <dt>Provider observed</dt>
                  <dd>
                    {entry.providerObservedAt ? (
                      <time dateTime={entry.providerObservedAt}>
                        {formatInstant(entry.providerObservedAt)}
                      </time>
                    ) : "No provider observation recorded"}
                  </dd>
                </div>
              </dl>
              <h2>Lifecycle actions</h2>
              <div className="admin-inline-actions">
                {isHttp ? lifecycleActions.map((action) => {
                  const reasonUnavailable = unavailableReason(entry, action, busy);
                  const label = `${lifecycleActionLabels[action]} ${entry.subjectId}`;
                  return reasonUnavailable ? (
                    <DisabledAdminAction
                      key={action}
                      label={label}
                      reason={reasonUnavailable}
                    />
                  ) : (
                    <button
                      aria-controls="user-lifecycle-status"
                      key={action}
                      onClick={(event) =>
                        openLifecycleConfirmation(entry, action, event)}
                      type="button"
                    >
                      {label}
                    </button>
                  );
                }) : (
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
        reason="User creation is production-only and requires configured identity-provider administration."
        />
      ) : null}

      {confirmation ? (
        <div
          aria-labelledby="user-lifecycle-confirmation-title"
          aria-modal="true"
          className="admin-lifecycle-dialog"
          ref={dialogContainerRef}
          role="dialog"
          tabIndex={-1}
        >
          <form
            className="admin-lifecycle-dialog__panel"
            onSubmit={(event) => void confirmRequest(event)}
          >
            <header>
              <div>
                <p className="eyebrow">Reasoned authority change</p>
                <h2 id="user-lifecycle-confirmation-title">
                  Confirm {confirmationLabel} for {confirmation.target}
                </h2>
              </div>
              <button
                aria-label="Close confirmation"
                onClick={() => closeConfirmation()}
                ref={dialogInitialFocusRef}
                type="button"
              >
                ×
              </button>
            </header>
            {confirmation.kind === "provision" ? (
              <dl>
                <div><dt>Email</dt><dd>{confirmation.input.email}</dd></div>
                <div><dt>Display name</dt><dd>{confirmation.input.displayName}</dd></div>
                <div><dt>Role</dt><dd>{confirmation.input.roles[0]}</dd></div>
                <div><dt>Organization</dt><dd>{confirmation.input.organizationId}</dd></div>
                <div><dt>Reason</dt><dd>{confirmation.input.reason}</dd></div>
              </dl>
            ) : (
              <>
                <p>
                  This request is revision-bound to membership revision{" "}
                  <b>{confirmation.entry.membershipRevision}</b>. Provider and
                  application state will be reconciled by the server.
                </p>
                {confirmation.action === "UPDATE_ROLES" ? (
                  <label>
                    New role
                    <select
                      aria-label="New role"
                      onChange={(event) => setLifecycleRole(event.target.value as Role)}
                      value={lifecycleRole}
                    >
                      {(confirmation.entry.organizationId === "CAA"
                        ? roles.filter((value) => value !== "auditee")
                        : ["auditee"] satisfies Role[]
                      ).map((value) => (
                        <option key={value} value={value}>{value}</option>
                      ))}
                    </select>
                  </label>
                ) : null}
                {confirmation.action === "TRANSFER_ORGANIZATION" ? (
                  <div className="admin-lifecycle-dialog__transfer">
                    <label>
                      Destination organization
                      <input
                        aria-label="Destination organization"
                        onChange={(event) => setTransferOrganizationId(event.target.value)}
                        required
                        value={transferOrganizationId}
                      />
                    </label>
                    <label>
                      Effective time
                      <input
                        aria-label="Effective time"
                        onChange={(event) => setTransferEffectiveAt(event.target.value)}
                        required
                        type="datetime-local"
                        value={transferEffectiveAt}
                      />
                    </label>
                  </div>
                ) : null}
                <label>
                  Action reason
                  <textarea
                    aria-label="Action reason"
                    onChange={(event) => setActionReason(event.target.value)}
                    required
                    rows={3}
                    value={actionReason}
                  />
                </label>
              </>
            )}
            <p className="admin-lifecycle-dialog__guardrail">
              No password, provider token, recovery code, or TOTP secret is
              created or displayed by AviaSurveil360.
            </p>
            <div className="admin-lifecycle-dialog__actions">
              <button onClick={() => closeConfirmation()} type="button">
                Cancel
              </button>
              <button disabled={confirmationDisabled} type="submit">
                Confirm {confirmationLabel}
              </button>
            </div>
          </form>
        </div>
      ) : null}
    </AdminPage>
  );
}
