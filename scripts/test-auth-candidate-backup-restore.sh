#!/bin/sh
set -eu

# The shared runner creates, destroys, and restores only a fresh task-owned
# candidate PostgreSQL volume. It never addresses a normal profile, Keycloak,
# or an external database.
AVIA_AUTH_CANDIDATE_BACKUP_RESTORE=1 exec "$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)/test-auth-candidate-runtime.sh"
