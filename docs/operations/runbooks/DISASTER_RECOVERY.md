# Local Disaster Recovery Drill

This runbook proves a same-host, logically isolated `candidate-only` recovery
workflow as `verified locally`. It is not production-ready and does not claim a
separate host, region, account, or legal retention boundary.

## Scope And Owner

Owner: Platform/Operations

Escalation owner: Release authority and Security

Scope is the automated two-point local drill, including corruption fallback,
full identity/application/object verification, browser proof, and cleanup.

## Preconditions

- The backup and restore runbooks have been reviewed.
- Nine accepted local image digests have current CycloneDX and Trivy evidence.
- Docker has capacity for one source full/recovery stack and one isolated
  restore stack.
- Candidate targets are RPO no greater than 15 minutes and RTO no greater than
  60 minutes.

## Symptoms

- Recovery readiness needs current evidence.
- A complete point, corruption fallback, RPO/RTO, identity scope, worker
  backlog, browser route, or cleanup assertion fails.
- A claimed recovery point is newer than the available complete component set.

## Safety Boundary

- The source stack remains the evidence producer and is never a restore target.
- Failure scenarios use isolated target loss; active source volumes and buckets
  are not deleted or overwritten.
- The corruption exercise mutates only a temporary catalog copy. Immutable
  backup objects and catalogs are not modified.
- Cleanup uses exact Compose project labels. Broad deletion commands are
  prohibited.

## Procedure

Run the two-point drill harness against one uniquely named source project. Let
it create a controlled change, restore the latest complete point, reject the
corrupt catalog copy before target creation, restore the prior complete point,
and verify exact cleanup after each isolated target.

## Diagnosis

Check the harness and scenario catalog before starting:

```bash
bash -n scripts/test-rpo-rto-drill.sh
jq empty deploy/recovery/drill-scenarios.json
```

## Expected Output

Both drills must report `verified locally`, `candidate-only`, matching
application/identity/object fingerprints, retained organization and role scope,
normal TOTP login, 86 direct route loads, database/object RPO at or below 900
seconds, RTO at or below 3600 seconds, and zero isolated residue.

## Reversible Mitigation

Run only the isolated drill harness:

```bash
./scripts/test-rpo-rto-drill.sh
```

The harness creates two points around a controlled change, restores the latest
point, rejects a corrupt catalog before target creation, falls back to the
prior complete point, and cleans exact project resources.

## Recovery Verification

Verify both drill results, database and object RPO separately, RTO, exact
fingerprints, normal TOTP login and role scope, restored worker delivery, all 86
route loads, corruption fallback, and zero task-owned residue.

## Evidence Capture

Capture the aggregate JSON, UTC timeline, two point IDs, expected and actual
fingerprints, database/object RPO, RTO, scenario results, fallback result,
browser count, and cleanup status. Label the failure domain
`same-host-logically-isolated`.

## Expected Evidence

The aggregate and both point records must be checksum-bound, contain separate
database/object RPO and RTO values, prove identity/application/object parity,
and use the literal `verified locally` and `candidate-only` labels.

## Cleanup

Remove only the exact drill source and restore projects plus their task-owned
state directories and temporary evidence copies. Verify zero matching
containers, volumes, networks, or secret files and never run a Docker prune.

## Escalation

Platform/Operations owns failed restore or RPO/RTO evidence. Identity and
Security own failed MFA, role, or fingerprint evidence. Records/Legal owns
production retention and legal-hold decisions.

## Authorization Required

A production or AWS drill, real dependency destruction, separate failure
domain, remote account or region, budget, production identity, and retention
change require new explicit authorization.
