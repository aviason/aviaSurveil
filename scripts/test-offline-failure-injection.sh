#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"

npm --prefix "$repo_root/apps/web" test -- \
  --run \
  src/offline/db-lifecycle.test.ts \
  src/offline/profile-authority.test.ts \
  src/offline/support-diagnostics.test.ts \
  src/offline/origin-boundary.test.ts \
  src/offline/resumable-upload.test.ts \
  tests/offline/field-repository.test.ts \
  tests/offline/opfs-inspection-attachment-store.test.ts \
  tests/offline/sync-engine.test.ts
go -C "$repo_root/apps/api" test ./internal/sync ./internal/inspections/attachments ./internal/inspections/finalization ./internal/platform/auditevent ./migrations
git -C "$repo_root" diff --check
git -C "$repo_root/apps/surveil" diff --check
