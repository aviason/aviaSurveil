# AWS Trial Decision Contract

**Status:** `candidate-only`; AWS execution is `not run`.

No account, region, domain, budget, capacity, retention, or owner value is a
repository default. A release authority must create an untracked
`decision.json` for one planning window. The file belongs under the gitignored
`.local/aviasurveil360/aws-plans/` tree with directory mode `0700` and file mode
`0600`.

## Required Decision

The JSON document uses `schemaVersion: 1` and contains:

| Field | Required decision |
|---|---|
| `approvalId` | Unique review record; it is not an apply authorization |
| `accountId` | Exact 12-digit trial account |
| `region` | Approved region and data-residency location |
| `dataResidencyApproved` | Explicit `true` from the records/data owner |
| `domain` / `certificateArn` | Trial hostname and matching ACM certificate |
| `budgetCeilingUsd` | Positive ceiling, never an inferred default |
| `capacity.min/desired/max` | Reviewed bounded capacity, maximum 20 |
| `backupRetentionDays` | Reviewed 1-35 day trial retention |
| `ownerContacts` | Platform, Security, Records, and Release contacts |
| `changeWindow` | ISO-8601 start/end for the exact planning window |
| `destroyDecision` | `destroy-after-trial` or separately approved retention |
| `approvedPhases` | Only phases currently approved for planning |

The four ordered phases are `bootstrap`, `foundation-ecr`,
`artifact-publication`, and `data-runtime`. Planning approval for one phase does
not approve apply, publication, smoke, rollback, destroy, or any later phase.

## Protected Bundle

Each phase writes a separate bundle containing the decision, binary plans, JSON
plans, and `manifest.json`. The manifest binds:

- caller ARN, account, region, phase, creation, expiry, and cleanup time;
- decision, plan, provider-lock, Terraform, Terragrunt, and wrapper hashes;
- digest-only image references, CycloneDX SBOM hashes, and zero
  HIGH/CRITICAL Trivy results;
- OPA result, cost estimate, reviewed capacity, and exact unit commands.

Active plaintext bundles are ephemeral and mode `0600`. Retained evidence must
be age-encrypted for named recipients before plaintext removal. State, plans,
caller output, command manifests, and decrypted evidence are never committed.
Expired bundles cannot be applied.

## Authorization

`preview` is non-executing. Every mutable wrapper prints an exact authorization
value derived from action, phase, account, region, and the ordered plan hashes:

```text
exact-authorization:<action>:<phase>:<account>:<region>:<aggregate-plan-sha256>
```

The user must approve that exact value in the current task. A changed plan,
wrapper, image, SBOM, caller, decision, or expired window invalidates it.

## Open Production Decisions

The trial does not approve a production AWS account, region, RPO/RTO, on-call
roster, alert recipients, email provider, domain, certificate authority,
records retention, legal hold, backup failure domain, budget, capacity, or
destroy/retention policy. Those remain owner decisions.
