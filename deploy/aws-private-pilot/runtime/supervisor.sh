#!/bin/sh
set -eu

project=aviasurveil360-private-pilot
compose_file=/opt/aviasurveil360/private-pilot/deploy/aws-private-pilot/compose.yaml
manifest=/etc/aviasurveil360/private-pilot/release-manifest.json
release_command=/opt/aviasurveil360/private-pilot/scripts/aws-private-pilot-release.sh
tunnel_container=aviasurveil360-private-pilot-tunnel
tunnel_token_file=/run/aviasurveil360-private-pilot/secrets/cloudflare_tunnel_token
tunnel_metrics_url=http://127.0.0.1:2000/metrics

compose() {
  docker compose --project-name "$project" --file "$compose_file" "$@"
}

required_services='gateway api worker keycloak'

require_root_private_file() {
  private_file=$1
  if [ ! -f "$private_file" ] || [ -L "$private_file" ]; then
    echo "required root-owned private-pilot file is unavailable or is a symlink" >&2
    exit 1
  fi
  if [ "$(stat -c '%a' "$private_file")" != 600 ]; then
    echo "required root-owned private-pilot file is not mode 0600" >&2
    exit 1
  fi
  if [ "$(stat -c '%u' "$private_file")" != 0 ]; then
    echo "required private-pilot file is not owned by root" >&2
    exit 1
  fi
}

require_tunnel_token_contents() {
  token_file=$1
  token_bytes=$(wc -c <"$token_file" | tr -d ' ')
  if [ "$token_bytes" -lt 40 ] || [ "$token_bytes" -gt 4096 ]; then
    echo "Cloudflare Tunnel connector token has an invalid length" >&2
    exit 1
  fi
  if grep -qx 'PENDING_SEPARATE_AUTHORIZATION' "$token_file"; then
    echo "Cloudflare Tunnel connector token remains an unauthorized placeholder" >&2
    exit 1
  fi
  if ! LC_ALL=C grep -Eq '^[A-Za-z0-9._=-]+$' "$token_file"; then
    echo "Cloudflare Tunnel connector token has an invalid shape" >&2
    exit 1
  fi
}

require_connector_private_file() {
  private_file=$1
  if [ ! -f "$private_file" ] || [ -L "$private_file" ]; then
    echo "required Cloudflare Tunnel connector token file is unavailable or is a symlink" >&2
    exit 1
  fi
  if [ "$(stat -c '%a' "$private_file")" != 400 ]; then
    echo "Cloudflare Tunnel connector token file is not mode 0400" >&2
    exit 1
  fi
  if [ "$(stat -c '%u:%g' "$private_file")" != 65532:65532 ]; then
    echo "Cloudflare Tunnel connector token file is not owned by the connector uid/gid" >&2
    exit 1
  fi
  require_tunnel_token_contents "$private_file"
}

validate_tunnel_image() {
  image=${AVIA_CLOUDFLARED_IMAGE:?required digest-bound cloudflared image}
  account=${AVIA_AWS_ACCOUNT_ID:?required AWS account id}
  region=${AVIA_AWS_REGION:?required AWS region}
  case "$account" in *[!0-9]*|'') echo "invalid AWS account id for cloudflared image" >&2; exit 1;; esac
  case "$region" in *[!a-z0-9-]*|'') echo "invalid AWS region for cloudflared image" >&2; exit 1;; esac
  expected_prefix="${account}.dkr-ecr.${region}.on.aws/"
  case "$image" in
    "$expected_prefix"*@sha256:*) ;;
    *) echo "cloudflared image is not an immutable target-account IPv6 ECR subject" >&2; exit 1 ;;
  esac
  digest=${image##*@sha256:}
  if [ "${#digest}" -ne 64 ] || printf '%s' "$digest" | grep -Eq '[^0-9a-f]'; then
    echo "cloudflared image digest is invalid" >&2
    exit 1
  fi
}

validate_tunnel_runtime() {
  [ "${AVIA_CLOUDFLARE_TUNNEL_TOKEN_FILE:?required connector token file}" = "$tunnel_token_file" ] || {
    echo "Cloudflare Tunnel connector token path escaped the owned runtime directory" >&2
    exit 1
  }
  [ "${AVIA_CLOUDFLARE_EDGE_IP_VERSION:?required Cloudflare edge IP version}" = 6 ] || {
    echo "Cloudflare Tunnel must be pinned to IPv6 edge connectivity" >&2
    exit 1
  }
  validate_tunnel_image
  require_connector_private_file "$tunnel_token_file"
}

