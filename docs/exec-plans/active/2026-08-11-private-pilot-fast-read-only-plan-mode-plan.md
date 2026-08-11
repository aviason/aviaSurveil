# AWS Private-Pilot Fast Read-Only Plan Mode

## Objective

Allow an explicitly selected provider-backed Terragrunt `plan` while the
private-pilot owner overlay is being assembled, without requiring the full
decision JSON for that diagnostic step. Keep all mutation commands guarded and
make the fast path visibly read-only.

## Scope

- Add a Terragrunt environment switch that removes the decision and
  remote-authority hooks from `plan` only.
- Keep `apply` and `destroy` on the existing hooks and authorization contract.
- Document the switch and add a focused source-contract assertion.
- Preserve the existing `avia` profile, `eu-central-1` region, backend controls,
  secret handling, and remote-mutation boundary.

## Explicit Exclusions

- No Terraform/Terragrunt apply, destroy, import, bootstrap, or deployment.
- No ECR login, image publication, Cloudflare mutation, DNS cutover, tunnel
  change, secret read/write, SMTP send, or production data change.
- No defaults or guessed values for missing Terraform inputs.
- No branch, commit, push, or unrelated worktree cleanup.

## Affected Interfaces

- `infra/terragrunt/environments/aws-private-pilot/root.hcl`
- `docs/operations/AWS_PRIVATE_PILOT_DECISIONS.md`
- `tests/aws-private-pilot-infrastructure-contract.test.mjs`

## Execution

1. Define `AVIA_AWS_PRIVATE_PILOT_FAST_READ_ONLY_PLAN=true` in the operator
   environment.
2. Run a provider-backed `terragrunt plan` with `-lock=false`, backend init
   disabled where appropriate, and the explicit `avia` profile. This remains
   read-only and may still fail on missing input/provider dependencies.
3. Run the focused contract test and inspect the diff. Do not run a mutation
   command through the fast path.

## Acceptance Criteria

- With the switch unset, `plan`, `apply`, and `destroy` retain both hooks.
- With the switch set, only `plan` omits those hooks; `apply` and `destroy`
  remain guarded.
- Documentation states that the switch grants no mutation authority.
- Focused contract verification passes, and no remote mutation occurs.

## Risks, Dependencies, and Recovery

The fast path can produce a partial or failed plan when owner inputs, provider
credentials, image digests, or external provider metadata are absent. It must
not be used as evidence of deployment readiness. Recovery is to unset the
environment variable and return to the complete decision contract; reverting
the three scoped file edits restores the prior behavior.

## Progress And Outcome

- 2026-08-11: User explicitly authorized this repository policy adjustment.
- 2026-08-11: Terragrunt fast read-only plan switch, documentation, and focused
  contract assertion implemented locally.
- 2026-08-11: Focused infrastructure contract verification passed 23/23. Strict
  render retained `plan`, `apply`, and `destroy` hooks; fast render retained
  only `apply` and `destroy`.
- 2026-08-11: The fast provider plan initialized the AWS and Cloudflare
  providers and reached Terraform validation, then stopped on the intentionally
  blank CIDR, Cloudflare, storage, release, SMTP, bucket, budget, and tag
  inputs. No apply, destroy, lock, or remote resource mutation ran.

## Execution Prompt

Review the scoped diff, run the focused private-pilot infrastructure contract
test, and if it passes run only the explicitly selected read-only Terragrunt
plan. Keep `apply`, `destroy`, and all provider mutations blocked unless a
separate exact authorization is provided.
