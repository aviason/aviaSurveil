# Preprod Identity And Data Profile Contract

**Contract status:** `active — version 1.0.0 approved`

**Profile and policy status:** `approved — owner decision recorded`

**Runtime status:** Tasks 2–6 identity-directory, lifecycle, invitation,
recovery, MFA, session-authority, Admin-experience, and isolated loader
foundations are `verified locally`; Tasks 7–9 remain `not run`.

This document began as the Plan 5 Task 1 contract and owner-decision package.
It defines machine-checkable identity, authority, privacy, and deterministic
data-profile boundaries. Tasks 2–6 now implement and locally verify the
directory, lifecycle, invitation/recovery/MFA, session, and complete Users and
Roles Admin-experience portions plus the deterministic out-of-process loader
and immutable control-record boundary. Connected scenario generation, runtime
profile qualification, and generated datasets remain `not run`.

The 2026-07-27 authorization allowed this Task 1 package to proceed without the
then-open Plan 1 visual stakeholder closure. Plan 1 and the combined Plans 2–4
local stakeholder disposition were completed on 2026-07-28. A later explicit
authorization started the sequential Plan 5 implementation. Tasks 2–6 are
complete, `verified locally`, and published through Task 6 revision
`26c7022`.

## Binding Invariants

- Public self-registration is disabled.
- The eight application roles are `inspector`, `leadInspector`, `manager`,
  `finance`, `gm`, `executiveDirector`, `auditee`, and `admin`.
- Every CAA role is scoped to the exact `CAA` organization. An `auditee`
  membership is scoped to exactly one non-CAA organization.
- Auditee and CAA roles cannot coexist. A CAA role outside `CAA`, an Auditee
  role inside `CAA`, multiple simultaneous organizations, duplicate roles, and
  any unapproved multi-role set are forbidden.
- Roles and organization come from server-owned desired membership and fresh
  provider observation. Browser or API clients cannot author authority.
- Provider account, desired membership, application profile, invitation and
  recovery delivery, MFA enrollment, and application session are separate
  authorities and separate state machines.
- Every authority mutation carries `expectedMembershipRevision`. Initial
  provisioning uses revision `0`; a stale revision fails with a typed conflict
  and no provider, delivery, session, audit-success, or outbox-success side
  effect.
- Normal API, worker, scheduler, migration, and HTTP artifacts contain no
  testprofile, loader, seed/reset route, seed/reset startup hook, loader
  command, or canonical-header authentication.
- Synthetic data contains no real or real-looking PII and no credential,
  token, secret, recovery code, or private key.
- Repeatability targets one complete disposable namespace. Selective deletion
  of append-only history is forbidden.

## Role And Organization Matrix

| Role | Permitted organization scope | Entry authority |
|---|---|---|
| `inspector` | Exact `CAA` | Server-owned desired membership plus fresh provider match |
| `leadInspector` | Exact `CAA` | Server-owned desired membership plus fresh provider match |
| `manager` | Exact `CAA` | Server-owned desired membership plus fresh provider match |
| `finance` | Exact `CAA` | Server-owned desired membership plus fresh provider match |
| `gm` | Exact `CAA` | Server-owned desired membership plus fresh provider match |
| `executiveDirector` | Exact `CAA` | Server-owned desired membership plus fresh provider match |
| `admin` | Exact `CAA` | Server-owned desired membership plus fresh provider match |
| `auditee` | Exactly one non-CAA organization | Server-owned desired membership plus fresh provider match |

No multi-role combination is authorized by this Task 1 package. The approved
fail-closed policy is exactly one role per membership. Any future CAA-only
combination requires an explicit allowlist in a new contract version. The
Auditee/CAA separation is invariant and is not an option in that decision.

## Authority And State Machines

| State machine | Authority | Required states |
|---|---|---|
| Provider account | Keycloak observation | absent, enabled, disabled, locked, unavailable |
| Desired membership | AviaSurveil append-only aggregate | requested, invited, active, suspended, deactivated, reactivation-pending |
| Application profile | AviaSurveil profile store | absent, pending-link, linked, retained-inactive |
| Invitation/recovery | AviaSurveil request and delivery facts plus Keycloak acknowledgement | none, issued, delivered, retryable-failure, terminal-failure, expired, consumed, cancelled |
| MFA | Keycloak observation only | unknown, unenrolled, enrollment-required, enrolled, reset-pending, unavailable |
| Session | AviaSurveil server-side session store | none, active, revocation-pending, revoked, expired, denied-stale-authority |

The membership business key is immutable `membershipId`; it is not an email,
provider subject, display name, or organization ID. Every version records a
monotonic positive `revision`, requested and effective times, actor subject,
reason, exact role set, exact organization, provider-observation reference,
and reconciliation state. Earlier versions remain append-only.

## Membership Lifecycle

- `requested`: a reasoned, revisioned application request exists; no login
  authority exists.
- `invited`: a Keycloak execute-actions invitation was issued and tracked; no
  application session exists.
- `active`: approved desired membership, fresh exact provider observation,
  required actions, MFA policy, application profile, and session revision may
  authorize login.
- `suspended`: temporary authority removal; provider login is disabled and
  sessions are revoked, while the membership identity and history remain.
- `deactivated`: future authority is removed without deleting identity or
  rewriting historical ownership; membership and provider identifiers are
  retained permanently and never reassigned.
- `reactivation-pending`: a new reasoned owner decision is awaiting provider
  reconciliation and any required invitation/recovery actions; it is not
  active authority.

## Authority Mutation Outcomes

| Mutation | Membership result | Provider/invitation/MFA result | Session result |
|---|---|---|---|
| Activate | invited → active after all gates | Provider enabled; approved required actions complete | New login allowed only at the current membership revision |
| Suspend | active → suspended, revision +1 | Provider disabled; pending invitation/recovery cancelled | All sessions and offline grants revoked |
| Deactivate | any non-deactivated state → deactivated, revision +1 | Provider disabled; delivery facts and permanent non-reuse tombstone retained | All sessions and offline grants revoked |
| Request reactivation | deactivated → reactivation-pending, revision +1 | No authority until owner approval and reconciliation | Login denied |
| Reactivate | reactivation-pending → invited or active, revision +1 | New execute-actions path when required; provider exact-match required | Old sessions remain revoked; fresh login required |
| Change roles | state retained, exact role set changed, revision +1 | Provider roles replaced atomically or operation fails closed | All sessions and offline grants revoked |
| Transfer organization | state retained, future organization changed, revision +1 | Provider organization replaced; historical record ownership unchanged | All sessions and offline grants revoked |
| MFA reset | membership state retained | Existing provider TOTP enrollment cleared; re-enrollment remains optional; no secret copied | All sessions and offline grants revoked until fresh login |
| Forced logout | membership revision checked but not incremented | Provider session logout requested and reconciled | All application sessions and offline grants revoked |
| Invitation resend | membership revision checked | New execute-actions delivery per owner policy; earlier delivery retained | No authority granted |
| Start recovery | membership revision checked | Approved recovery execute-actions only; no application password/token | All sessions revoked before recovery |

## Invitation And Provider Observation

Provisioning uses authenticated local SMTP/Mailpit and Keycloak
execute-actions delivery. `UPDATE_PASSWORD` and `VERIFY_EMAIL` are mandatory
for every newly provisioned passwordless account. Invitations expire after 24
hours; resend invalidates the prior action and is limited to three attempts per
24 hours. TOTP is optional for every role, so `CONFIGURE_TOTP` is never a
required action in this contract version. Recovery and MFA reset are reasoned
Admin-assisted execute-actions operations with a 15-minute expiry and session
revocation. MFA reset clears the existing enrollment but does not force
re-enrollment. No temporary password, action token, recovery code, or TOTP
secret may be generated, returned, or stored by the application.

Provider reconciliation uses a 30-second heartbeat, a 60-second maximum
observation age, and a 120-second fail-closed denial deadline. Observation age
is not user inactivity: healthy reconciliation does not log out an idle user.
An observation older than 60 seconds denies new login and authority mutation
and starts session revocation; all affected sessions and offline grants must
be denied by 120 seconds after the last successful observation. Provider
disablement, role drift, and organization drift fail closed immediately.

## Four Deterministic Profiles

All volumes and resource envelopes below are approved contract targets, not
runtime feasibility claims. Every profile uses the same lifecycle scenario catalog, all
eight roles, the complete visible-action catalog, and all 86 routes with an
explicit authorized-data, intentional-empty, or denied disposition.

| Profile version | Organizations | Users | Audits | Checklist responses | Findings | CAP revisions | Evidence versions | Report versions | Communications + notifications | Audit events | Approved maximum duration / memory / disk |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---|
| `smoke@1.0.0` | 3 | 9 | 2 | 24 | 8 | 12 | 16 | 6 | 40 | 250 | 120 s / 1,024 MiB / 2,048 MiB |
| `acceptance@1.0.0` | 25 | 250 | 1,000 | 10,000 | 3,000 | 4,500 | 6,000 | 2,000 | 20,000 | 100,000 | 1,200 s / 4,096 MiB / 20,480 MiB |
| `realistic@1.0.0` | 100 | 2,000 | 20,000 | 250,000 | 60,000 | 100,000 | 200,000 | 75,000 | 1,000,000 | 5,000,000 | 7,200 s / 12,288 MiB / 51,200 MiB |
| `stress@1.0.0` | 200 | 4,000 | 40,000 | 500,000 | 120,000 | 200,000 | 400,000 | 150,000 | 2,000,000 | 10,000,000 | 28,800 s / 12,288 MiB / 65,536 MiB |