materialize_tunnel_token() {
  parameter_name=${AVIA_CLOUDFLARE_TUNNEL_TOKEN_PARAMETER_NAME:?required SSM connector-token parameter name}
  case "$parameter_name" in
    /aviasurveil360/private-pilot/*) ;;
    *) echo "Cloudflare Tunnel connector-token parameter escaped the private-pilot namespace" >&2; exit 1 ;;
  esac

  secret_directory=$(dirname "$tunnel_token_file")
  install -d -m 0700 -o root -g root "$secret_directory"
  raw_file=$(mktemp "$secret_directory/.cloudflare-tunnel-token.raw.XXXXXX")
  token_file=$(mktemp "$secret_directory/.cloudflare-tunnel-token.XXXXXX")
  cleanup_token_files() {
    rm -f "$raw_file" "$token_file"
  }
  trap cleanup_token_files EXIT HUP INT TERM

  aws --region "${AVIA_AWS_REGION:?required AWS region}" ssm get-parameter \
    --name "$parameter_name" \
    --with-decryption \
    --query Parameter.Value \
    --output text >"$raw_file"
  tr -d '\r\n' <"$raw_file" >"$token_file"
  raw_bytes=$(wc -c <"$raw_file" | tr -d ' ')
  token_bytes=$(wc -c <"$token_file" | tr -d ' ')
  if [ $((raw_bytes - token_bytes)) -gt 2 ]; then
    echo "Cloudflare Tunnel connector token contains unexpected line breaks" >&2
    exit 1
  fi
  require_tunnel_token_contents "$token_file"
  chmod 0400 "$token_file"
  chown 65532:65532 "$token_file"
  mv -f "$token_file" "$tunnel_token_file"
  rm -f "$raw_file"
  trap - EXIT HUP INT TERM
}

publish_tunnel_connections() {
  connections=$1
  instance_id=${AVIA_INSTANCE_ID:?required EC2 instance id}
  if ! printf '%s' "$instance_id" | grep -Eq '^i-[0-9a-f]{17}$'; then
    echo "invalid EC2 instance id for tunnel health metric" >&2
    exit 1
  fi
  aws --region "${AVIA_AWS_REGION:?required AWS region}" cloudwatch put-metric-data \
    --namespace AviaSurveil360/PrivatePilot \
    --metric-data "MetricName=CloudflaredTunnelHAConnections,Dimensions=[{Name=InstanceId,Value=${instance_id}}],Value=${connections},Unit=Count"
}

case "${1:-}" in
  validate-runtime)
    test -x "$release_command"
    require_root_private_file "$manifest"
    require_root_private_file /etc/aviasurveil360/private-pilot/runtime.env
    require_root_private_file "${DOCKER_CONFIG:?required private-pilot Docker configuration}/config.json"
    for variable in \
      AVIA_APP_DATABASE_PASSWORD_FILE AVIA_KEYCLOAK_DATABASE_PASSWORD_FILE \
      AVIA_OIDC_CLIENT_SECRET_FILE AVIA_SESSION_ENCRYPTION_KEY_FILE \
      AVIA_KEYCLOAK_SERVICE_CLIENT_SECRET_FILE AVIA_APP_SMTP_PASSWORD_FILE \
      AVIA_KEYCLOAK_SMTP_PASSWORD_FILE
    do
      secret_path=$(printenv "$variable")
      case "$secret_path" in
        /run/aviasurveil360-private-pilot/secrets/*) ;;
        *) echo "private-pilot secret path escaped the owned runtime directory" >&2; exit 1 ;;
      esac
      require_root_private_file "$secret_path"
    done
    "$release_command" validate "$manifest" >/dev/null
    compose config --quiet
    actual=$(compose config --services | tr '\n' ' ')
    for service in $required_services; do
      case " $actual " in
        *" $service "*) ;;
        *) echo "required runtime service is absent" >&2; exit 1 ;;
      esac
    done
    ;;
  materialize-tunnel-token)
    materialize_tunnel_token
    ;;
  validate-tunnel)
    validate_tunnel_runtime
    ;;
  run-tunnel)
    validate_tunnel_runtime
    exec docker run --rm \
      --name "$tunnel_container" \
      --label com.aviasurveil360.runtime=aws-private-pilot-tunnel \
      --pull never \
      --platform linux/arm64 \
      --network host \
      --init \
      --read-only \
      --user 65532:65532 \
      --cap-drop ALL \
      --security-opt no-new-privileges:true \
      --pids-limit 128 \
      --cpus 0.25 \
      --memory 128m \
      --memory-swap 128m \
      --tmpfs /tmp:rw,noexec,nosuid,nodev,size=16m,uid=65532,gid=65532 \
      --log-driver json-file \
      --log-opt max-size=10m \
      --log-opt max-file=3 \
      --stop-timeout 30 \
      --mount "type=bind,source=${tunnel_token_file},target=/run/secrets/cloudflare_tunnel_token,readonly" \
      --env HOME=/tmp \
      "$AVIA_CLOUDFLARED_IMAGE" \
      tunnel --no-autoupdate --edge-ip-version 6 --protocol auto --loglevel info \
      --metrics 127.0.0.1:2000 run --token-file /run/secrets/cloudflare_tunnel_token
    ;;
  stop-tunnel)
    if docker inspect "$tunnel_container" >/dev/null 2>&1; then
      docker stop --time 30 "$tunnel_container"
    fi
    ;;
  tunnel-health)
    metrics_file=$(mktemp /run/aviasurveil360-private-pilot/.cloudflared-metrics.XXXXXX)
    cleanup_metrics() { rm -f "$metrics_file"; }
    trap cleanup_metrics EXIT HUP INT TERM
    connections=0
    if curl --fail --silent --show-error --connect-timeout 2 --max-time 5 "$tunnel_metrics_url" >"$metrics_file"; then
      connections=$(awk '
        /^cloudflared_tunnel_ha_connections(\{[^}]*\})?[[:space:]]/ { total += $NF; found = 1 }
        END { if (found) printf "%.0f", total; else print "0" }
      ' "$metrics_file")
    fi
    case "$connections" in *[!0-9]*|'') connections=0;; esac
    publish_tunnel_connections "$connections"
    if [ "$connections" -lt 4 ]; then
      echo "Cloudflare Tunnel has fewer than four healthy edge connections" >&2
      exit 1
    fi
    ;;
  health)
    for service in $required_services; do
      container=$(compose ps --quiet "$service")
      test -n "$container"
      state=$(docker inspect --format '{{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{end}}' "$container")
      case "$state" in
        "running healthy"|"running ") ;;
        *) echo "required private-pilot service is not healthy" >&2; exit 1 ;;
      esac
    done
    ;;
  drain)
    compose stop --timeout 45 worker
    ;;
  *)
    echo "unsupported private-pilot supervisor action" >&2
    exit 64
    ;;
esac
