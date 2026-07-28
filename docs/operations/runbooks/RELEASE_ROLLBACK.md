# Local Candidate Release And Rollback

This runbook gates only a local `candidate-only` artifact. It is not
production-ready, and release or rollback evidence is `not run` until the
listed local gates complete.

## Scope And Owner

Owner: Platform/Operations

Escalation owner: Release authority

Scope is validation of immutable local image evidence, a fresh local candidate
stack, and abandonment of a failed task-owned candidate without changing an
accepted running stack.

## Preconditions

- Identify the candidate commit and all accepted image digests.
- Confirm SBOM and HIGH/CRITICAL vulnerability evidence match those digests.
- Keep the accepted stack and its state unchanged while validating a new unique
  candidate project.
- Record the rollback decision owner before any promotion.

## Symptoms

- Image digest, SBOM, vulnerability, runtime, migration, browser, or cleanup
  evidence fails.
- A new local candidate cannot become healthy or regresses a required
  scenario.
- Release authority withholds approval.

## Safety Boundary

- Local `GO` is not deployment or production readiness.
- Never replace an accepted stack in place while evaluating a candidate.
- Do not weaken scans, masks, thresholds, identity, privacy, or semantic
  assertions to obtain a passing result.

## Diagnosis

```bash
./scripts/check-local-image-evidence.sh full
./scripts/test-local-full-profile.sh
git diff --check
```

## Expected Output

Every image digest matches accepted evidence; the clean full profile proves
required local scenarios and zero task-owned residue. Any failed gate stops the
candidate.

## Reversible Mitigation

Leave the accepted stack unchanged and remove only the failed candidate:

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-release-candidate"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
./scripts/local-stack.sh down full
```

Return to the previously accepted commit and image set through the repository's
normal reviewed change process; do not rewrite shared history or retag an image
without provenance.

## Recovery Verification

```bash
export AVIA_LOCAL_PROJECT="aviasurveil360-task-accepted-example"
export AVIASURVEIL_LOCAL_STATE_DIR="$PWD/.local/aviasurveil360/projects/$AVIA_LOCAL_PROJECT"
./scripts/local-stack.sh check full
```

Confirm the accepted stack was unchanged, the failed candidate has zero
project-labeled residue, and the decision record is complete before labeling
the rollback `verified locally`.

## Evidence Capture

Capture commit, image digests, evidence hashes, gate results, UTC decision
timeline, exact candidate project, rollback owner, accepted-stack health, and
candidate cleanup status.

## Escalation

Escalate failed integrity or vulnerability evidence to Security, runtime
failures to the owning engineering team, and any promotion decision to Release
authority.

## Authorization Required

Commit, push, tag, registry publication, deployment, migration of shared data,
production release or rollback, traffic change, legacy removal, and AWS action
require new explicit authorization.