The stress profile has an exact 8 GiB synthetic object payload. A profile that
cannot preserve its exact counts and complete catalogs inside its approved
envelope fails; silent row, route, action, scenario, or payload reduction is
forbidden. Feasibility remains `not run` until the owning loader/runtime tasks
are separately authorized and executed.

The machine-readable contract below closes the generated-family catalog,
defines every exact count and distribution, and requires a SHA-256 relationship
digest for every family. The digest is computed later over actual canonical
relationship tuples by the separately authorized loader; placeholders are
forbidden. Task 1 validates the digest requirement and tuple schema, not
runtime-generated digest values.

## Synthetic Data Dictionary And Source Boundary

Allowed names are visibly artificial, such as `SYNTHETIC OPERATOR ALPHA`,
`SYNTHETIC CAA USER 0001`, and `SYNTHETIC CHECKLIST ITEM 0001`. Email addresses
use only `synthetic.invalid` or `example.invalid`. Phone numbers, street
addresses, passenger/customer/licence identifiers, natural-person names,
free-form copied text, and realistic external domains are forbidden.

Generation may use only this contract, the canonical product specs, the
versioned 86-route catalog, an explicit deterministic seed, and an explicit
clock origin. It must never read repository history, logs, customer exports,
production or operational sources, local address books, shell history, browser
profiles, or home-directory data.

Passwords, TOTP secrets, recovery codes, provider action tokens, access and
refresh tokens, private keys, API keys, and client secrets are forbidden in
fixtures, manifests, logs, mail, documents, evidence, and generated records.

## Loader And Cleanup Boundary

The only loader target is a complete, dedicated, disposable `local-preprod`
namespace containing the application database, isolated Keycloak realm and
database/accounts/client, Mailpit namespace, object bucket/prefix, and
loader-owned queues/jobs. Load, resume, and drop/recreate are separate
single-use authorities. Cleanup may drop/recreate only that exact whole
namespace after ownership and authorization preflight.

Selective deletion by run ID, deletion of individual users, or deletion from
append-only membership, Evidence, configuration, audit, manifest, or history
tables is forbidden. Intent, result, checkpoints, authorization-token hashes,
consumption facts, and cleanup attestations remain in an append-only control
store outside the disposable target.

## Owner Decision Package

On 2026-07-28 the user explicitly confirmed authority to decide for every
named owner group and approved each row individually. The approval references
below are repository-local durable labels for that direct owner directive;
they do not claim an external ticket, meeting, or runtime verification.

| Decision | Owner | Options | Recommended selection | Rationale | Affected tasks | Blocker if unresolved |
|---|---|---|---|---|---|---|
| Invitation channel, expiry, resend | Identity + Security; Product / CAA Operations | Local SMTP/Mailpit 8 h, 24 h, or 72 h; resend reuses or invalidates prior action; bounded or unbounded resend; `VERIFY_EMAIL` for all, selected, or no roles | Authenticated local SMTP/Mailpit, 24 h, resend invalidates the prior action, maximum 3 per 24 h, and `VERIFY_EMAIL` for all roles | Observable, time-bounded delivery without application credentials | 2, 3, 5, 9 | Blocks invitation implementation and all-eight-role first login |
| Auditee MFA | Identity + Security; Product / CAA Operations | Required TOTP for all; risk-based; optional | Require `CONFIGURE_TOTP` for all Auditee memberships | One consistent preprod assurance path and no UI-simulated MFA | 3, 4, 5, 9 | Blocks Auditee activation/session acceptance |
| Recovery and MFA reset | Identity + Security; Privacy / Records | Admin-assisted execute-actions; provider self-service; no recovery | Reasoned Admin-assisted execute-actions, session revocation, bounded expiry, forced re-enrollment | Keeps secrets in Keycloak and makes reset auditable | 3, 4, 5, 9 | Blocks recovery and MFA-reset runtime |
| Suspension/deactivation/reactivation | Product / CAA Operations; Identity + Security; Privacy / Records | Disable-only; suspend plus delete; distinct retained deactivation/reactivation | Temporary suspension, retained deactivation tombstone, explicit reactivation-pending approval | Preserves history while removing future authority | 2, 3, 4, 5, 9 | Blocks lifecycle enums and persistence |
| Organization transfer | Product / CAA Operations; Privacy / Records | Prohibit; immediate transfer; effective-dated transfer | Reasoned effective-dated future-authority transfer, no historical ownership rewrite, session revocation | Protects record identity and organization privacy | 2, 3, 4, 5, 9 | Blocks transfer API and UI |
| Identifier retention/reuse | Privacy / Records / Legal | Immediate reuse; timed tombstone; permanent non-reuse | Minimum retained tombstone and no reuse unless Legal approves a bounded period | Prevents subject and audit-history collision | 2, 3, 6, 8, 9 | Blocks deactivation cleanup and namespace policy |
| Permissible multi-role combinations | Product / CAA Operations; Identity + Security | Any internal combination; explicit allowlist; single role only | Single role initially; later explicit CAA-only allowlist by versioned decision | Least privilege and deterministic route/session authority | 2, 3, 4, 5, 7, 9 | Blocks multi-role provisioning; Auditee/CAA mixing is always forbidden |
| Bootstrap Admin and break-glass | Identity + Security; Operations | Shared bootstrap credential; permanent realm admin; separate one-shot/bootstrap and alarmed break-glass | One-shot bootstrap removed from runtime; separate no-membership break-glass with two-person custody, alarm, audit, and 15-minute window | Prevents a standing application super-credential | 2, 3, 4, 6, 9 | Blocks normal Keycloak administration |
| Least-privilege Keycloak service account | Identity + Security | `realm-admin`; broad management client; fine-grained confidential client | Client credentials with only `query-users`, `view-users`, `manage-users`, and `view-realm`; explicitly deny realm/client/impersonation administration | Supports directory/lifecycle operations without bootstrap-admin | 2, 3, 4, 6, 9 | Blocks Keycloak directory and mutations |
| Provider observation freshness/deadline | Identity + Security; Operations / SRE | 30/60 s, 60/120 s, or 300/600 s max-age/deadline | 30 s heartbeat, 60 s maximum age, 120 s fail-closed deadline | Bounds stale authority while tolerating one missed observation | 2, 3, 4, 9 | Blocks session freshness and outage behavior |
| Profile volumes and resource limits | Product / Domain + QA; Platform / DBA + Security | Current four proposals; reduced laptop tier; staged scale tiers | Accept the exact proposed counts only after measured feasibility; keep all four catalogs complete | Prevents silent workload reduction and protects the host | 6, 7, 8, 9 | Blocks loader implementation and scale qualification |

### Effective Owner Decisions

Every row has status `approved — owner decision recorded` and is effective in
contract `1.0.0`.

| Approval reference | Decision | Effective selection |
|---|---|---|
| `OWNER-DIRECTIVE-2026-07-28-P5T1-01` | Invitation channel, expiry, resend | Authenticated local SMTP/Mailpit; 24-hour expiry; resend invalidates the prior action; maximum three resends per 24 hours; `VERIFY_EMAIL` for all eight roles |
| `OWNER-DIRECTIVE-2026-07-28-P5T1-02` | Auditee MFA | TOTP is optional for Auditee and every other role; `CONFIGURE_TOTP` is not required; provider self-enrollment remains available |
| `OWNER-DIRECTIVE-2026-07-28-P5T1-03` | Recovery and MFA reset | Reasoned Admin-assisted Keycloak execute-actions; 15-minute expiry; revoke sessions before the action; recovery requires `UPDATE_PASSWORD`; MFA reset clears the old enrollment and leaves re-enrollment optional |
| `OWNER-DIRECTIVE-2026-07-28-P5T1-04` | Suspension/deactivation/reactivation | Temporary suspension; retained non-authoritative deactivation tombstone; explicit owner-approved `reactivation-pending`; no automatic expiry or old-session revival |
| `OWNER-DIRECTIVE-2026-07-28-P5T1-05` | Organization transfer | Reasoned future effective-dated transfer; atomic provider organization change; no historical ownership rewrite; fail closed and revoke sessions at transition |
| `OWNER-DIRECTIVE-2026-07-28-P5T1-06` | Identifier retention/reuse | Permanent non-reuse for membership ID, provider subject, username, and login identifier; returning subjects reactivate the retained membership |
| `OWNER-DIRECTIVE-2026-07-28-P5T1-07` | Permissible multi-role combinations | Exactly one role per membership; role changes atomically replace the role; Auditee/CAA mixing remains forbidden |
| `OWNER-DIRECTIVE-2026-07-28-P5T1-08` | Bootstrap Admin and break-glass | One-shot bootstrap removed from runtime; separate no-membership break-glass; two approvals, 15-minute window, alarm, audit, incident, post-use rotation, and session closure |
| `OWNER-DIRECTIVE-2026-07-28-P5T1-09` | Least-privilege Keycloak service account | Confidential client credentials with only `query-users`, `view-users`, `manage-users`, and `view-realm`; realm/client administration, impersonation, and cross-realm access denied |
| `OWNER-DIRECTIVE-2026-07-28-P5T1-10` | Provider observation freshness/deadline | 30-second heartbeat, 60-second maximum age, 120-second denial deadline; not an inactivity timeout; drift or disablement fails closed immediately |
| `OWNER-DIRECTIVE-2026-07-28-P5T1-11` | Profile volumes and resource limits | Exact `1.0.0` manifests; disk 2/20/50/64 GiB; stress 12 GiB memory, 8-hour duration, and 8 GiB object payload; runtime feasibility remains `not run`; silent reduction forbidden |

