#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
runtime_directory=$(mktemp -d /private/tmp/avia-auth-mailpit.XXXXXX)
container_name="avia-auth-mailpit-$$"

cleanup() {
  docker rm --force "$container_name" >/dev/null 2>&1 || true
  rm -rf "$runtime_directory"
}

trap cleanup EXIT INT TERM

umask 077
openssl req -x509 -newkey rsa:2048 -nodes -days 1 \
  -keyout "$runtime_directory/smtp.key" \
  -out "$runtime_directory/smtp.crt" \
  -subj "/CN=127.0.0.1" \
  -addext "subjectAltName=IP:127.0.0.1" >/dev/null 2>&1
openssl rand -hex 32 >"$runtime_directory/password"
printf 'aviasurveil360:%s\n' "$(tr -d '\r\n' <"$runtime_directory/password")" >"$runtime_directory/auth"

docker run --detach --rm --name "$container_name" \
  --read-only --tmpfs /data:rw,noexec,nosuid,size=32m --tmpfs /tmp:rw,noexec,nosuid,size=8m \
  --mount "type=bind,src=$runtime_directory,dst=/run/auth,readonly" \
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

cd "$repository_root/apps/auth"
AVIA_AUTH_MAILPIT_ADDRESS="127.0.0.1:$smtp_port" \
AVIA_AUTH_MAILPIT_HTTP_URL="http://127.0.0.1:$http_port" \
AVIA_AUTH_MAILPIT_PASSWORD_FILE="$runtime_directory/password" \
AVIA_AUTH_MAILPIT_CA_FILE="$runtime_directory/smtp.crt" \
GOTOOLCHAIN=local GOCACHE=/private/tmp/avia-auth-go-cache \
go test ./internal/mail -run '^TestMailpitSTARTTLSDelivery$' -count=1 -v
