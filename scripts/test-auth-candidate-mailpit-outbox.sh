#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
compose_file="$repository_root/deploy/local/compose.test.yaml"
runtime_directory=$(mktemp -d /private/tmp/avia-auth-mailpit-outbox.XXXXXX)
mailpit_directory="$runtime_directory/mailpit"
project_name="avia-auth-mailpit-outbox-$$"
container_name="avia-auth-mailpit-outbox-$$"
postgres_port=${AVIA_AUTH_TEST_POSTGRES_PORT:-55479}

cleanup() {
  docker rm --force "$container_name" >/dev/null 2>&1 || true
  docker compose --project-name "$project_name" --file "$compose_file" down --volumes --remove-orphans >/dev/null 2>&1 || true
  rm -rf "$runtime_directory"
}

trap cleanup EXIT INT TERM

. "$repository_root/scripts/lib/init-local-test-runtime.sh"
initialize_local_test_runtime "$runtime_directory" "http://127.0.0.1:4174" "$repository_root"
mkdir -p "$mailpit_directory"
chmod 0700 "$mailpit_directory"

openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
  -keyout "$mailpit_directory/smtp.key" \
  -out "$mailpit_directory/smtp.crt" \
  -subj "/CN=127.0.0.1" \
  -addext "subjectAltName=IP:127.0.0.1" >/dev/null 2>&1
openssl rand -hex 32 >"$mailpit_directory/password"
printf 'aviasurveil360:%s\n' "$(tr -d '\r\n' <"$mailpit_directory/password")" >"$mailpit_directory/auth"
chmod 0400 "$mailpit_directory/smtp.key" "$mailpit_directory/smtp.crt" "$mailpit_directory/password" "$mailpit_directory/auth"

export AVIA_TEST_RUNTIME_DIR="$runtime_directory"
export AVIA_TEST_POSTGRES_PORT="$postgres_port"
docker compose --project-name "$project_name" --file "$compose_file" up --detach --wait postgres

docker run --detach --rm --name "$container_name" \
  --read-only --tmpfs /data:rw,noexec,nosuid,size=32m --tmpfs /tmp:rw,noexec,nosuid,size=8m \
  --mount "type=bind,src=$mailpit_directory,dst=/run/auth,readonly" \
  --publish 127.0.0.1::1025 --publish 127.0.0.1::8025 \
  axllent/mailpit:v1.27.8@sha256:6abc8e633df15eaf785cfcf38bae48e66f64beecdc03121e249d0f9ec15f0707 \
  --database /data/mailpit.db --smtp-auth-file /run/auth/auth --smtp-tls-cert /run/auth/smtp.crt --smtp-tls-key /run/auth/smtp.key --smtp-require-starttls --smtp-disable-rdns >/dev/null

for attempt in 1 2 3 4 5 6 7 8 9 10; do
  if docker exec "$container_name" mailpit readyz >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "$container_name" mailpit readyz >/dev/null

smtp_port=$(docker port "$container_name" 1025/tcp | sed -n 's/.*:\([0-9][0-9]*\)$/\1/p')
http_port=$(docker port "$container_name" 8025/tcp | sed -n 's/.*:\([0-9][0-9]*\)$/\1/p')
test -n "$smtp_port" && test -n "$http_port"
database_password=$(tr -d '\r\n' <"$runtime_directory/secrets/app_database_password")
export AVIA_AUTH_TEST_DATABASE_URL="postgresql://aviasurveil:${database_password}@127.0.0.1:${postgres_port}/aviasurveil?sslmode=disable"
unset database_password

cd "$repository_root/apps/auth"
AVIA_AUTH_MAILPIT_ADDRESS="127.0.0.1:$smtp_port" \
AVIA_AUTH_MAILPIT_HTTP_URL="http://127.0.0.1:$http_port" \
AVIA_AUTH_MAILPIT_PASSWORD_FILE="$mailpit_directory/password" \
AVIA_AUTH_MAILPIT_CA_FILE="$mailpit_directory/smtp.crt" \
GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache GOMODCACHE=/private/tmp/avia-auth-outbox-mod-cache \
go test ./internal/mail -run '^TestMailpitSTARTTLS(Delivery|OutboxRetryDelivery)$' -count=1 -v

printf '%s\n' 'auth-candidate Mailpit outbox: verified locally'
