# Local Secret Files

Run `./scripts/init-local-secrets.sh` from the repository root before starting
a local Compose profile. The command creates random credentials beneath
`.local/aviasurveil360/secrets/` with directory mode `0700` and file mode
`0600`. That directory is ignored by Git.

The initializer refuses to replace any existing credential. Use
`./scripts/init-local-secrets.sh --rotate` only when the current task explicitly
requires every local credential to be replaced. Rotation invalidates existing
local database, identity, object-store, and session state; use fresh scoped
Compose volumes with the rotated credentials.

Compose mounts these files as Docker secrets. Do not copy their values into
Compose environment variables, application YAML, shell history, logs, browser
bundles, or image layers.

`deploy/local/config/application.example.yaml` contains only non-secret local
configuration and file-backed secret paths. No encrypted scanner or external
renderer endpoint is retained in the candidate topology.
