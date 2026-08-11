# Canonical AGA External Preprod Release And Handoff ExecPlan

This ExecPlan is a living document. Keep `Progress`, `Decision Log`,
`Discoveries`, and `Outcome` synchronized with actual work. Follow
[`docs/PLANS.md`](../../PLANS.md), the repository
[`AGENTS.md`](../../../AGENTS.md), and the literal evidence vocabulary in the
[`output contract`](../../agent-harness/output-contract.md).

## Status

- Plan status: `paused` — deliberately deferred by the user on 2026-08-10.
- Execution authority: none. No cloud discovery, provisioning, deployment,
  upload, DNS, identity, cost-bearing, or other remote mutation is authorized.
- Current result: `not run`.
- Dependency: local Tasks 8–9 are complete, including user-selected donor
  deletion and post-deletion requalification. The stakeholder-owned
  1440x900, 1024x768, and 390x844 review remains pending.
- Current public demo: `https://demo.aviasurveil.com` is a local-origin named
  Cloudflare Tunnel publication. It is sufficient for the current demo and
  stakeholder review, but it is not external-preprod evidence because the
  application and disposable services still run on the operator's Mac.
- Next: remain paused until the user explicitly resumes this plan after the
  local demo milestone. Resumption does not itself authorize any remote action.

## Objective

Make an accepted, locally qualified canonical AGA candidate available in a
dedicated remote preprod environment that operates independently of the
operator's Mac, without conflating a public local-origin demo with deployment,
preprod verification, production readiness, or provider authorization.

## User-Visible Outcome

If this plan is later authorized and completed, authorized stakeholders can use
the same HTTPS/OIDC canonical workflow on a dedicated remote environment while
the operator's Mac and local Docker profile are offline. The strongest possible
result is `preprod verified`; this plan cannot establish `production-ready`.

## Scope

- select an explicitly approved remote provider and environment identity;
- freeze exact application image/artifact, migration, and synthetic data-profile
  digests;
- provision a dedicated, disposable preprod namespace with HTTPS, OIDC,
  PostgreSQL, object storage, malware scanning, document rendering, and mail
  capture/delivery equivalents;
- load only privacy-safe synthetic exercise data after proving the target is
  not shared or stable;
- run the remote role/privacy/browser and full hero lifecycle matrices;
- prove exact backup, restore, rollback, whole-namespace cleanup, and residue;
  and
- produce an operator handoff with literal evidence labels.

## Explicit Exclusions

- The current `demo.aviasurveil.com` local-origin Tunnel lifecycle remains in
  the local canonical demo plan and is not recreated here.
- No production deployment, production data, real stakeholder provisioning,
  regulatory publication, source-owner approval, or production cutover.
- No provider is selected by this document.
- No Terraform/Terragrunt initialization, cloud discovery, plan, apply, upload,
  DNS, Cloudflare Access, secret creation, or cost-bearing action without its
  own current explicit authorization.
- This plan does not block truthful local demo completion or stakeholder
  acceptance while it is paused.

## Entry Gates

1. Satisfied: the [Canonical AGA Preprod End-To-End Product](2026-08-07-canonical-aga-preprod-end-to-end-product-plan.md)
   records the passed Task 8 fault/restart matrix.
2. Satisfied: the user selected Task 9 `delete`, physical removal completed,
   and the post-deletion matrix passed.
3. The user completes the 1440x900, 1024x768, and 390x844 manual review or
   explicitly records a different stakeholder disposition.
4. The local release candidate artifacts, migrations, rollback inputs,
   backup/restore evidence, OIDC/TLS settings, and runbook are internally
   consistent.
5. The user explicitly resumes this plan and separately authorizes the first
   read-only provider-discovery slice.

## Ordered Work

### Task 1 — Freeze The Candidate Release Input

Record exact source revision, OCI image digests, SBOMs, migration set, exercise
profile digest, runtime configuration contract, backup/restore receipts, and
rollback input. Do not rebuild silently after the freeze.

### Task 2 — Select And Authorize The Remote Environment

Choose the provider and exact account/project/region/environment identity.
Review the applicable
[AWS Preprod Validation](2026-07-27-aws-preprod-validation-plan.md) or create an
equivalent provider plan. Obtain separate authorization for read-only discovery
and for every later cost-bearing or mutating slice.

### Task 3 — Provision A Dedicated Disposable Target

Provision only the approved resources and prove the exercise target is a
dedicated disposable environment, tenant, database, or schema whose complete
namespace can be destroyed without selectively deleting append-only history.
Reject shared or stable targets before any data write.

### Task 4 — Deploy, Load, And Qualify

Deploy the frozen artifacts, apply the frozen migrations, configure HTTPS/OIDC,
load the approved synthetic profile, and run the complete remote nine-role,
privacy, authority, object, report, notification, and canonical hero lifecycle
matrix. Do not use real AGA bodies in visual artifacts or real users/data.

