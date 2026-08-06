# Department Manager AGA Multi-Role Demo Runbook

## Boundary

This runbook operates only the disposable `local-preprod` AGA demo namespace.
The result is `candidate-only`, `release pending`, and
`production-ready: not established`. It creates no canonical Audit Plan,
Checklist, Assignment, Finding, CAP, Evidence, Organization, Provider, or
publication record.

The 1,310 source questions remain sealed candidate material. A simulation
disposition is not source approval, technical approval, legal attestation, or
publication.

## Prerequisites

- Run from the repository root on the current branch.
- Use the accepted `AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip` input and
  the existing target-bound local-preprod authorization workflow.
- Keep private roots, authorization bundles, passwords, OIDC values, and
  receipt files outside the repository with mode `0700` for directories and
  `0600` for files. Do not copy them into this runbook or workspace evidence.
- Use an isolated browser profile. Text-bearing runs have screenshot, video,
  and trace retention disabled.

## Start and qualify an isolated session

The wrapper is the single connected entry point. `prepare` intentionally stops
after creating a target-bound intent and returns exit code `2` while awaiting
the private operator authorization; this is an expected pending-authority
state, not a successful qualification.

```bash
bash scripts/test-aga-manager-multi-role-demo-connected.sh prepare
```

After the private target-bound authority is issued, continue the same prepared
target. Do not create a second target or replace the target fingerprint.

```bash
bash scripts/test-aga-manager-multi-role-demo-connected.sh qualify
```

`qualify` provisions the isolated fixture, proves the 1,310-question Manager
package, runs the connected browser scenario, exercises the multi-role
lifecycle, writes privacy-safe receipts, and performs whole-namespace cleanup.
The accepted run reports 14 completed happy-path phases, 17 browser tests, and
zero final residue.

For a manual presentation that must retain the named disposable target until
the operator stops it, use the same prepared target-bound authority and the
HTTP artifact on `127.0.0.1:4174`:

```bash
npm --prefix apps/web run build:http
npm --prefix apps/web exec -- vite preview --outDir dist/http --host 127.0.0.1 --port 4174 --strictPort
```

Start only the exact local-preprod services required by the prepared target;
the connected harness is the authority for the complete Compose profile and
service ordering. Never expose this namespace publicly.

## Presentation sequence

1. Sign in as the exactly bound Department Manager and open
   `/department-manager/aga-demo-workspace/inspection-package`.
2. Show `1,310 candidate AGA questions`, page through all 53 pages, and show
   that one active text page contains at most 25 sealed candidate bodies.
3. Use the deterministic presentation subset. Show the server-issued batch
   count, preview digest, `500` item cap, eligible subset, and simulation-only
   language. Confirm the batch before executing it.
4. Review the package summary, bindings, provider/target/profile scope, source
   gaps, and the `ready for demo simulation` state. Mark readiness only through
   the visible server-backed control.
5. Create the current synthetic recommendation and release the current
   synthetic inspection. Reload once to demonstrate that the same current
   objects are returned.
6. Sign in as the assigned Inspector at
   `/inspector/aga-demo-workspace/inspection`, execute the visible checklist,
   and create the synthetic Potential Finding.
7. Sign in as the assigned Lead Inspector at
   `/lead-inspector/aga-demo-workspace/potential-findings`, review the Finding,
   and use the visible CAP transition. CAP acceptance must remain visibly
   separate from Finding closure.
8. Sign in as the Auditee at
   `/auditee/aga-demo-workspace/caps-evidence`, submit the CAP and Evidence
   revisions. The Auditee projection must not show Internal CAA Notes or CAA
   workload.
9. Use the separately bound CAA Reviewer presentation route
   `/caa-reviewer/aga-demo-workspace/caps-evidence` to review the CAP and
   Evidence. The connected fixture reuses the Lead Inspector browser session
   for this separately bound CAA Reviewer membership; this does not merge the
   server-side role bindings.
10. Return to the assigned Inspector or Lead Inspector route, verify accepted
    Evidence, and use `VERIFY_EVIDENCE` followed by `CLOSE`. The final Finding
    state must be `CLOSED` with closure basis `EVIDENCE_VERIFIED`.

## Recovery

If a process stops after an authority or receipt boundary, inspect the private
phase journal and use the matching recovery mode against the same target:

```bash
bash scripts/test-aga-manager-multi-role-demo-connected.sh recover-status
bash scripts/test-aga-manager-multi-role-demo-connected.sh recover-qualify
```

Receipt replay is the recovery mechanism. Do not reconstruct IDs in the
browser, reset a nonterminal lifecycle, edit a receipt, or delete a shared
database directory. A stale batch preview, stale setup digest, stale binding,
or changed idempotency input must fail closed and be regenerated through the
server-backed flow.

The fault-matrix recovery authority is exercised separately with the matching
`fault-matrix-*` and `recover-*` modes. Its four cases cover inherited-base,
workspace-transaction, concurrent-token, and cleanup receipt gaps; each must
finish `CLEANED` with zero duplicate effects and zero residue.

## Stop and prove cleanup

Use the exact private target-bound cleanup authorization, then inspect the
task-owned resources. The cleanup target must be the named disposable root,
never a broad workspace or home-directory path.

```bash
bash scripts/test-aga-manager-multi-role-demo-connected.sh cleanup-prepared
docker compose --project-name aviasurveil360-local-preprod \
  --file deploy/local/compose.yaml \
  --profile local-preprod-loader \
  --profile aga-candidate-demo \
  --profile aga-demo-workspace-loader \
  --profile aga-candidate-demo-oidc-fixture \
  --profile preproddemo down --volumes --remove-orphans
pgrep -fl 'playwright|vite|preprod-aga-demo-api|test-aga-manager|test-aga-hybrid' || true
```

The final operator check must report no task-owned browser, Playwright, Vite,
API, or Compose process and no disposable namespace residue. Preserve unrelated
user files, including the independent untracked AGA source-risk deliverable.

## Expected handoff

```text
interactive local-preprod multi-role AGA demo; verified locally
candidate-only
release pending
production-ready: not established
```
