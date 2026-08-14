.DEFAULT_GOAL := help

DEMO_HOST ?= 127.0.0.1
DEMO_PORT ?= 4173
DEMO_STATE_DIR ?= apps/web/.local/demo
DEMO_PID_FILE ?= $(DEMO_STATE_DIR)/vite.pid
DEMO_LOG_FILE ?= $(DEMO_STATE_DIR)/vite.log
DEMO_NODE ?= node
DEMO_VITE_ENTRY ?= apps/web/node_modules/vite/bin/vite.js

CANONICAL_PREPROD_STATE_DIR ?= $(CURDIR)/.local/aviasurveil360-canonical-preprod
CANONICAL_PREPROD_HTTPS_PORT ?= 8445
CANONICAL_PREPROD_CLOUDFLARE_STATE_DIR ?= $(CURDIR)/.local/aviasurveil360-canonical-preprod-cloudflare
CANONICAL_PREPROD_CLOUDFLARE_RUNTIME_DIR ?= $(CURDIR)/.local/aviasurveil360-canonical-preprod-cloudflare-tunnel
CANONICAL_PREPROD_HTTP_PORT ?= 8085
CANONICAL_PREPROD_CLOUDFLARE_DEMO_STATE_DIR ?= $(CURDIR)/.local/aviasurveil360-canonical-preprod-cloudflare-demo
CANONICAL_PREPROD_CLOUDFLARE_DEMO_RUNTIME_DIR ?= $(CURDIR)/.local/aviasurveil360-canonical-preprod-cloudflare-demo-tunnel
CANONICAL_PREPROD_CLOUDFLARE_DEMO_PROJECT ?= aviasurveil360-local-preprod-cloudflare-demo
CANONICAL_PREPROD_CLOUDFLARE_DEMO_HTTP_PORT ?= 8086
CLOUDFLARE_DEMO_HOSTNAME ?= demo.aviasurveil.com
CLOUDFLARE_DEMO_KEYCHAIN_SERVICE ?= com.aviasurveil360.cloudflare-tunnel
CLOUDFLARE_DEMO_KEYCHAIN_ACCOUNT ?= $(CLOUDFLARE_DEMO_HOSTNAME)

.PHONY: help harness-check harness-maintenance demo-up demo-down demo-status preprod-up preprod-down preprod-status preprod-test-fault-restart preprod-cloudflare-up preprod-cloudflare-link preprod-cloudflare-down preprod-cloudflare-status preprod-cloudflare-users preprod-cloudflare-test-panels preprod-cloudflare-test-lifecycle preprod-cloudflare-demo-token preprod-cloudflare-demo-up preprod-cloudflare-demo-down preprod-cloudflare-demo-status preprod-cloudflare-demo-users

help:
	@printf '%s\n' \
		'demo-up      Start the local React demo at http://$(DEMO_HOST):$(DEMO_PORT)' \
		'harness-check Validate repository-native harness routes and semantic smoke assertions' \
		'harness-maintenance Run local harness maintenance without certification writes' \
		'demo-down    Stop the demo process started by demo-up' \
		'demo-status  Show whether the mock demo URL is responding' \
		'preprod-up   Start the canonical disposable local-preprod stack' \
		'preprod-down Stop the canonical disposable local-preprod stack and erase its state' \
		'preprod-status Show canonical local-preprod health and runtime metadata' \
		'preprod-test-fault-restart Run the disposable Task 8 negative/fault/restart matrix' \
		'preprod-cloudflare-up Start an approved disposable anonymous Quick Tunnel profile' \
		'preprod-cloudflare-link Print or start the disposable Quick Tunnel URL' \
		'preprod-cloudflare-down Stop the Quick Tunnel profile and erase all task-owned state' \
		'preprod-cloudflare-status Show Quick Tunnel profile health and ownership status' \
		'preprod-cloudflare-users Print the live URL and privacy-safe demo login matrix' \
		'preprod-cloudflare-test-panels Login as all nine demo users and verify their role panels' \
		'preprod-cloudflare-test-lifecycle Run the connected 1,310-question multi-role lifecycle' \
		'preprod-cloudflare-demo-token Store/rotate the named Tunnel token in macOS Keychain' \
		'preprod-cloudflare-demo-up Publish the local candidate at https://$(CLOUDFLARE_DEMO_HOSTNAME)' \
		'preprod-cloudflare-demo-status Verify the named Tunnel and local candidate' \
		'preprod-cloudflare-demo-users Print the named URL and privacy-safe demo login matrix' \
		'preprod-cloudflare-demo-down Stop named exposure and erase disposable local state'

harness-check:
	node tests/harness-docs-smoke.test.js

harness-maintenance: harness-check
	@echo "verified locally: Surveil harness maintenance completed; certification remains candidate-only without authorized S/A commits and caller-owned HMAC key custody"

