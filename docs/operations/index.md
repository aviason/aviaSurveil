# Local Candidate Operations

This collection governs the `candidate-only` local reliability surface. It is
not production-ready. Unless a row says otherwise, procedure evidence is
`not run`; completed local exercises are labeled `verified locally`.
The final local matrix, exact hashes, recovery measurements, and explicit AWS
`not run` boundary are recorded in
[Local Reliability, DR, And Infrastructure Evidence](../demo-evidence/LOCAL_RELIABILITY_AND_DR_2026-07-22.md).

## Operating Contracts

| Document | Purpose |
|---|---|
| [Service objectives](SERVICE_OBJECTIVES.md) | Measurable local engineering objectives. |
| [Telemetry contract](TELEMETRY_CONTRACT.md) | Signal names, attributes, and redaction. |
| [Alert catalog](ALERT_CATALOG.md) | Alert symptoms, owners, recovery conditions, and runbook links. |
| [Ownership](OWNERSHIP.md) | Local owners and escalation boundaries. |
| [AWS trial decisions](AWS_TRIAL_DECISIONS.md) | Required owner inputs, protected plan bundles, and exact action authorization. |
| [AWS trial runbook](AWS_TRIAL_RUNBOOK.md) | Phase-scoped plan, apply, smoke, rollback, retention, and destroy procedure. |

## Runbooks

| Runbook | Use |
|---|---|
| [Start and stop](runbooks/START_STOP.md) | Start, inspect, and remove one exact local stack. |
| [Incident response](runbooks/INCIDENT_RESPONSE.md) | Classify and recover warning or critical local incidents. |
| [Identity and MFA](runbooks/IDENTITY_MFA.md) | Diagnose local OIDC, session, TOTP, and role-scope failures. |
| [Evidence scan](runbooks/EVIDENCE_SCAN.md) | Diagnose quarantine and malware-scan processing. |
| [Email and document workers](runbooks/EMAIL_DOCUMENT_WORKERS.md) | Diagnose outbox, email, and document render work. |
| [Backup](runbooks/BACKUP.md) | Create and verify same-host logical recovery points. |
| [Restore](runbooks/RESTORE.md) | Restore one point into an isolated local target. |
| [Disaster recovery](runbooks/DISASTER_RECOVERY.md) | Exercise two complete restore drills and fallback. |
| [Release and rollback](runbooks/RELEASE_ROLLBACK.md) | Gate a local candidate and abandon a failed candidate safely. |
| [Secret rotation](runbooks/SECRET_ROTATION.md) | Replace local credentials through a parallel fresh stack. |

## Runbook Drill Matrix

| Severity | Mode | Scenario | Owner | Recovery | Evidence |
|---|---|---|---|---|---|
| warning | tabletop | API read latency exceeds its five-minute objective | Backend | Confirm bounded diagnosis, reversible API restart, and p95 recovery check | `verified locally` |
| warning | live | Warning fixtures reach Alertmanager, group once, and emit a resolved message | Backend | Warning fixtures and resolved notification are observed through the isolated observability profile | `verified locally` |
| critical | tabletop | A required dependency is unavailable while derivative alerts are inhibited | Platform/Operations | Confirm dependency-first diagnosis, scoped restart, and two-minute ready condition | `verified locally` |
| critical | live | Critical worker, backup-age, and dependency fixtures reach Alertmanager | Platform/Operations | All critical fixtures are present and the isolated profile returns healthy with zero residue | `verified locally` |

The matrix records local exercises only. It does not establish staffed on-call
coverage, production recipients, a production failure domain, or release
authority.

## Drill Findings

The 2026-07-26 dry-run found and corrected three gaps before the successful
rerun:

- Full-stack startup required the repository-supported Node.js runtime, so the
  start/stop and secret-rotation preconditions now say so.
- Runtime secret scanning treated an unavailable `rg` command like zero
  matches. The checker now fails closed with exit 69 before evidence collection,
  and a regression test exercises that behavior.
- Scoped `docker compose up --wait` examples omitted the exact HTTPS port and
  public origin, which caused configuration drift. Every startup/recovery block
  now carries both values, and the contract rejects omissions.

The corrected dry-run proved exact project status, readiness, OIDC discovery,
read-only scan/outbox/document diagnosis, scoped API/worker/Keycloak restart,
image evidence, secret-log scanning, and zero full-stack residue. The live
observability rerun proved all eight alert fixtures, one grouped delivery, a
resolved message, telemetry persistence across restart, and zero observability
residue as `verified locally`.