## Exact Gate Before Task 2 — Satisfied

Task 2 cannot start until all of the following are true:

1. every owner-decision row has an explicit owner approval reference, effective
   value, and versioned contract update;
2. Task 1 acceptance is recorded only after the contract test passes and the
   role/account/profile matrix has no ambiguous implementation value;
3. the combined local predecessor stakeholder disposition is recorded; this
   gate was satisfied on 2026-07-28;
4. the user separately and explicitly authorizes Task 2; this was subsequently
   satisfied; and
5. the known normal-artifact testprofile link, bootstrap-admin credential use,
   directory placeholders, missing lifecycle actions, and absence of a loader
   remain blockers to be fixed and verified in their owning tasks, not claims
   made by this document.

## Current Repository Observations

Task 1 read-only inspection found the following. Tasks 2–4 subsequently
resolved the artifact, service-client, membership-foundation, directory, and
account-lifecycle rows:

- OpenAPI and Go define the same eight roles.
- Authority validation rejects Auditee/CAA organization mixing and enforces the
  approved exactly-one-role policy.
- Lifecycle actions now include `PROVISION`, `UPDATE_ROLES`, `SUSPEND`,
  `REACTIVATE`, `DEACTIVATE`, `TRANSFER_ORGANIZATION`,
  `RESEND_INVITATION`, `RESET_PASSWORD`, `RESET_MFA`, and `FORCE_LOGOUT`,
  guarded by exact expected-membership revisions and reason-required audits.
- Task 2 replaced the session-derived Administration directory with bounded
  provider-account reads and explicit desired membership, lifecycle, profile,
  MFA, required-action, drift, and last-session fields.
- Task 2 replaced API/worker bootstrap-admin password grant credentials with
  the exact least-privilege service client. Task 3 replaced direct credential
  bootstrap with the approved expiring `UPDATE_PASSWORD`/`VERIFY_EMAIL`
  execute-actions invitation, authenticated local delivery, and optional
  provider-derived TOTP enrollment.
- Task 4 binds application sessions to the exact desired-membership revision,
  fresh observed provider authority, and matching OIDC role/organization
  claims. It enforces the approved 30/60/120-second observation boundary,
  revokes old authority after lifecycle changes, and proves live provider MFA,
  required-action, disabled-account, restart, and session-revocation behavior.
  Bootstrap and break-glass identities remain outside application membership;
  actual break-glass use remains blocked without the separately approved alarm
  and incident gate.
- Task 2 split normal and canonical-test API composition and proved that normal
  API/worker/scheduler/migration artifacts do not link testprofile/reset code.
- The canonical test profile has nine principals across eight roles and is
  test-only. It is not the preprod loader.
- The React route source contains 86 unique routes. This contract requires the
  same complete catalog in every data profile but does not claim runtime data
  exists for it.

These are planning observations, not runtime acceptance. No API, Keycloak,
migration, SQLC, frontend, loader, or runtime code was changed by Task 1.

## Machine-Readable Contract

The following fenced JSON is the test-consumed Task 1 contract. Human prose
must not silently override it. Any owner-approved change requires a new
contract/profile version and a corresponding test update.