demo-up:
	@set -eu; \
	mkdir -p "$(DEMO_STATE_DIR)"; \
	if [ -f "$(DEMO_PID_FILE)" ] && kill -0 "$$(cat "$(DEMO_PID_FILE)")" 2>/dev/null; then \
		echo "Demo already running (PID $$(cat "$(DEMO_PID_FILE)")): http://$(DEMO_HOST):$(DEMO_PORT)"; \
		exit 0; \
	fi; \
	rm -f "$(DEMO_PID_FILE)"; \
	if curl --fail --silent --output /dev/null "http://$(DEMO_HOST):$(DEMO_PORT)/"; then \
		echo "Demo already responding at http://$(DEMO_HOST):$(DEMO_PORT) (process not owned by demo-up)"; \
		exit 0; \
	fi; \
	node_path="$$(command -v "$(DEMO_NODE)" 2>/dev/null || true)"; \
	if [ -z "$$node_path" ] || [ ! -f "$(DEMO_VITE_ENTRY)" ]; then \
		echo "Missing Node/Vite dependencies. Set DEMO_NODE or install apps/web dependencies first." >&2; \
		exit 1; \
	fi; \
	AVIA_BUILD_PROFILE=demo nohup "$(DEMO_NODE)" "$(DEMO_VITE_ENTRY)" --host "$(DEMO_HOST)" --port "$(DEMO_PORT)" --strictPort >"$(DEMO_LOG_FILE)" 2>&1 < /dev/null & \
	pid=$$!; printf '%s\n' "$$pid" >"$(DEMO_PID_FILE)"; \
	for attempt in $$(seq 1 40); do \
		if curl --fail --silent --output /dev/null "http://$(DEMO_HOST):$(DEMO_PORT)/"; then \
			echo "Demo running at http://$(DEMO_HOST):$(DEMO_PORT) (PID $$(cat "$(DEMO_PID_FILE)"))"; \
			exit 0; \
		fi; \
		sleep 0.25; \
	done; \
	echo "Demo did not become ready; see $(DEMO_LOG_FILE)" >&2; \
	if [ -f "$(DEMO_PID_FILE)" ]; then kill "$$(cat "$(DEMO_PID_FILE)")" 2>/dev/null || true; fi; \
	exit 1

demo-down:
	@if [ -f "$(DEMO_PID_FILE)" ]; then \
		pid="$$(cat "$(DEMO_PID_FILE)")"; \
		if kill -0 "$$pid" 2>/dev/null; then kill "$$pid" 2>/dev/null || true; fi; \
		rm -f "$(DEMO_PID_FILE)"; \
		echo "Stopped demo process $$pid"; \
	else \
		echo "No demo process owned by demo-up is recorded"; \
	fi

demo-status:
	@if curl --fail --silent --output /dev/null "http://$(DEMO_HOST):$(DEMO_PORT)/"; then \
		echo "Demo responding at http://$(DEMO_HOST):$(DEMO_PORT)"; \
	else \
		echo "Demo is not responding at http://$(DEMO_HOST):$(DEMO_PORT)"; \
		exit 1; \
	fi

preprod-up:
	@AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_STATE_DIR)" \
	AVIA_PREPROD_HTTPS_PORT="$(CANONICAL_PREPROD_HTTPS_PORT)" \
		bash scripts/start-canonical-preprod.sh

preprod-down:
	@AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_STATE_DIR)" \
	AVIA_PREPROD_HTTPS_PORT="$(CANONICAL_PREPROD_HTTPS_PORT)" \
		bash scripts/stop-canonical-preprod.sh

preprod-status:
	@AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_STATE_DIR)" \
	AVIA_PREPROD_HTTPS_PORT="$(CANONICAL_PREPROD_HTTPS_PORT)" \
		bash scripts/status-canonical-preprod.sh

preprod-test-fault-restart:
	@bash scripts/test-canonical-preprod-fault-restart.sh

preprod-cloudflare-up:
	@AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_STATE_DIR)" \
	AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_RUNTIME_DIR)" \
	AVIA_PREPROD_HTTP_PORT="$(CANONICAL_PREPROD_HTTP_PORT)" \
		bash scripts/start-canonical-preprod-cloudflare.sh

preprod-cloudflare-link:
	@AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_STATE_DIR)" \
	AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_RUNTIME_DIR)" \
	AVIA_PREPROD_HTTP_PORT="$(CANONICAL_PREPROD_HTTP_PORT)" \
		bash scripts/link-canonical-preprod-cloudflare.sh

preprod-cloudflare-down:
	@AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_STATE_DIR)" \
	AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_RUNTIME_DIR)" \
	AVIA_PREPROD_HTTP_PORT="$(CANONICAL_PREPROD_HTTP_PORT)" \
		bash scripts/stop-canonical-preprod-cloudflare.sh

preprod-cloudflare-status:
	@AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_STATE_DIR)" \
	AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_RUNTIME_DIR)" \
	AVIA_PREPROD_HTTP_PORT="$(CANONICAL_PREPROD_HTTP_PORT)" \
		bash scripts/status-canonical-preprod-cloudflare.sh

