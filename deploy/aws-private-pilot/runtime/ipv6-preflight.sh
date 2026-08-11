#!/bin/sh
set -eu

fail() {
  echo "IPv6 runtime preflight failed: $1" >&2
  exit 1
}

require_hostname() {
  hostname=$1
  case "$hostname" in
    ''|*[!A-Za-z0-9.-]*|.*|*..*|*.) fail "invalid DNS hostname" ;;
  esac
}

resolve_global_ipv6() {
  hostname=$1
  require_hostname "$hostname"
  address=$(getent ahostsv6 "$hostname" | awk '$2 == "STREAM" { print $1; exit }')
  [ -n "$address" ] || fail "$hostname has no usable AAAA result"
  case "$address" in
    ::1|fe80:*|fc*|fd*|*.*) fail "$hostname did not resolve to a global IPv6 address" ;;
  esac
  printf '%s\n' "$address"
}

verify_tls() {
  connect_hostname=$1
  tls_hostname=$2
  port=$3
  transport=$4
  ca_file=${5:-}
  client_certificate=${6:-}
  client_key=${7:-}
  address=$(resolve_global_ipv6 "$connect_hostname")

  set -- openssl s_client \
    -connect "[${address}]:${port}" \
    -servername "$tls_hostname" \
    -verify_hostname "$tls_hostname" \
    -verify_return_error \
    -brief
  if [ -n "$ca_file" ]; then
    set -- "$@" -CAfile "$ca_file"
  else
    set -- "$@" -CApath /etc/ssl/certs
  fi
  if [ "$transport" = starttls ]; then
    set -- "$@" -starttls smtp
  fi
  if [ -n "$client_certificate" ] && [ -n "$client_key" ]; then
    set -- "$@" -cert "$client_certificate" -key "$client_key"
  fi
  timeout 20 "$@" </dev/null >/dev/null 2>&1 || fail "certificate-verified IPv6 TLS failed for $connect_hostname"
}

[ "${1:-}" = runtime ] || fail "unsupported preflight mode"
[ "${AVIA_FORCE_IPV6:?required IPv6-only public egress contract}" = true ] || fail "public egress is not pinned to IPv6"
region=${AVIA_AWS_REGION:?required AWS region}
account=${AVIA_AWS_ACCOUNT_ID:?required AWS account id}

for aws_hostname in \
  "ecr.${region}.api.aws" \
  "ssm.${region}.api.aws" \
  "ssmmessages.${region}.api.aws" \
  "ec2messages.${region}.api.aws" \
  "monitoring.${region}.api.aws" \
  "logs.${region}.api.aws" \
  "${account}.dkr-ecr.${region}.on.aws"
do
  verify_tls "$aws_hostname" "$aws_hostname" 443 tls
done

old_ifs=$IFS
IFS=,
for edge_hostname in ${AVIA_CLOUDFLARE_EDGE_HOSTS:?required Cloudflare edge hostnames}; do
  resolve_global_ipv6 "$edge_hostname" >/dev/null
done
IFS=$old_ifs

smtp_hostname=${AVIA_SMTP_HOSTNAME:?required external SMTP hostname}
smtp_tls_hostname=${AVIA_SMTP_TLS_SERVER_NAME:?required SMTP TLS server name}
smtp_port=${AVIA_SMTP_PORT:?required external SMTP port}
case "${AVIA_SMTP_TRANSPORT:?required SMTP transport}" in
  implicit-tls) [ "$smtp_port" = 465 ] || fail "implicit TLS must use the approved port 465"; smtp_mode=tls ;;
  starttls) [ "$smtp_port" = 587 ] || fail "STARTTLS must use the approved port 587"; smtp_mode=starttls ;;
  *) fail "plaintext or unknown SMTP transport is forbidden" ;;
esac
verify_tls "$smtp_hostname" "$smtp_tls_hostname" "$smtp_port" "$smtp_mode"

echo "verified locally: runtime dependencies expose certificate-verified IPv6 paths"
