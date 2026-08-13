# Local Candidate Alert Catalog

Alerts represent symptoms with bounded durations and recovery conditions. They
do not make legal, enforcement, certificate, Finding-closure, or business KPI
decisions.

## Alert catalog

| ID | Symptom | Expression | Duration | Severity | Owner | Runbook | Deduplication key | Fixture | Recovery |
|---|---|---|---|---|---|---|---|---|---|
| api-read-latency | API read p95 exceeds the candidate objective | histogram_quantile(0.95,aviasurveil_http_server_duration_bucket) > 500 | 5m | warning | Backend | docs/operations/runbooks/INCIDENT_RESPONSE.md | api-read-latency | api-read-latency | p95 remains at or below 500 ms for 5 minutes |
| api-command-latency | API command p95 exceeds the candidate objective | histogram_quantile(0.95,aviasurveil_http_server_duration_bucket) > 1000 | 5m | warning | Backend | docs/operations/runbooks/INCIDENT_RESPONSE.md | api-command-latency | api-command-latency | p95 remains at or below 1000 ms for 5 minutes |
| outbox-ready-warning | Ready outbox work is delayed | aviasurveil_outbox_ready_age_seconds > 120 | 2m | warning | Backend | docs/operations/runbooks/EMAIL_DOCUMENT_WORKERS.md | outbox-ready | outbox-ready-warning | oldest ready age stays at or below 120 seconds for 2 minutes |
| outbox-ready-critical | Ready outbox work is critically delayed | aviasurveil_outbox_ready_age_seconds > 600 | 2m | critical | Backend | docs/operations/runbooks/EMAIL_DOCUMENT_WORKERS.md | outbox-ready | outbox-ready-critical | oldest ready age stays below 600 seconds for 2 minutes |
| worker-attempts | A scan, email, or document job exhausted bounded attempts | increase(aviasurveil_worker_job_attempts_total[15m]) >= 3 | 1m | critical | Backend | docs/operations/runbooks/EMAIL_DOCUMENT_WORKERS.md | worker-attempts | worker-attempts | the affected bounded job completes or is explicitly dead-lettered |
| backup-incremental-age | Incremental recovery metadata is stale | aviasurveil_backup_recovery_point_age_seconds > 1800 | 5m | warning | Platform/Operations | docs/operations/runbooks/BACKUP.md | backup-age | backup-incremental-age | a verified recovery point is newer than 30 minutes |
| backup-full-age | Full or differential recovery data is critically stale | aviasurveil_backup_full_age_seconds > 93600 | 5m | critical | Platform/Operations | docs/operations/runbooks/BACKUP.md | backup-age | backup-full-age | a verified full or differential point is newer than 26 hours |
| required-dependency-down | A required local dependency is unavailable | min(aviasurveil_dependency_health) < 1 | 1m | critical | Platform/Operations | docs/operations/runbooks/INCIDENT_RESPONSE.md | full-stack-dependency | required-dependency-down | all required dependencies report ready for 2 minutes |

Alertmanager groups by alert name, service, and severity; inhibits derivative
latency/job alerts during the declared full-stack dependency outage; and sends
one local firing and one resolved notification to the local Mailpit receiver.
Production recipients and staffed on-call ownership are `not run`; cloud
targets must use their separately configured SMTP provider.