### Task 5 — Prove Recovery, Cleanup, And Handoff

Prove backup/restore, rollback to the frozen predecessor, whole-namespace
cleanup, exact environment identity, and zero unintended residue. Record the
literal outcome and operator runbook. Retention requires a separate explicit
decision; a test environment is not retained by inference.

## Verification And Expected Observations

Provider-specific commands are intentionally absent while the provider and
authorization slices are unselected. When resumed, add exact read-only and
mutating commands only after their matching authorization is recorded.

Required observations are:

- the remote hostname remains healthy while the operator's Mac and local
  Docker stack are offline;
- OIDC issuer, redirect URIs, Secure/HttpOnly cookie behavior, role authority,
  organization privacy, and logout/account switching match the frozen contract;
- the full canonical lifecycle passes against distinct remote role sessions;
- the exact synthetic profile is accepted only by the dedicated disposable
  target;
- signed object upload/download, malware scan, PDF rendering, notifications,
  backup/restore, rollback, and cleanup use the selected remote dependencies;
  and
- every resource and residue query is bound to the exact environment identity.

## Acceptance

- Every remote action has its own recorded authorization and literal result.
- The deployed inputs match the frozen digests.
- The target is proven dedicated/disposable and shared-target loading fails
  closed before any write.
- HTTPS/OIDC/role/privacy and the complete hero lifecycle pass remotely.
- Recovery, rollback, cleanup, and residue checks pass against exact resource
  identities.
- The result is labelled at most `preprod verified`; `production-ready` remains
  unestablished.

## Risks And Mitigations

| Risk | Mitigation |
|---|---|
| A public local Tunnel is mistaken for remote preprod | Require the environment to remain healthy with the operator's Mac/local Docker offline. |
| Local success is inferred as deployment authority | Require action-by-action user authorization and literal `not run` until each slice executes. |
| Exercise data pollutes a shared append-only environment | Reject shared/stable targets before loading and use whole-namespace cleanup only. |
| Artifact drift invalidates rollback | Freeze exact image, migration, data-profile, configuration, and predecessor digests. |
| Cost or remote residue is left behind | Bind every resource to exact environment identity and prove cleanup/residue or explicitly authorize retention. |
| Preprod is described as production-ready | Cap the claim at `preprod verified`; production remains a separate plan and decision. |

## Idempotence And Recovery

- Discovery is read-only and records exact account/project/region identity.
- Provisioning and deployment commands must be environment-bound and safe to
  resume without creating parallel unnamed resources.
- Migration and exercise loading use frozen digests and reject changed replay.
- Recovery restores only into the exact approved environment or an explicitly
  named recovery target.
- Destruction targets only enumerated environment-owned resources; broad or
  unresolved delete commands are forbidden.

## Progress

- [x] 2026-08-10: external preprod work was transferred out of the canonical
  local demo plan at the user's direction.
- [x] 2026-08-10: the user accepted `demo.aviasurveil.com` as sufficient for
  the current local-origin demo milestone.
- [ ] Resume this plan after the local Task 8, explicit Task 9 decision, and
  manual three-viewport stakeholder sequence is complete.
- [ ] Select a provider and authorize read-only discovery.
- [ ] Execute Tasks 1–5.

## Decision Log

### 2026-08-10 — Defer External Preprod Until After The Demo Milestone

The user directed that remote preprod must not remain a task or closeout gate
inside the canonical local demo plan. The local-origin named Tunnel is adequate
for the current demo. This separate plan is therefore `paused`, every remote
action is `not run`, and no action is authorized by creating this document.

## Discoveries

- A stable public hostname does not establish remote hosting: the named Tunnel
  currently forwards to task-owned services on the operator's Mac.
- The existing AWS validation plan remains a possible implementation dependency,
  not an authorization and not a provider choice.

## Outcome

`not run`. The plan is paused by deliberate product sequencing, not blocked by
a technical failure. The current product remains `candidate-only` and `release
pending`.

## Execution Prompt

```text
Resume docs/exec-plans/active/2026-08-10-canonical-aga-external-preprod-release-plan.md only after the canonical local demo plan records its remaining Task 8 fault/restart matrix, the user makes the separate Task 9 donor decision, and the manual 1440x900/1024x768/390x844 stakeholder review is complete or explicitly dispositioned.

Read this plan, docs/PLANS.md, AGENTS.md, ARCHITECTURE.md, the canonical local demo plan, the selected provider plan, and current evidence before any action. Confirm the exact provider/account/project/region/environment and obtain separate explicit authorization for read-only discovery. Do not infer authority for provisioning, deployment, upload, DNS, identity, secrets, cost-bearing resources, retention, rollback, or cleanup from plan resumption or local demo success. Keep every remote action literally `not run` until its own authorization and result exist. Never use production or real stakeholder data. The strongest possible outcome is `preprod verified`; production readiness remains unestablished.
```
