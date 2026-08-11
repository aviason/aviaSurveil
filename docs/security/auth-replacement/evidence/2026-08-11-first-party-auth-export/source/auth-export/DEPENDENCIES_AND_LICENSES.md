# Dependencies and licenses

## Export module

| Component | Version | Use | License evidence | Security note |
|---|---:|---|---|---|
| Go standard library | Go 1.26.4 used for verification | HTTP, crypto/RSA, SQL interfaces, JSON, time | Go project BSD-style license | Runtime/toolchain supplied by target. |
| `golang.org/x/crypto` | `v0.51.0` | `argon2` only in copied source | BSD-3-Clause (`LICENSE` in local module cache) | Offline scanner reports advisories in other package paths; upgrade to a reviewed release and retest. |
| `golang.org/x/sys` | `v0.44.0` indirect | Transitive support for `x/crypto` on supported platforms | BSD-3-Clause (`LICENSE` in local module cache) | Indirect. |

The export deliberately has no chi dependency: `examples/chi_mount.go` targets
the small `MethodFunc` interface already implemented by chi. It also has no pgx
compile dependency because the copied store uses `database/sql`; an
AviaSurveil360 adapter may register pgx's stdlib driver or replace the store with
native pgx repositories.

## Relevant source dependency

The source repository uses `github.com/jackc/pgx/v5 v5.7.2` under the MIT
License. An offline scan reported critical advisories fixed in later 5.9.x
releases. The vulnerable source version is not added to this export's `go.mod`,
but integration with PostgreSQL must select a patched pgx release and rerun
database/concurrency tests.

## Source repository license blocker

No root license file was found for the source repository during this review.
Therefore this technical export does **not** establish permission to copy,
redistribute, modify, or integrate the EMSI source into AviaSurveil360. Obtain a
clear license/ownership decision before distribution or incorporation. The ZIP
is a candidate evaluation input only.

## Generated and candidate-only files

Documentation, the compact migration projection, `auth/ports.go`, examples,
tests, fixtures, OpenAPI description, and manifest were created for this task.
They do not silently relicense the copied EMSI files or their dependencies.

## Dependency procedure before integration

1. Resolve source-code licensing and third-party notices.
2. Upgrade `x/crypto`, `x/sys`, and the Avia-selected pgx driver to reviewed
   compatible releases.
3. Run `go mod tidy`, `go mod verify`, `go vet`, unit/race/integration tests, and
   an online-current vulnerability scan in the AviaSurveil360 repository.
4. Validate that scanner findings are reachable before accepting or closing
   them; do not treat the offline inventory as a call-graph proof.
5. Preserve dependency license texts in the final distributed application as
   required by their licenses.
