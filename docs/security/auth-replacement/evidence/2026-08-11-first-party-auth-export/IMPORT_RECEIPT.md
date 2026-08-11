# First-Party Authentication Export Receipt

## Status

- Evidence status: `verified locally`
- Source status: `candidate-only`
- Runtime integration: `not run`
- Keycloak replacement: `not established`

## Ownership Decision

On 2026-08-11 the repository owner stated that the exported application source
is entirely their own code and that they have authority to copy, modify, and
integrate it into AviaSurveil360. Third-party dependency licenses remain subject
to their own notices and obligations.

## Integrity

- Original temporary path: `/private/tmp/aviasurveil360-auth-export.zip`
- Retained archive: `aviasurveil360-auth-export.zip`
- SHA-256: `7fa982300440cb3e79d28bc0f7f22ebb59124bc9c125dededb22dea306fc7fb7`
- Source repository revision recorded by the export:
  `60dbe494318106569f6d9dbea121d6b1c841ae95`
- Payload files recorded by the export manifest: `30`

The retained archive hash matches the owner-supplied digest. The archive was
extracted under `source/auth-export/`; the retained extraction matched the
independently inspected temporary extraction with `diff -qr` and produced no
differences. No symlink, absolute path, or traversal entry was observed.

## Use Boundary

The retained files are immutable design evidence and source input. They are not
part of an application build and must not be wired into Compose or production
until the Auth Replacement ExecPlan closes its security, protocol, database,
browser, migration, and recovery gates. The export's evaluation migration must
never be applied to the existing AviaSurveil360 database.

## Fresh Retained-Copy Verification

The following checks were run after the retained copy was created:

| Check | Result |
|---|---|
| Retained archive `shasum -a 256` | `verified locally`; exact expected digest |
| Retained archive `unzip -t` | `verified locally`; no compressed-data error |
| Retained extraction versus inspected temporary extraction | `verified locally`; `diff -qr` produced no difference |
| High-signal private-key and AWS access-key pattern scan | `verified locally`; no match |
| Export `go test ./...` using an offline task-owned modfile and local dependency sources | `verified locally`; auth, examples compile, and export tests passed |
| Export `go test -race ./auth ./tests` using the same offline resolution | `verified locally`; both packages passed |
| Export `go vet ./...` using the same offline resolution | `verified locally`; no diagnostic |
| PostgreSQL integration and migration execution | `not run`; no task-owned database was started for source intake |
| Current online dependency/vulnerability scan | `not run` |
| OIDC conformance, application integration, and ARM64 capacity | `not run` |