preprod-cloudflare-users:
	@AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_STATE_DIR)" \
	AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_RUNTIME_DIR)" \
	AVIA_PREPROD_HTTP_PORT="$(CANONICAL_PREPROD_HTTP_PORT)" \
		bash scripts/show-canonical-preprod-cloudflare-users.sh

preprod-cloudflare-test-panels:
	@AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_STATE_DIR)" \
	AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_RUNTIME_DIR)" \
	AVIA_PREPROD_HTTP_PORT="$(CANONICAL_PREPROD_HTTP_PORT)" \
		bash scripts/test-canonical-preprod-cloudflare-panels.sh

preprod-cloudflare-test-lifecycle:
	@AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_STATE_DIR)" \
	AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_RUNTIME_DIR)" \
		bash scripts/test-canonical-preprod-cloudflare-lifecycle.sh

preprod-cloudflare-demo-token:
	@AVIA_PREPROD_PUBLIC_HOSTNAME="$(CLOUDFLARE_DEMO_HOSTNAME)" \
	AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_SERVICE="$(CLOUDFLARE_DEMO_KEYCHAIN_SERVICE)" \
	AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_ACCOUNT="$(CLOUDFLARE_DEMO_KEYCHAIN_ACCOUNT)" \
		bash scripts/store-canonical-preprod-cloudflare-token.sh

preprod-cloudflare-demo-up:
	@AVIA_PREPROD_CLOUDFLARE_MODE=named \
	AVIA_PREPROD_PUBLIC_HOSTNAME="$(CLOUDFLARE_DEMO_HOSTNAME)" \
	AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_SERVICE="$(CLOUDFLARE_DEMO_KEYCHAIN_SERVICE)" \
	AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_ACCOUNT="$(CLOUDFLARE_DEMO_KEYCHAIN_ACCOUNT)" \
	AVIA_CANONICAL_PREPROD_PROJECT="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_PROJECT)" \
	AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_STATE_DIR)" \
	AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_RUNTIME_DIR)" \
	AVIA_PREPROD_HTTP_PORT="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_HTTP_PORT)" \
		bash scripts/start-canonical-preprod-cloudflare.sh

preprod-cloudflare-demo-status:
	@AVIA_PREPROD_CLOUDFLARE_MODE=named \
	AVIA_PREPROD_PUBLIC_HOSTNAME="$(CLOUDFLARE_DEMO_HOSTNAME)" \
	AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_SERVICE="$(CLOUDFLARE_DEMO_KEYCHAIN_SERVICE)" \
	AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_ACCOUNT="$(CLOUDFLARE_DEMO_KEYCHAIN_ACCOUNT)" \
	AVIA_CANONICAL_PREPROD_PROJECT="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_PROJECT)" \
	AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_STATE_DIR)" \
	AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_RUNTIME_DIR)" \
	AVIA_PREPROD_HTTP_PORT="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_HTTP_PORT)" \
		bash scripts/status-canonical-preprod-cloudflare.sh

preprod-cloudflare-demo-users:
	@AVIA_PREPROD_CLOUDFLARE_MODE=named \
	AVIA_PREPROD_PUBLIC_HOSTNAME="$(CLOUDFLARE_DEMO_HOSTNAME)" \
	AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_SERVICE="$(CLOUDFLARE_DEMO_KEYCHAIN_SERVICE)" \
	AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_ACCOUNT="$(CLOUDFLARE_DEMO_KEYCHAIN_ACCOUNT)" \
	AVIA_CANONICAL_PREPROD_PROJECT="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_PROJECT)" \
	AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_STATE_DIR)" \
	AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_RUNTIME_DIR)" \
	AVIA_PREPROD_HTTP_PORT="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_HTTP_PORT)" \
		bash scripts/show-canonical-preprod-cloudflare-users.sh

preprod-cloudflare-demo-down:
	@AVIA_PREPROD_CLOUDFLARE_MODE=named \
	AVIA_PREPROD_PUBLIC_HOSTNAME="$(CLOUDFLARE_DEMO_HOSTNAME)" \
	AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_SERVICE="$(CLOUDFLARE_DEMO_KEYCHAIN_SERVICE)" \
	AVIA_CLOUDFLARE_TUNNEL_KEYCHAIN_ACCOUNT="$(CLOUDFLARE_DEMO_KEYCHAIN_ACCOUNT)" \
	AVIA_CANONICAL_PREPROD_PROJECT="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_PROJECT)" \
	AVIA_CANONICAL_PREPROD_STATE_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_STATE_DIR)" \
	AVIA_CANONICAL_PREPROD_TUNNEL_RUNTIME_DIR="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_RUNTIME_DIR)" \
	AVIA_PREPROD_HTTP_PORT="$(CANONICAL_PREPROD_CLOUDFLARE_DEMO_HTTP_PORT)" \
		bash scripts/stop-canonical-preprod-cloudflare.sh
