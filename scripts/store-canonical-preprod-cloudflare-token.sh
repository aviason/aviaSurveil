#!/usr/bin/env bash
set -euo pipefail

umask 077

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
token_validator="$repository_root/scripts/validate-cloudflare-tunnel-token.mjs"
keychain_writer="$repository_root/scripts/store-cloudflare-tunnel-token-keychain.swift"
hostname="${AVIA_PREPROD_PUBLIC_HOSTNAME:-demo.aviasurveil.com}"
keychain_service="${AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_SERVICE:-com.aviasurveil360.cloudflare-tunnel}"
keychain_account="${AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_ACCOUNT:-$hostname}"
token=""
confirmation=""
writer_build_dir=""
writer_binary=""

cleanup() {
  token=""
  confirmation=""
  case "$writer_build_dir" in
    /tmp/aviasurveil360-cloudflare-keychain.*)
      [[ -z "$writer_binary" || ! -f "$writer_binary" || -L "$writer_binary" ]] ||
        rm -f -- "$writer_binary"
      [[ ! -d "$writer_build_dir" || -L "$writer_build_dir" ]] ||
        rmdir -- "$writer_build_dir" 2>/dev/null || true
      ;;
  esac
}
trap cleanup EXIT

fail() {
  printf 'canonical-preprod-cloudflare-token: %s\n' "$*" >&2
  exit 1
}

[[ "$#" == 0 ]] ||
  fail "accepts no arguments; the tunnel credential must be entered only at the hidden terminal prompt"
[[ "$(uname -s)" == Darwin ]] ||
  fail "this credential helper requires macOS Keychain"
[[ -x /usr/bin/security ]] || fail "macOS Keychain command is unavailable"
command -v node >/dev/null 2>&1 || fail "node is required to validate the connector token"
swiftc_binary="$(command -v swiftc || true)"
[[ -n "$swiftc_binary" ]] || fail "Apple Swift is required to write the long credential to macOS Keychain"
[[ -f "$token_validator" ]] || fail "the connector-token validator is missing"
[[ -f "$keychain_writer" ]] || fail "the macOS Keychain writer is missing"
[[ "$hostname" =~ ^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$ ]] ||
  fail "AVIA_PREPROD_PUBLIC_HOSTNAME must be a lowercase DNS hostname without a scheme, port, or path"
[[ "$keychain_service" =~ ^[A-Za-z0-9._:-]+$ ]] ||
  fail "AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_SERVICE contains unsupported characters"
[[ "$keychain_account" =~ ^[A-Za-z0-9._:@-]+$ ]] ||
  fail "AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_ACCOUNT contains unsupported characters"

printf 'Preparing the native macOS Keychain writer...\n'
writer_build_dir="$(mktemp -d /tmp/aviasurveil360-cloudflare-keychain.XXXXXX)" ||
  fail "could not create the temporary Keychain-writer directory"
writer_binary="$writer_build_dir/keychain-writer"
"$swiftc_binary" -suppress-warnings "$keychain_writer" -o "$writer_binary" ||
  fail "could not compile the native macOS Keychain writer"
chmod 0700 "$writer_binary"

printf 'Paste only the Cloudflare Tunnel connector token for %s (the eyJ... value).\n' "$hostname"
printf 'The terminal will hide the input; do not paste the full cloudflared install command.\n'
printf 'Connector token: '
IFS= read -r -s token </dev/tty || fail "could not read the connector token from the terminal"
printf '\nRetype connector token: '
IFS= read -r -s confirmation </dev/tty || fail "could not confirm the connector token from the terminal"
printf '\n'
[[ -n "$token" ]] || fail "the connector token must not be empty"
[[ "$token" == "$confirmation" ]] || fail "the two connector-token entries do not match"

# Bash's silent read has no 128-byte readpassphrase ceiling. The token remains
# only in this process's memory and is sent through anonymous pipes: first to
# the validator, then to the Security-framework Keychain writer. It never
# enters shell history, argv, the process environment, logs, or repository
# state.
if ! printf '%s' "$token" | node "$token_validator"; then
  fail "the connector token is malformed or truncated; paste the complete uninterrupted eyJ... value"
fi

# Replace only the exact item after both hidden entries and the decoded payload
# have passed validation. Using /usr/bin/security for this non-secret deletion
# avoids the authorization prompt raised when a different process edits the
# legacy CLI-created item. The native writer then creates the long value with
# secret reads restricted to /usr/bin/security.
if /usr/bin/security find-generic-password \
  -a "$keychain_account" \
  -s "$keychain_service" >/dev/null 2>&1; then
  /usr/bin/security delete-generic-password \
    -a "$keychain_account" \
    -s "$keychain_service" >/dev/null ||
    fail "could not replace the existing macOS Keychain item"
fi

if ! printf '%s' "$token" | "$writer_binary" \
  "$keychain_service" "$keychain_account" "$hostname"; then
  fail "could not store the connector token in macOS Keychain"
fi
token=""
confirmation=""

/usr/bin/security find-generic-password \
  -a "$keychain_account" \
  -s "$keychain_service" >/dev/null

# Read the saved value back through a pipe to verify both the Keychain ACL and
# the exact encoded payload. The value is never echoed, added to argv/env, or
# written to repository state.
if ! /usr/bin/security find-generic-password \
  -a "$keychain_account" \
  -s "$keychain_service" \
  -w | node "$token_validator"; then
  fail "the saved connector token is malformed or truncated; rerun this command and paste the complete eyJ... value"
fi

printf 'Cloudflare Tunnel connector token stored in macOS Keychain.\n'
printf 'Service: %s\nAccount: %s\n' "$keychain_service" "$keychain_account"
