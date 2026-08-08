#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
api_root="$repo_root/apps/api"
module_path="github.com/MarlonJD/aviaSurveil360/apps/api"
temporary_root="$(mktemp -d "${TMPDIR:-/tmp}/aviasurveil360-normal-artifact.XXXXXX")"
export GOCACHE="$temporary_root/go-build-cache"

cleanup() {
  rm -rf "$temporary_root"
}
trap cleanup EXIT

fail() {
  printf 'normal-artifact-boundary: %s\n' "$*" >&2
  exit 1
}

normal_packages=(
  "./cmd/api"
  "./cmd/worker"
  "./cmd/reminder-scheduler"
  "./cmd/migrate"
)

for package in "${normal_packages[@]}"; do
  dependencies="$(go -C "$api_root" list -deps "$package")"
  if grep -Fxq "$module_path/internal/testprofile" <<<"$dependencies"; then
    fail "$package transitively links internal/testprofile"
  fi
  if grep -Fxq "$module_path/internal/preproddata" <<<"$dependencies"; then
    fail "$package transitively links internal/preproddata"
  fi
  if grep -Fxq "$module_path/internal/preproddata/agacandidatedemo" <<<"$dependencies"; then
    fail "$package transitively links the AGA candidate demo loader"
  fi
  if grep -Fxq "$module_path/internal/agacandidatedemo" <<<"$dependencies"; then
    fail "$package transitively links the tagged AGA candidate demo handler"
  fi
  if grep -Fxq "$module_path/internal/agademoworkspace" <<<"$dependencies"; then
    fail "$package transitively links the tagged AGA demo workspace handler"
  fi
done

go -C "$api_root" build -trimpath -o "$temporary_root/api" ./cmd/api
go -C "$api_root" build -trimpath -o "$temporary_root/worker" ./cmd/worker
go -C "$api_root" build -trimpath -o "$temporary_root/scheduler" ./cmd/reminder-scheduler
go -C "$api_root" build -trimpath -o "$temporary_root/migrate" ./cmd/migrate

for binary in api worker scheduler migrate; do
  if strings "$temporary_root/$binary" |
    grep -Eiq 'internal/testprofile|internal/preproddata|internal/agacandidatedemo|internal/agademoworkspace|aga-candidate-demo|aga-demo-workspace|AGACandidateDemo|AGADemoWorkspace|/__test/reset|Canonical test profile reset'; then
    fail "$binary contains a test-profile, loader, donor, or reset marker"
  fi
done

canonical_dependencies="$(
  go -C "$api_root" list -tags canonicaltest -deps ./cmd/api
)"
if ! grep -Fxq "$module_path/internal/testprofile" <<<"$canonical_dependencies"; then
  fail "canonical-test-api does not positively link internal/testprofile"
fi
go -C "$api_root" build -trimpath -tags canonicaltest \
  -o "$temporary_root/canonical-test-api" \
  ./cmd/api

loader_dependencies="$(go -C "$api_root" list -deps ./cmd/preprod-data-loader)"
if ! grep -Fxq "$module_path/internal/preproddata" <<<"$loader_dependencies"; then
  fail "preprod-data-loader does not positively link internal/preproddata"
fi
go -C "$api_root" build -trimpath \
  -o "$temporary_root/preprod-data-loader" \
  ./cmd/preprod-data-loader

aga_loader_dependencies="$(go -C "$api_root" list -deps ./cmd/preprod-aga-candidate-demo-loader)"
if ! grep -Fxq "$module_path/internal/preproddata/agacandidatedemo" <<<"$aga_loader_dependencies"; then
  fail "preprod AGA candidate demo loader does not positively link its overlay package"
fi
for forbidden_dependency in \
  "$module_path/internal/identity" \
  "$module_path/internal/objectstore" \
  "$module_path/internal/datafeed" \
  "$module_path/internal/checklistintake"; do
  if grep -Fxq "$forbidden_dependency" <<<"$aga_loader_dependencies"; then
    fail "preprod AGA candidate demo loader links forbidden provider/domain dependency $forbidden_dependency"
  fi
done
go -C "$api_root" build -trimpath \
  -o "$temporary_root/preprod-aga-candidate-demo-loader" \
  ./cmd/preprod-aga-candidate-demo-loader

go -C "$api_root" test -count=1 ./internal/httpapi \
  -run '^TestNormalApplicationDoesNotRegisterTestProfileRoutes$'
go -C "$api_root" test -count=1 -tags canonicaltest ./internal/httpapi \
  -run '^TestCanonicalTestBoundary'

printf 'normal-artifact-boundary: ok\n'
