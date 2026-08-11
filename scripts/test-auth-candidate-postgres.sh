#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
compose_file="$repository_root/deploy/local/compose.test.yaml"
runtime_directory=$(mktemp -d /private/tmp/avia-auth-postgres.XXXXXX)
project_name="avia-auth-postgres-$$"
postgres_port=${AVIA_AUTH_TEST_POSTGRES_PORT:-55478}

cleanup() {
  docker compose --project-name "$project_name" --file "$compose_file" down --volumes --remove-orphans >/dev/null 2>&1 || true
  rm -rf "$runtime_directory"
}

trap cleanup EXIT INT TERM

. "$repository_root/scripts/lib/init-local-test-runtime.sh"
initialize_local_test_runtime "$runtime_directory" "http://127.0.0.1:4174" "$repository_root"

export AVIA_TEST_RUNTIME_DIR="$runtime_directory"
export AVIA_TEST_POSTGRES_PORT="$postgres_port"

docker compose --project-name "$project_name" --file "$compose_file" up --detach --wait postgres

database_password=$(tr -d '\r\n' <"$runtime_directory/secrets/app_database_password")
export AVIA_AUTH_TEST_DATABASE_URL="postgresql://aviasurveil:${database_password}@127.0.0.1:${postgres_port}/aviasurveil?sslmode=disable"
unset database_password

cd "$repository_root/apps/auth"
GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache GOMODCACHE=/private/tmp/avia-auth-outbox-mod-cache \
go test ./internal/identity ./internal/session ./internal/mail ./internal/mfa ./internal/challenge ./internal/provider ./migrations -run 'Test(PostgreSQL|IdentityMigration)' -count=1 -v