<!-- PREPROD_IDENTITY_DATA_CONTRACT:BEGIN -->
```json
{
  "schemaVersion": "preprod-identity-data-contract/v1",
  "contractId": "aviasurveil360-preprod-identity-data",
  "contractVersion": "1.0.0",
  "status": "active — Task 1 authorized",
  "runtimeVerification": "not run",
  "registration": {
    "public": false,
    "creationAuthority": "admin-controlled"
  },
  "identity": {
    "roleAuthority": "server-owned-desired-membership",
    "clientAuthorityInput": "reject",
    "roles": [
      { "id": "inspector", "organizationScope": "exact-CAA" },
      { "id": "leadInspector", "organizationScope": "exact-CAA" },
      { "id": "manager", "organizationScope": "exact-CAA" },
      { "id": "finance", "organizationScope": "exact-CAA" },
      { "id": "gm", "organizationScope": "exact-CAA" },
      { "id": "executiveDirector", "organizationScope": "exact-CAA" },
      { "id": "auditee", "organizationScope": "exactly-one-non-CAA" },
      { "id": "admin", "organizationScope": "exact-CAA" }
    ],
    "forbiddenCombinations": [
      { "id": "AUDITEE_WITH_CAA_ROLE", "outcome": "reject" },
      { "id": "AUDITEE_IN_CAA", "outcome": "reject" },
      { "id": "CAA_ROLE_OUTSIDE_CAA", "outcome": "reject" },
      { "id": "MULTIPLE_ORGANIZATIONS", "outcome": "reject" },
      { "id": "DUPLICATE_ROLE", "outcome": "reject" },
      { "id": "UNAPPROVED_MULTI_ROLE_SET", "outcome": "reject" }
    ]
  },
  "stateMachines": {
    "providerAccount": {
      "authority": "keycloak-observation",
      "states": ["absent", "enabled", "disabled", "locked", "unavailable"]
    },
    "desiredMembership": {
      "authority": "aviasurveil-append-only-aggregate",
      "businessKey": "membershipId",
      "revision": "monotonic-positive-integer",
      "states": [
        "requested",
        "invited",
        "active",
        "suspended",
        "deactivated",
        "reactivation-pending"
      ],
      "requiredFields": [
        "membershipId",
        "revision",
        "requestedAt",
        "effectiveAt",
        "actorSubjectId",
        "reason",
        "roles",
        "organizationId",
        "providerObservationId",
        "reconciliationState"
      ]
    },
    "applicationProfile": {
      "authority": "aviasurveil-profile-store",
      "states": ["absent", "pending-link", "linked", "retained-inactive"]
    },
    "invitationRecovery": {
      "authority": "aviasurveil-delivery-facts-and-keycloak-acknowledgement",
      "states": [
        "none",
        "issued",
        "delivered",
        "retryable-failure",
        "terminal-failure",
        "expired",
        "consumed",
        "cancelled"
      ]
    },
    "mfa": {
      "authority": "keycloak-observation",
      "states": [
        "unknown",
        "unenrolled",
        "enrollment-required",
        "enrolled",
        "reset-pending",
        "unavailable"
      ]
    },
    "session": {
      "authority": "aviasurveil-server-session-store",
      "states": [
        "none",
        "active",
        "revocation-pending",
        "revoked",
        "expired",
        "denied-stale-authority"
      ]
    }
  },
  "revisionContract": {
    "initialExpectedMembershipRevision": 0,
    "staleRevisionOutcome": "conflict-with-no-side-effects",
    "successRule": "membership-authority-change-increments-by-one",
    "nonMembershipAuthorityRule": "revision-must-match-but-does-not-increment"
  },
  "authorityMutations": [
    {
      "id": "activate",
      "expectedMembershipRevision": "required",
      "membershipOutcome": "invited-to-active-revision-plus-one",
      "sessionOutcome": "fresh-login-only"
    },
    {
      "id": "suspend",
      "expectedMembershipRevision": "required",
      "membershipOutcome": "suspended-revision-plus-one",
      "sessionOutcome": "revoke-all-sessions-and-offline-grants"
    },
    {
      "id": "deactivate",
      "expectedMembershipRevision": "required",
      "membershipOutcome": "deactivated-revision-plus-one",
      "sessionOutcome": "revoke-all-sessions-and-offline-grants"
    },
    {
      "id": "request-reactivation",
      "expectedMembershipRevision": "required",
      "membershipOutcome": "reactivation-pending-revision-plus-one",
      "sessionOutcome": "deny-login"
    },
    {
      "id": "reactivate",
      "expectedMembershipRevision": "required",
      "membershipOutcome": "invited-or-active-revision-plus-one",
      "sessionOutcome": "fresh-login-only"
    },
    {
      "id": "change-roles",
      "expectedMembershipRevision": "required",
      "membershipOutcome": "replace-exact-role-set-revision-plus-one",
      "sessionOutcome": "revoke-all-sessions-and-offline-grants"
    },
    {
      "id": "transfer-organization",
      "expectedMembershipRevision": "required",
      "membershipOutcome": "future-organization-revision-plus-one-history-unchanged",
      "sessionOutcome": "revoke-all-sessions-and-offline-grants"
    },
    {
      "id": "reset-mfa",
      "expectedMembershipRevision": "required",
      "membershipOutcome": "membership-state-retained",
      "sessionOutcome": "revoke-all-until-fresh-policy-compliant-login"
    },
    {
      "id": "force-logout",
      "expectedMembershipRevision": "required",
      "membershipOutcome": "membership-state-and-revision-retained",
      "sessionOutcome": "revoke-all-sessions-and-offline-grants"
    },
    {
      "id": "resend-invitation",
      "expectedMembershipRevision": "required",
      "membershipOutcome": "membership-state-retained",
      "sessionOutcome": "no-authority-granted"
    },
    {
      "id": "start-recovery",
      "expectedMembershipRevision": "required",
      "membershipOutcome": "membership-state-retained",
      "sessionOutcome": "revoke-all-before-recovery"
    }
  ],
  "invitation": {
    "providerModel": "keycloak-execute-actions",
    "requiredActions": ["UPDATE_PASSWORD"],
    "temporaryPassword": "forbidden",
    "actionTokenStorage": "forbidden",
    "deliveryPolicy": {
      "decisionId": "INVITATION_CHANNEL_EXPIRY_RESEND",
      "status": "approved — owner decision recorded",
      "proposedValue": {
        "channel": "authenticated-local-smtp-mailpit",
        "expirySeconds": 86400,
        "resend": "invalidate-prior-action",
        "maximumResendsPer24Hours": 3
      },
      "effectiveValue": {
        "channel": "authenticated-local-smtp-mailpit",
        "expirySeconds": 86400,
        "resend": "invalidate-prior-action",
        "maximumResendsPer24Hours": 3
      }
    },
    "verifyEmail": {
      "decisionId": "INVITATION_CHANNEL_EXPIRY_RESEND",
      "status": "approved — owner decision recorded",
      "proposedValue": "role-policy-required",
      "proposedByRole": {
        "inspector": true,
        "leadInspector": true,
        "manager": true,
        "finance": true,
        "gm": true,
        "executiveDirector": true,
        "auditee": true,
        "admin": true
      },
      "effectiveValue": "required-all-roles",
      "effectiveByRole": {
        "inspector": true,
        "leadInspector": true,
        "manager": true,
        "finance": true,
        "gm": true,
        "executiveDirector": true,
        "auditee": true,
        "admin": true
      }
    },
    "configureTotp": {
      "decisionId": "AUDITEE_MFA",
      "status": "approved — owner decision recorded",
      "proposedValue": "required-for-all-roles",
      "proposedByRole": {
        "inspector": true,
        "leadInspector": true,
        "manager": true,
        "finance": true,
        "gm": true,
        "executiveDirector": true,
        "auditee": true,
        "admin": true
      },
      "effectiveValue": "optional-all-roles",
      "effectiveByRole": {
        "inspector": false,
        "leadInspector": false,
        "manager": false,
        "finance": false,
        "gm": false,
        "executiveDirector": false,
        "auditee": false,
        "admin": false
      }
    }
  },
  "providerObservation": {
    "decisionId": "PROVIDER_OBSERVATION_FRESHNESS_DEADLINE",
    "status": "approved — owner decision recorded",
    "proposedValue": {
      "heartbeatSeconds": 30,
      "maximumAgeSeconds": 60,
      "reconciliationDeadlineSeconds": 120
    },
    "effectiveValue": {
      "heartbeatSeconds": 30,
      "maximumAgeSeconds": 60,
      "reconciliationDeadlineSeconds": 120,
      "ageIsUserInactivity": false,
      "staleNewLogin": "deny",
      "staleAuthorityMutation": "deny",
      "staleExistingSessions": "revocation-pending-then-revoked-by-deadline",
      "driftOrDisablement": "immediate-fail-closed",
      "recovery": "fresh-exact-observation-and-new-login"
    },
    "failClosedOn": [
      "provider-unavailable",
      "provider-disabled",
      "role-drift",
      "organization-drift",
      "stale-observation"
    ]
  },
  "normalArtifact": {
    "appliesTo": ["api", "worker", "scheduler", "migration", "web-http"],
    "publicRegistration": false,
    "authentication": ["oidc", "secure-http-only-application-session"],
    "forbiddenSurfaces": [
      "seed-route",
      "reset-route",
      "seed-startup-hook",
      "reset-startup-hook",
      "loader-command",
      "canonical-test-subject-header",
      "canonical-test-token-header"
    ],
    "forbiddenImports": [
      "apps/api/internal/testprofile",
      "apps/api/internal/preproddata"
    ],
    "startupDataPolicy": "no-demo-canonical-or-preprod-data"
  },
  "catalogs": {
    "routeCatalog": {
      "source": "apps/web/src/parity/legacy-screen-source.json",
      "exactCount": 86,
      "coverage": "complete"
    },
    "visibleActionCatalog": {
      "source": "tests/parity/behavior-ledger.json",
      "coverage": "complete",
      "proposedExactDispositionCount": 306
    },
    "roleCatalog": {
      "roles": [
        "inspector",
        "leadInspector",
        "manager",
        "finance",
        "gm",
        "executiveDirector",
        "auditee",
        "admin"
      ]
    },
    "lifecycleScenarioCatalog": {
      "scenarios": [
        "planned",
        "active",
        "overdue",
        "returned",
        "rejected",
        "corrected",
        "superseded",
        "reopened",
        "partially-closed",
        "not-closed",
        "authorized-closed",
        "verified-closed"
      ]
    }
  },
  "syntheticData": {
    "piiAllowed": false,
    "secretsAllowed": false,
    "allowedEmailDomains": ["synthetic.invalid", "example.invalid"],
    "exampleEmails": [
      "caa-user-0001@synthetic.invalid",
      "auditee-user-0001@example.invalid"
    ],
    "nameTemplates": [
      "SYNTHETIC CAA USER {NNNN}",
      "SYNTHETIC AUDITEE USER {NNNN}",
      "SYNTHETIC OPERATOR {ALPHA}"
    ],
    "allowedSources": [
      "this-versioned-contract",
      "canonical-product-specs",
      "versioned-86-route-catalog",
      "explicit-deterministic-seed",
      "explicit-clock-origin"
    ],
    "forbiddenSources": [
      "repository-history",
      "logs",
      "customer-exports",
      "production-data",
      "operational-data",
      "local-address-books",
      "shell-history",
      "browser-profiles",
      "home-directory-data"
    ],
    "forbiddenFields": [
      "password",
      "totpSecret",
      "recoveryCode",
      "providerActionToken",
      "accessToken",
      "refreshToken",
      "privateKey",
      "apiKey",
      "clientSecret"
    ]
  },
  "loaderBoundary": {
    "environment": "local-preprod",
    "target": "whole-disposable-namespace",
    "components": [
      "application-database",
      "isolated-keycloak-realm-and-database-accounts-client",
      "mailpit-namespace",
      "object-bucket-prefix",
      "loader-owned-queues-and-jobs"
    ],
    "selectiveAppendOnlyDeletion": "forbidden",
    "individualUserDeletionForReplay": "forbidden",
    "retainedControlStore": "outside-target",
    "operationAuthorities": [
      "LOAD_EMPTY_TARGET",
      "RESUME_RUN",
      "DROP_RECREATE_TARGET"
    ]
  },
  "dataProfiles": {
    "decisionId": "PROFILE_VOLUMES_RESOURCE_LIMITS",
    "digestCanonicalization": "RFC8785-canonical-JSON-lines-sorted-by-business-key-revision",
    "defaultDistributionFamilies": [
      "applicationProfiles",
      "recoveryRequests",
      "sessions",
      "offlineGrants",
      "surveillancePlans",
      "planningApprovals",
      "assignments",
      "checklistTemplates",
      "checklistTemplateVersions",
      "checklistQuestions",
      "inspectionPackages",
      "checklistResponses",
      "evidenceReferences",
      "reviewDecisions",
      "communications",
      "notifications",
      "auditEvents",
      "outboxMessages",
      "deliveryJobs",
      "scannerJobs",
      "renderJobs",
      "objects",
      "objectVersions",
      "calendarRecords",
      "syncChanges"
    ],
    "familyDefinitions": [
      {
        "id": "organizations",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["organizationId", "organizationType"] }
      },
      {
        "id": "providerAccounts",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["providerSubjectId", "membershipId", "observedState"] }
      },
      {
        "id": "desiredMembershipVersions",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["membershipId", "revision", "organizationId", "roles", "state"] }
      },
      {
        "id": "applicationProfiles",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["profileId", "membershipId", "organizationId"] }
      },
      {
        "id": "invitations",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["invitationId", "membershipId", "deliveryId", "state"] }
      },
      {
        "id": "recoveryRequests",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["recoveryId", "membershipId", "requestedAt", "state"] }
      },
      {
        "id": "mfaEnrollments",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["membershipId", "providerSubjectId", "observedState"] }
      },
      {
        "id": "sessions",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["sessionId", "membershipId", "membershipRevision", "state"] }
      },
      {
        "id": "offlineGrants",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["grantId", "sessionId", "membershipRevision", "expiresAt"] }
      },
      {
        "id": "surveillancePlans",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["planningItemId", "organizationId", "revision", "state"] }
      },
      {
        "id": "planningApprovals",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["decisionId", "planningItemId", "actorRole", "revision"] }
      },
      {
        "id": "audits",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["auditId", "planningItemId", "organizationId", "revision"] }
      },
      {
        "id": "assignments",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["assignmentId", "auditId", "membershipId", "questionId"] }
      },
      {
        "id": "checklistTemplates",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["templateId", "ownerOrganizationId"] }
      },
      {
        "id": "checklistTemplateVersions",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["templateVersionId", "templateId", "revision", "predecessorId"] }
      },
      {
        "id": "checklistQuestions",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["questionId", "templateVersionId", "sectionId"] }
      },
      {
        "id": "inspectionPackages",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["packageId", "auditId", "templateVersionId"] }
      },
      {
        "id": "checklistResponses",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["responseId", "auditId", "questionId", "membershipId", "revision"] }
      },
      {
        "id": "potentialFindings",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["potentialFindingId", "responseId", "auditId", "state"] }
      },
      {
        "id": "findings",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["findingId", "potentialFindingId", "auditId", "organizationId", "state"] }
      },
      {
        "id": "capRevisions",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["capRevisionId", "findingId", "revision", "predecessorId", "state"] }
      },
      {
        "id": "evidenceReferences",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["evidenceId", "findingId", "capRevisionId"] }
      },
      {
        "id": "evidenceVersions",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["evidenceVersionId", "evidenceId", "revision", "objectVersionId", "state"] }
      },
      {
        "id": "reviewDecisions",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["reviewDecisionId", "recordId", "recordRevision", "actorMembershipId", "decision"] }
      },
      {
        "id": "reportVersions",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["reportVersionId", "auditId", "revision", "predecessorId", "state"] }
      },
      {
        "id": "communications",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["communicationId", "organizationId", "senderMembershipId", "recipientMembershipId"] }
      },
      {
        "id": "notifications",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["notificationId", "organizationId", "recipientMembershipId", "eventId"] }
      },
      {
        "id": "auditEvents",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["eventId", "entityType", "entityId", "entityRevision", "actorMembershipId"] }
      },
      {
        "id": "outboxMessages",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["outboxId", "aggregateType", "aggregateId", "aggregateRevision"] }
      },
      {
        "id": "deliveryJobs",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["deliveryJobId", "outboxId", "notificationId", "state"] }
      },
      {
        "id": "scannerJobs",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["scannerJobId", "objectVersionId", "state"] }
      },
      {
        "id": "renderJobs",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["renderJobId", "reportVersionId", "objectVersionId", "state"] }
      },
      {
        "id": "objects",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["objectId", "ownerOrganizationId", "recordType", "recordId"] }
      },
      {
        "id": "objectVersions",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["objectVersionId", "objectId", "revision", "contentDigest"] }
      },
      {
        "id": "calendarRecords",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["calendarRecordId", "organizationId", "recordType", "recordId"] }
      },
      {
        "id": "syncChanges",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["changeId", "subjectMembershipId", "entityType", "entityId", "entityRevision"] }
      },
      {
        "id": "routeDispositions",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["routeAuditId", "role", "disposition", "scenarioId"] }
      },
      {
        "id": "visibleActionDispositions",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["routeAuditId", "actionId", "role", "disposition"] }
      },
      {
        "id": "identityLifecycleCases",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["scenarioId", "membershipId", "state", "providerState", "sessionState"] }
      },
      {
        "id": "lifecycleScenarioCases",
        "relationshipDigest": { "required": true, "algorithm": "sha256", "placeholderAllowed": false, "tupleFields": ["scenarioId", "recordType", "recordId", "lifecycleState"] }
      }
    ],
    "profiles": [
      {
        "name": "smoke",
        "version": "1.0.0",
        "status": "approved — owner decision recorded",
        "implementationAllowed": false,
        "changePolicy": "new-version-required",
        "catalogs": {
          "routeCount": 86,
          "visibleActionCoverage": "complete",
          "roles": ["inspector", "leadInspector", "manager", "finance", "gm", "executiveDirector", "auditee", "admin"],
          "lifecycleScenarios": ["planned", "active", "overdue", "returned", "rejected", "corrected", "superseded", "reopened", "partially-closed", "not-closed", "authorized-closed", "verified-closed"]
        },
        "resourceEnvelope": {
          "seedRequired": true,
          "clockOrigin": "2026-01-01T00:00:00Z",
          "identityNamespace": "synthetic-smoke-v1",
          "cpuCores": 2,
          "memoryMiB": 1024,
          "diskMiB": 2048,
          "objectBytes": 134217728,
          "durationSeconds": 120,
          "cleanupSeconds": 60
        },
        "expectedCounts": {
          "organizations": 3,
          "providerAccounts": 9,
          "desiredMembershipVersions": 18,
          "applicationProfiles": 9,
          "invitations": 6,
          "recoveryRequests": 2,
          "mfaEnrollments": 9,
          "sessions": 18,
          "offlineGrants": 4,
          "surveillancePlans": 4,
          "planningApprovals": 12,
          "audits": 2,
          "assignments": 3,
          "checklistTemplates": 4,
          "checklistTemplateVersions": 6,
          "checklistQuestions": 24,
          "inspectionPackages": 2,
          "checklistResponses": 24,
          "potentialFindings": 12,
          "findings": 8,
          "capRevisions": 12,
          "evidenceReferences": 8,
          "evidenceVersions": 16,
          "reviewDecisions": 16,
          "reportVersions": 6,
          "communications": 16,
          "notifications": 24,
          "auditEvents": 250,
          "outboxMessages": 80,
          "deliveryJobs": 48,
          "scannerJobs": 16,
          "renderJobs": 6,
          "objects": 22,
          "objectVersions": 24,
          "calendarRecords": 20,
          "syncChanges": 120,
          "routeDispositions": 86,
          "visibleActionDispositions": 306,
          "identityLifecycleCases": 18,
          "lifecycleScenarioCases": 12
        },
        "exactDistributions": {
          "organizations": { "caa": 1, "auditee": 2 },
          "providerAccounts": { "inspector": 1, "leadInspector": 1, "manager": 1, "finance": 1, "gm": 1, "executiveDirector": 1, "auditee": 2, "admin": 1 },
          "desiredMembershipVersions": { "requested": 2, "invited": 2, "active": 8, "suspended": 2, "deactivated": 2, "reactivation-pending": 2 },
          "invitations": { "issued": 1, "delivered": 1, "retryable-failure": 1, "expired": 1, "consumed": 1, "cancelled": 1 },
          "mfaEnrollments": { "enrolled": 5, "enrollment-required": 0, "reset-pending": 1, "unenrolled": 3 },
          "audits": { "planned": 1, "active": 1, "overdue": 0, "verified-closed": 0 },
          "potentialFindings": { "pending": 4, "returned": 2, "rejected": 2, "corrected": 2, "converted": 2 },
          "findings": { "open": 1, "overdue": 1, "reopened": 1, "partially-closed": 1, "not-closed": 1, "authorized-closed": 1, "verified-closed": 2 },
          "capRevisions": { "draft": 2, "submitted": 2, "returned": 2, "rejected": 2, "corrected": 2, "superseded": 1, "accepted": 1 },
          "evidenceVersions": { "uploaded": 4, "returned": 2, "rejected": 2, "corrected": 2, "superseded": 2, "accepted": 4 },
          "reportVersions": { "draft": 1, "returned": 1, "rejected": 1, "corrected": 1, "issued": 2 },
          "routeDispositions": { "authorized-data": 60, "intentional-empty": 10, "denied": 16 },
          "visibleActionDispositions": { "executable": 200, "disabled-by-role": 50, "disabled-by-state": 56 },
          "identityLifecycleCases": { "requested": 1, "invited": 1, "active": 5, "suspended": 1, "deactivated": 1, "reactivation-pending": 1, "role-changed": 1, "transferred": 1, "mfa-reset": 1, "forced-logout": 1, "invitation-expired": 1, "provider-unavailable": 1, "provider-drift": 1, "recovered": 1 },
          "lifecycleScenarioCases": { "planned": 1, "active": 1, "overdue": 1, "returned": 1, "rejected": 1, "corrected": 1, "superseded": 1, "reopened": 1, "partially-closed": 1, "not-closed": 1, "authorized-closed": 1, "verified-closed": 1 }
        }
      },
      {
        "name": "acceptance",
        "version": "1.0.0",
        "status": "approved — owner decision recorded",
        "implementationAllowed": false,
        "changePolicy": "new-version-required",
        "catalogs": {
          "routeCount": 86,
          "visibleActionCoverage": "complete",
          "roles": ["inspector", "leadInspector", "manager", "finance", "gm", "executiveDirector", "auditee", "admin"],
          "lifecycleScenarios": ["planned", "active", "overdue", "returned", "rejected", "corrected", "superseded", "reopened", "partially-closed", "not-closed", "authorized-closed", "verified-closed"]
        },
        "resourceEnvelope": {
          "seedRequired": true,
          "clockOrigin": "2026-01-01T00:00:00Z",
          "identityNamespace": "synthetic-acceptance-v1",
          "cpuCores": 4,
          "memoryMiB": 4096,
          "diskMiB": 20480,
          "objectBytes": 2147483648,
          "durationSeconds": 1200,
          "cleanupSeconds": 600
        },
        "expectedCounts": {
          "organizations": 25,
          "providerAccounts": 250,
          "desiredMembershipVersions": 350,
          "applicationProfiles": 250,
          "invitations": 100,
          "recoveryRequests": 25,
          "mfaEnrollments": 250,
          "sessions": 500,
          "offlineGrants": 125,
          "surveillancePlans": 1250,
          "planningApprovals": 4000,
          "audits": 1000,
          "assignments": 1500,
          "checklistTemplates": 50,
          "checklistTemplateVersions": 100,
          "checklistQuestions": 500,
          "inspectionPackages": 1000,
          "checklistResponses": 10000,
          "potentialFindings": 4500,
          "findings": 3000,
          "capRevisions": 4500,
          "evidenceReferences": 3000,
          "evidenceVersions": 6000,
          "reviewDecisions": 6000,
          "reportVersions": 2000,
          "communications": 8000,
          "notifications": 12000,
          "auditEvents": 100000,
          "outboxMessages": 30000,
          "deliveryJobs": 20000,
          "scannerJobs": 6000,
          "renderJobs": 2000,
          "objects": 8000,
          "objectVersions": 9000,
          "calendarRecords": 5000,
          "syncChanges": 50000,
          "routeDispositions": 86,
          "visibleActionDispositions": 306,
          "identityLifecycleCases": 250,
          "lifecycleScenarioCases": 1200
        },
        "exactDistributions": {
          "organizations": { "caa": 1, "auditee": 24 },
          "providerAccounts": { "inspector": 70, "leadInspector": 25, "manager": 25, "finance": 20, "gm": 20, "executiveDirector": 10, "auditee": 70, "admin": 10 },
          "desiredMembershipVersions": { "requested": 35, "invited": 35, "active": 200, "suspended": 30, "deactivated": 30, "reactivation-pending": 20 },
          "invitations": { "issued": 15, "delivered": 15, "retryable-failure": 10, "expired": 15, "consumed": 35, "cancelled": 10 },
          "mfaEnrollments": { "enrolled": 160, "enrollment-required": 0, "reset-pending": 20, "unenrolled": 70 },
          "audits": { "planned": 200, "active": 400, "overdue": 100, "verified-closed": 300 },
          "potentialFindings": { "pending": 900, "returned": 600, "rejected": 600, "corrected": 600, "converted": 1800 },
          "findings": { "open": 900, "overdue": 450, "reopened": 300, "partially-closed": 450, "not-closed": 300, "authorized-closed": 150, "verified-closed": 450 },
          "capRevisions": { "draft": 450, "submitted": 900, "returned": 675, "rejected": 450, "corrected": 675, "superseded": 450, "accepted": 900 },
          "evidenceVersions": { "uploaded": 1200, "returned": 900, "rejected": 600, "corrected": 900, "superseded": 600, "accepted": 1800 },
          "reportVersions": { "draft": 400, "returned": 300, "rejected": 200, "corrected": 300, "issued": 800 },
          "routeDispositions": { "authorized-data": 70, "intentional-empty": 8, "denied": 8 },
          "visibleActionDispositions": { "executable": 240, "disabled-by-role": 30, "disabled-by-state": 36 },
          "identityLifecycleCases": { "requested": 25, "invited": 25, "active": 100, "suspended": 20, "deactivated": 20, "reactivation-pending": 10, "role-changed": 10, "transferred": 10, "mfa-reset": 10, "forced-logout": 5, "invitation-expired": 5, "provider-unavailable": 3, "provider-drift": 3, "recovered": 4 },
          "lifecycleScenarioCases": { "planned": 100, "active": 100, "overdue": 100, "returned": 100, "rejected": 100, "corrected": 100, "superseded": 100, "reopened": 100, "partially-closed": 100, "not-closed": 100, "authorized-closed": 100, "verified-closed": 100 }
        }
      },
      {
        "name": "realistic",
        "version": "1.0.0",
        "status": "approved — owner decision recorded",
        "implementationAllowed": false,
        "changePolicy": "new-version-required",
        "catalogs": {
          "routeCount": 86,
          "visibleActionCoverage": "complete",
          "roles": ["inspector", "leadInspector", "manager", "finance", "gm", "executiveDirector", "auditee", "admin"],
          "lifecycleScenarios": ["planned", "active", "overdue", "returned", "rejected", "corrected", "superseded", "reopened", "partially-closed", "not-closed", "authorized-closed", "verified-closed"]
        },
        "resourceEnvelope": {
          "seedRequired": true,
          "clockOrigin": "2026-01-01T00:00:00Z",
          "identityNamespace": "synthetic-realistic-v1",
          "cpuCores": 8,
          "memoryMiB": 12288,
          "diskMiB": 51200,
          "objectBytes": 21474836480,
          "durationSeconds": 7200,
          "cleanupSeconds": 2700
        },
        "expectedCounts": {
          "organizations": 100,
          "providerAccounts": 2000,
          "desiredMembershipVersions": 3000,
          "applicationProfiles": 2000,
          "invitations": 800,
          "recoveryRequests": 200,
          "mfaEnrollments": 2000,
          "sessions": 4000,
          "offlineGrants": 1000,
          "surveillancePlans": 25000,
          "planningApprovals": 80000,
          "audits": 20000,
          "assignments": 30000,
          "checklistTemplates": 200,
          "checklistTemplateVersions": 400,
          "checklistQuestions": 5000,
          "inspectionPackages": 20000,
          "checklistResponses": 250000,
          "potentialFindings": 90000,
          "findings": 60000,
          "capRevisions": 100000,
          "evidenceReferences": 100000,
          "evidenceVersions": 200000,
          "reviewDecisions": 200000,
          "reportVersions": 75000,
          "communications": 400000,
          "notifications": 600000,
          "auditEvents": 5000000,
          "outboxMessages": 1500000,
          "deliveryJobs": 1000000,
          "scannerJobs": 200000,
          "renderJobs": 75000,
          "objects": 275000,
          "objectVersions": 350000,
          "calendarRecords": 100000,
          "syncChanges": 2500000,
          "routeDispositions": 86,
          "visibleActionDispositions": 306,
          "identityLifecycleCases": 2000,
          "lifecycleScenarioCases": 24000
        },
        "exactDistributions": {
          "organizations": { "caa": 1, "auditee": 99 },
          "providerAccounts": { "inspector": 600, "leadInspector": 200, "manager": 200, "finance": 150, "gm": 150, "executiveDirector": 100, "auditee": 500, "admin": 100 },
          "desiredMembershipVersions": { "requested": 300, "invited": 300, "active": 1800, "suspended": 200, "deactivated": 250, "reactivation-pending": 150 },
          "invitations": { "issued": 120, "delivered": 120, "retryable-failure": 80, "expired": 120, "consumed": 280, "cancelled": 80 },
          "mfaEnrollments": { "enrolled": 1300, "enrollment-required": 0, "reset-pending": 150, "unenrolled": 550 },
          "audits": { "planned": 4000, "active": 8000, "overdue": 2000, "verified-closed": 6000 },
          "potentialFindings": { "pending": 18000, "returned": 12000, "rejected": 12000, "corrected": 12000, "converted": 36000 },
          "findings": { "open": 18000, "overdue": 9000, "reopened": 6000, "partially-closed": 9000, "not-closed": 6000, "authorized-closed": 3000, "verified-closed": 9000 },
          "capRevisions": { "draft": 10000, "submitted": 20000, "returned": 15000, "rejected": 10000, "corrected": 15000, "superseded": 10000, "accepted": 20000 },
          "evidenceVersions": { "uploaded": 40000, "returned": 30000, "rejected": 20000, "corrected": 30000, "superseded": 20000, "accepted": 60000 },
          "reportVersions": { "draft": 15000, "returned": 11250, "rejected": 7500, "corrected": 11250, "issued": 30000 },
          "routeDispositions": { "authorized-data": 72, "intentional-empty": 6, "denied": 8 },
          "visibleActionDispositions": { "executable": 250, "disabled-by-role": 24, "disabled-by-state": 32 },
          "identityLifecycleCases": { "requested": 200, "invited": 200, "active": 800, "suspended": 160, "deactivated": 160, "reactivation-pending": 80, "role-changed": 80, "transferred": 80, "mfa-reset": 80, "forced-logout": 40, "invitation-expired": 40, "provider-unavailable": 24, "provider-drift": 24, "recovered": 32 },
          "lifecycleScenarioCases": { "planned": 2000, "active": 2000, "overdue": 2000, "returned": 2000, "rejected": 2000, "corrected": 2000, "superseded": 2000, "reopened": 2000, "partially-closed": 2000, "not-closed": 2000, "authorized-closed": 2000, "verified-closed": 2000 }
        }
      },
      {
        "name": "stress",
        "version": "1.0.0",
        "status": "approved — owner decision recorded",
        "implementationAllowed": false,
        "changePolicy": "new-version-required",
        "catalogs": {
          "routeCount": 86,
          "visibleActionCoverage": "complete",
          "roles": ["inspector", "leadInspector", "manager", "finance", "gm", "executiveDirector", "auditee", "admin"],
          "lifecycleScenarios": ["planned", "active", "overdue", "returned", "rejected", "corrected", "superseded", "reopened", "partially-closed", "not-closed", "authorized-closed", "verified-closed"]
        },
        "resourceEnvelope": {
          "seedRequired": true,
          "clockOrigin": "2026-01-01T00:00:00Z",
          "identityNamespace": "synthetic-stress-v1",
          "cpuCores": 12,
          "memoryMiB": 12288,
          "diskMiB": 65536,
          "objectBytes": 8589934592,
          "durationSeconds": 28800,
          "cleanupSeconds": 5400
        },
        "expectedCounts": {
          "organizations": 200,
          "providerAccounts": 4000,
          "desiredMembershipVersions": 6000,
          "applicationProfiles": 4000,
          "invitations": 1600,
          "recoveryRequests": 400,
          "mfaEnrollments": 4000,
          "sessions": 8000,
          "offlineGrants": 2000,
          "surveillancePlans": 50000,
          "planningApprovals": 160000,
          "audits": 40000,
          "assignments": 60000,
          "checklistTemplates": 400,
          "checklistTemplateVersions": 800,
          "checklistQuestions": 10000,
          "inspectionPackages": 40000,
          "checklistResponses": 500000,
          "potentialFindings": 180000,
          "findings": 120000,
          "capRevisions": 200000,
          "evidenceReferences": 200000,
          "evidenceVersions": 400000,
          "reviewDecisions": 400000,
          "reportVersions": 150000,
          "communications": 800000,
          "notifications": 1200000,
          "auditEvents": 10000000,
          "outboxMessages": 3000000,
          "deliveryJobs": 2000000,
          "scannerJobs": 400000,
          "renderJobs": 150000,
          "objects": 550000,
          "objectVersions": 700000,
          "calendarRecords": 200000,
          "syncChanges": 5000000,
          "routeDispositions": 86,
          "visibleActionDispositions": 306,
          "identityLifecycleCases": 4000,
          "lifecycleScenarioCases": 48000
        },
        "exactDistributions": {
          "organizations": { "caa": 1, "auditee": 199 },
          "providerAccounts": { "inspector": 1200, "leadInspector": 400, "manager": 400, "finance": 300, "gm": 300, "executiveDirector": 200, "auditee": 1000, "admin": 200 },
          "desiredMembershipVersions": { "requested": 600, "invited": 600, "active": 3600, "suspended": 400, "deactivated": 500, "reactivation-pending": 300 },
          "invitations": { "issued": 240, "delivered": 240, "retryable-failure": 160, "expired": 240, "consumed": 560, "cancelled": 160 },
          "mfaEnrollments": { "enrolled": 2600, "enrollment-required": 0, "reset-pending": 300, "unenrolled": 1100 },
          "audits": { "planned": 8000, "active": 16000, "overdue": 4000, "verified-closed": 12000 },
          "potentialFindings": { "pending": 36000, "returned": 24000, "rejected": 24000, "corrected": 24000, "converted": 72000 },
          "findings": { "open": 36000, "overdue": 18000, "reopened": 12000, "partially-closed": 18000, "not-closed": 12000, "authorized-closed": 6000, "verified-closed": 18000 },
          "capRevisions": { "draft": 20000, "submitted": 40000, "returned": 30000, "rejected": 20000, "corrected": 30000, "superseded": 20000, "accepted": 40000 },
          "evidenceVersions": { "uploaded": 80000, "returned": 60000, "rejected": 40000, "corrected": 60000, "superseded": 40000, "accepted": 120000 },
          "reportVersions": { "draft": 30000, "returned": 22500, "rejected": 15000, "corrected": 22500, "issued": 60000 },
          "routeDispositions": { "authorized-data": 72, "intentional-empty": 6, "denied": 8 },
          "visibleActionDispositions": { "executable": 250, "disabled-by-role": 24, "disabled-by-state": 32 },
          "identityLifecycleCases": { "requested": 400, "invited": 400, "active": 1600, "suspended": 320, "deactivated": 320, "reactivation-pending": 160, "role-changed": 160, "transferred": 160, "mfa-reset": 160, "forced-logout": 80, "invitation-expired": 80, "provider-unavailable": 48, "provider-drift": 48, "recovered": 64 },
          "lifecycleScenarioCases": { "planned": 4000, "active": 4000, "overdue": 4000, "returned": 4000, "rejected": 4000, "corrected": 4000, "superseded": 4000, "reopened": 4000, "partially-closed": 4000, "not-closed": 4000, "authorized-closed": 4000, "verified-closed": 4000 }
        }
      }
    ]
  },
  "ownerDecisions": [
    {
      "id": "INVITATION_CHANNEL_EXPIRY_RESEND",
      "owner": "Identity + Security; Product / CAA Operations",
      "options": ["local SMTP/Mailpit with 8-hour expiry and role-selected VERIFY_EMAIL", "local SMTP/Mailpit with 24-hour expiry and role-selected VERIFY_EMAIL", "local SMTP/Mailpit with 72-hour expiry and role-selected VERIFY_EMAIL"],
      "recommended": "authenticated local SMTP/Mailpit, 24-hour expiry, resend invalidates prior action, maximum 3 per 24 hours, VERIFY_EMAIL for all roles",
      "rationale": "observable bounded delivery without application credentials",
      "affectedTasks": [2, 3, 5, 9],
      "blockerIfUnresolved": "invitation implementation and all-eight-role first login",
      "status": "approved — owner decision recorded",
      "approved": true,
      "frozen": true,
      "approvedAt": "2026-07-28",
      "approvalReference": "OWNER-DIRECTIVE-2026-07-28-P5T1-01",
      "effectiveContractVersion": "1.0.0",
      "implementationValue": {
        "channel": "authenticated-local-smtp-mailpit",
        "expirySeconds": 86400,
        "resend": "invalidate-prior-action",
        "maximumResendsPer24Hours": 3,
        "verifyEmailByRole": {
          "inspector": true,
          "leadInspector": true,
          "manager": true,
          "finance": true,
          "gm": true,
          "executiveDirector": true,
          "auditee": true,
          "admin": true
        }
      }
    },
    {
      "id": "AUDITEE_MFA",
      "owner": "Identity + Security; Product / CAA Operations",
      "options": ["required TOTP for all Auditees", "risk-based TOTP", "optional TOTP"],
      "recommended": "required CONFIGURE_TOTP for every Auditee membership",
      "rationale": "one consistent preprod assurance path with provider enforcement",
      "affectedTasks": [3, 4, 5, 9],
      "blockerIfUnresolved": "Auditee activation and session acceptance",
      "status": "approved — owner decision recorded",
      "approved": true,
      "frozen": true,
      "approvedAt": "2026-07-28",
      "approvalReference": "OWNER-DIRECTIVE-2026-07-28-P5T1-02",
      "effectiveContractVersion": "1.0.0",
      "implementationValue": {
        "policy": "optional-all-roles",
        "configureTotpByRole": {
          "inspector": false,
          "leadInspector": false,
          "manager": false,
          "finance": false,
          "gm": false,
          "executiveDirector": false,
          "auditee": false,
          "admin": false
        },
        "selfEnrollment": true
      }
    },
    {
      "id": "RECOVERY_AND_MFA_RESET",
      "owner": "Identity + Security; Privacy / Records",
      "options": ["Admin-assisted execute-actions", "provider self-service", "no recovery"],
      "recommended": "reasoned Admin-assisted execute-actions with session revocation, bounded expiry, and forced re-enrollment",
      "rationale": "keeps secrets in Keycloak and makes reset auditable",
      "affectedTasks": [3, 4, 5, 9],
      "blockerIfUnresolved": "recovery and MFA-reset runtime",
      "status": "approved — owner decision recorded",
      "approved": true,
      "frozen": true,
      "approvedAt": "2026-07-28",
      "approvalReference": "OWNER-DIRECTIVE-2026-07-28-P5T1-03",
      "effectiveContractVersion": "1.0.0",
      "implementationValue": {
        "initiation": "reasoned-admin-assisted",
        "providerModel": "keycloak-execute-actions",
        "expirySeconds": 900,
        "revokeSessionsBeforeAction": true,
        "accountRecoveryRequiredActions": ["UPDATE_PASSWORD"],
        "mfaReset": "clear-existing-enrollment",
        "mfaReenrollment": "optional",
        "applicationSecretHandling": "forbidden"
      }
    },
    {
      "id": "SUSPENSION_DEACTIVATION_REACTIVATION",
      "owner": "Product / CAA Operations; Identity + Security; Privacy / Records",
      "options": ["disable-only", "suspend plus delete", "distinct retained deactivation and reactivation"],
      "recommended": "temporary suspension, retained deactivation tombstone, explicit reactivation-pending approval",
      "rationale": "preserves history while removing future authority",
      "affectedTasks": [2, 3, 4, 5, 9],
      "blockerIfUnresolved": "lifecycle enums and persistence",
      "status": "approved — owner decision recorded",
      "approved": true,
      "frozen": true,
      "approvedAt": "2026-07-28",
      "approvalReference": "OWNER-DIRECTIVE-2026-07-28-P5T1-04",
      "effectiveContractVersion": "1.0.0",
      "implementationValue": {
        "suspension": "temporary-until-explicit-reactivation",
        "deactivation": "retained-tombstone-no-future-authority",
        "reactivation": "owner-approved-reactivation-pending",
        "automaticExpiry": false,
        "revokeSessionsAndOfflineGrants": true,
        "reviveOldSessions": false
      }
    },
    {
      "id": "ORGANIZATION_TRANSFER",
      "owner": "Product / CAA Operations; Privacy / Records",
      "options": ["prohibit transfer", "immediate transfer", "effective-dated transfer"],
      "recommended": "reasoned effective-dated future-authority transfer with no historical ownership rewrite and session revocation",
      "rationale": "protects record identity and organization privacy",
      "affectedTasks": [2, 3, 4, 5, 9],
      "blockerIfUnresolved": "transfer API and UI",
      "status": "approved — owner decision recorded",
      "approved": true,
      "frozen": true,
      "approvedAt": "2026-07-28",
      "approvalReference": "OWNER-DIRECTIVE-2026-07-28-P5T1-05",
      "effectiveContractVersion": "1.0.0",
      "implementationValue": {
        "mode": "reasoned-effective-dated",
        "requireFutureEffectiveAt": true,
        "historicalOwnershipRewrite": "forbidden",
        "providerOrganizationChange": "atomic-at-effective-time",
        "reconciliationFailure": "fail-closed",
        "revokeSessionsAndOfflineGrants": true,
        "forbiddenRoleOrganizationCombination": "reject"
      }
    },
    {
      "id": "IDENTIFIER_RETENTION_REUSE",
      "owner": "Privacy / Records / Legal",
      "options": ["immediate reuse", "timed tombstone", "permanent non-reuse"],
      "recommended": "minimum retained tombstone and no reuse unless Legal approves a bounded period",
      "rationale": "prevents subject and audit-history collision",
      "affectedTasks": [2, 3, 6, 8, 9],
      "blockerIfUnresolved": "deactivation cleanup and namespace policy",
      "status": "approved — owner decision recorded",
      "approved": true,
      "frozen": true,
      "approvedAt": "2026-07-28",
      "approvalReference": "OWNER-DIRECTIVE-2026-07-28-P5T1-06",
      "effectiveContractVersion": "1.0.0",
      "implementationValue": {
        "policy": "permanent-non-reuse",
        "identifiers": [
          "membershipId",
          "providerSubject",
          "username",
          "loginIdentifier"
        ],
        "reactivationUsesExistingMembership": true,
        "automaticRelease": "forbidden"
      }
    },
    {
      "id": "PERMISSIBLE_MULTI_ROLE_COMBINATIONS",
      "owner": "Product / CAA Operations; Identity + Security",
      "options": ["any internal combination", "explicit CAA-only allowlist", "single role only"],
      "recommended": "single role initially with later explicit CAA-only allowlist by versioned decision",
      "rationale": "least privilege and deterministic route and session authority",
      "affectedTasks": [2, 3, 4, 5, 7, 9],
      "blockerIfUnresolved": "multi-role provisioning; Auditee and CAA mixing remains forbidden",
      "status": "approved — owner decision recorded",
      "approved": true,
      "frozen": true,
      "approvedAt": "2026-07-28",
      "approvalReference": "OWNER-DIRECTIVE-2026-07-28-P5T1-07",
      "effectiveContractVersion": "1.0.0",
      "implementationValue": {
        "maximumRolesPerMembership": 1,
        "roleChange": "atomic-replacement",
        "auditeeCaaMix": "forbidden",
        "futureCaaCombinationPolicy": "new-versioned-allowlist-required"
      }
    },
    {
      "id": "BOOTSTRAP_ADMIN_BREAK_GLASS",
      "owner": "Identity + Security; Operations",
      "options": ["shared bootstrap credential", "permanent realm admin", "separate one-shot bootstrap and alarmed break-glass"],
      "recommended": "one-shot bootstrap removed from runtime plus separate no-membership break-glass with two-person custody, alarm, audit, and 15-minute window",
      "rationale": "prevents a standing application super-credential",
      "affectedTasks": [2, 3, 4, 6, 9],
      "blockerIfUnresolved": "normal Keycloak administration",
      "status": "approved — owner decision recorded",
      "approved": true,
      "frozen": true,
      "approvedAt": "2026-07-28",
      "approvalReference": "OWNER-DIRECTIVE-2026-07-28-P5T1-08",
      "effectiveContractVersion": "1.0.0",
      "implementationValue": {
        "bootstrap": "one-shot-removed-from-runtime",
        "breakGlassApplicationMembership": false,
        "custodyApprovals": 2,
        "windowSeconds": 900,
        "alarmRequired": true,
        "auditRequired": true,
        "incidentRequired": true,
        "rotateAfterUse": true,
        "closeSessionsAfterUse": true,
        "sharedCredential": "forbidden",
        "permanentRealmAdmin": "forbidden"
      }
    },
    {
      "id": "KEYCLOAK_SERVICE_ACCOUNT_PRIVILEGES",
      "owner": "Identity + Security",
      "options": ["realm-admin", "broad management client", "fine-grained confidential client"],
      "recommended": "client credentials with query-users, view-users, manage-users, and view-realm only; deny realm/client/impersonation administration",
      "rationale": "supports directory and lifecycle operations without bootstrap-admin",
      "affectedTasks": [2, 3, 4, 6, 9],
      "blockerIfUnresolved": "Keycloak directory and mutations",
      "status": "approved — owner decision recorded",
      "approved": true,
      "frozen": true,
      "approvedAt": "2026-07-28",
      "approvalReference": "OWNER-DIRECTIVE-2026-07-28-P5T1-09",
      "effectiveContractVersion": "1.0.0",
      "implementationValue": {
        "clientType": "confidential",
        "grantType": "client_credentials",
        "allowedRealmRoles": [
          "query-users",
          "view-users",
          "manage-users",
          "view-realm"
        ],
        "deniedCapabilities": [
          "realm-admin",
          "manage-realm",
          "manage-clients",
          "impersonation",
          "cross-realm-access"
        ],
        "interactiveLogin": false,
        "applicationMembership": false,
        "credentialStorage": "environment-specific-secret-management"
      }
    },
    {
      "id": "PROVIDER_OBSERVATION_FRESHNESS_DEADLINE",
      "owner": "Identity + Security; Operations / SRE",
      "options": ["30-second maximum age and 60-second deadline", "60-second maximum age and 120-second deadline", "300-second maximum age and 600-second deadline"],
      "recommended": "30-second heartbeat, 60-second maximum age, 120-second fail-closed deadline",
      "rationale": "bounds stale authority while tolerating one missed observation",
      "affectedTasks": [2, 3, 4, 9],
      "blockerIfUnresolved": "session freshness and outage behavior",
      "status": "approved — owner decision recorded",
      "approved": true,
      "frozen": true,
      "approvedAt": "2026-07-28",
      "approvalReference": "OWNER-DIRECTIVE-2026-07-28-P5T1-10",
      "effectiveContractVersion": "1.0.0",
      "implementationValue": {
        "heartbeatSeconds": 30,
        "maximumAgeSeconds": 60,
        "reconciliationDeadlineSeconds": 120,
        "ageIsUserInactivity": false,
        "staleNewLogin": "deny",
        "staleAuthorityMutation": "deny",
        "staleExistingSessions": "revocation-pending-then-revoked-by-deadline",
        "driftOrDisablement": "immediate-fail-closed",
        "recovery": "fresh-exact-observation-and-new-login"
      }
    },
    {
      "id": "PROFILE_VOLUMES_RESOURCE_LIMITS",
      "owner": "Product / Domain + QA; Platform / DBA + Security",
      "options": ["current four exact proposals", "reduced laptop tier", "staged scale tiers"],
      "recommended": "accept the exact proposed counts only after measured feasibility while keeping all four catalogs complete",
      "rationale": "prevents silent workload reduction and protects the host",
      "affectedTasks": [6, 7, 8, 9],
      "blockerIfUnresolved": "loader implementation and scale qualification",
      "status": "approved — owner decision recorded",
      "approved": true,
      "frozen": true,
      "approvedAt": "2026-07-28",
      "approvalReference": "OWNER-DIRECTIVE-2026-07-28-P5T1-11",
      "effectiveContractVersion": "1.0.0",
      "implementationValue": {
        "profileVersions": {
          "smoke": "1.0.0",
          "acceptance": "1.0.0",
          "realistic": "1.0.0",
          "stress": "1.0.0"
        },
        "exactCounts": "machine-profile-manifests",
        "resourceEnvelopes": {
          "smoke": {
            "cpuCores": 2,
            "memoryMiB": 1024,
            "diskMiB": 2048,
            "objectBytes": 134217728,
            "durationSeconds": 120,
            "cleanupSeconds": 60
          },
          "acceptance": {
            "cpuCores": 4,
            "memoryMiB": 4096,
            "diskMiB": 20480,
            "objectBytes": 2147483648,
            "durationSeconds": 1200,
            "cleanupSeconds": 600
          },
          "realistic": {
            "cpuCores": 8,
            "memoryMiB": 12288,
            "diskMiB": 51200,
            "objectBytes": 21474836480,
            "durationSeconds": 7200,
            "cleanupSeconds": 2700
          },
          "stress": {
            "cpuCores": 12,
            "memoryMiB": 12288,
            "diskMiB": 65536,
            "objectBytes": 8589934592,
            "durationSeconds": 28800,
            "cleanupSeconds": 5400
          }
        },
        "runtimeFeasibility": "not run",
        "silentReduction": "forbidden"
      }
    }
  ]
}
```
<!-- PREPROD_IDENTITY_DATA_CONTRACT:END -->
