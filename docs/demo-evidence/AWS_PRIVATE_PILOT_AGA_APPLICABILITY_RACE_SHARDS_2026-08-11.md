# AWS Private-Pilot AGA Applicability Local Race-Shard Evidence

**Evidence date:** 11 August 2026  
**Scope:** local-only AGA applicability race requalification  
**Runner:** [`test-aws-private-pilot-agaapplicability-race-shards.sh`](../../scripts/test-aws-private-pilot-agaapplicability-race-shards.sh)  
**Result:** `verified locally`

## Result

The runner discovered the expected 45 top-level `agaapplicability` tests and
executed all of them across 13 sequential race shards. Every shard exited
successfully; no race report, test failure, or shard timeout occurred.

| Shard | Timeout | Result | Elapsed |
|---|---:|---|---:|
| `authority-inputs` | 360s | `verified locally` | 188.546s |
| `classification-boundaries` | 360s | `verified locally` | 327.646s |
| `sealed-classification` | 360s | `verified locally` | 246.183s |
| `draft-core` | 360s | `verified locally` | 140.345s |
| `runtime-public-roundtrip` | 360s | `verified locally` | 198.640s |
| `runtime-hydration` | 420s | `verified locally` | 291.136s |
| `workspace-identity` | 360s | `verified locally` | 126.165s |
| `batch-readiness` | 360s | `verified locally` | 207.108s |
| `recommendation-current-leaf` | 420s | `verified locally` | 400.838s |
| `recommendation-derived-facts` | 420s | `verified locally` | 164.460s |
| `recommendation-kind-profile` | 420s | `verified locally` | 201.924s |
| `recommendation-ambiguous-leaf` | 420s | `verified locally` | 202.202s |
| `recommendation-readiness-snapshot` | 420s | `verified locally` | 182.443s |

The runner uses `-race -p 1 -count=1` and task-specific Go locations:

```text
GOMODCACHE=/private/tmp/avia-task8-modcache
GOPATH=/private/tmp/avia-task8-gopath
GOCACHE=/private/tmp/avia-task8-gocache
GOTMPDIR=/private/tmp/avia-task8-gotmp
```

The old unsharded package-wide race command remains `blocked` by its ten-minute
wall-clock limit. It is intentionally not the local gate anymore; the 13-shard
runner is the durable gate.

## Related local evidence

- `./scripts/test-aws-private-pilot-local-integrations.sh` — PostgreSQL,
  MinIO, ClamAV, Mailpit, migration, object identity, scanner-loss, Mailpit,
  and cleanup checks: `verified locally`.
- Full disposable-copy non-race Go suite with local PostgreSQL and MinIO —
  `go test -p 1 -count=1 ./...`: `verified locally`.
- Main working tree after verification: clean; no Go module files were changed.

AWS, ECR, Cloudflare, provider, deployment, and remote smoke/recovery actions
were `not run`.

Local evidence is `candidate-only`; `production-readiness not claimed`.
