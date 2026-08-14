# AviaSurveil360 Environment Contract

The root static demo is a local behavior oracle. React/Vite and Go/PostgreSQL
are separate local candidates under `apps/web/` and `apps/api/`; Workspace owns
their Compose composition and `shared/auth` OIDC integration. Use project-local
state paths and ports from the selected command; do not adopt a shared runtime
or browser profile by inference.

Run browser work with an isolated profile when practical and clean task-owned
browser, test-runner, Vite, and service processes afterward. For a runtime
issue, use the Workspace secret-safe `make diagnose TARGET=namibia/dev
SERVICE=surveil` route before reading logs manually. Auth endpoint
qualification needs its own caller-provided database and Mailpit fixtures and
is not a Surveil harness gate.

No Data fixture or harness contract exists here: Data deliberately retired its
harness under `DEBT-001`. Do not create `shared/data/docs/agent-harness`, a
Data certification tree, HMAC snapshots, or an inferred Data fixture.
