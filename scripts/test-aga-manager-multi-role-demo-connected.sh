#!/usr/bin/env bash
set -euo pipefail

# Reuse the accepted target-bound disposable qualification protocol. The
# manager extension adds its own setup-only inventory receipt, isolated
# text-bearing Playwright project, finalizer, and terminal evidence without
# changing the predecessor's phase journal or canonical snapshot gates.
AVIA_AGA_MANAGER_DEMO_MODE=1 exec bash "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/test-aga-hybrid-demo-workspace-connected.sh" "$@"
