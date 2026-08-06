.DEFAULT_GOAL := help

DEMO_HOST ?= 127.0.0.1
DEMO_PORT ?= 4173
DEMO_STATE_DIR ?= apps/web/.local/demo
DEMO_PID_FILE ?= $(DEMO_STATE_DIR)/vite.pid
DEMO_LOG_FILE ?= $(DEMO_STATE_DIR)/vite.log
DEMO_NODE ?= node
DEMO_VITE_ENTRY ?= apps/web/node_modules/vite/bin/vite.js

AGA_DEMO_STATE_DIR ?= /tmp/avia-surveil360-aga-demo
AGA_DEMO_NODE ?= $(DEMO_NODE)
AGA_DEMO_API_PORT ?= 58081
AGA_DEMO_OIDC_PORT ?= 58082
AGA_DEMO_WEB_PORT ?= 4174
AGA_DEMO_OIDC_HOST ?= 127.0.0.1
AGA_DEMO_WEB_ORIGIN ?= http://127.0.0.1:$(AGA_DEMO_WEB_PORT)
AGA_DEMO_WEB_IMAGE ?= node:24.16.0-alpine3.23@sha256:2bdb65ed1dab192432bc31c95f94155ca5ad7fc1392fb7eb7526ab682fa5bf14

.PHONY: help demo-up demo-down demo-status aga-demo-up aga-demo-down aga-demo-status

help:
	@printf '%s\n' \
		'demo-up      Start the local React demo at http://$(DEMO_HOST):$(DEMO_PORT)' \
		'demo-down    Stop the demo process started by demo-up' \
		'demo-status  Show whether the mock demo URL is responding' \
		'aga-demo-up  Start API + PostgreSQL + Keycloak + HTTP UI with 1,310 questions' \
		'aga-demo-down Stop the disposable API-backed AGA demo and its data' \
		'aga-demo-status Show API/web health and the loaded question count'

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

aga-demo-up:
	@DEMO_NODE="$(AGA_DEMO_NODE)" \
	AVIA_AGA_DEMO_STATE_DIR="$(AGA_DEMO_STATE_DIR)" \
	AVIA_PREPROD_AGA_API_PORT="$(AGA_DEMO_API_PORT)" \
	AVIA_PREPROD_AGA_OIDC_PORT="$(AGA_DEMO_OIDC_PORT)" \
	AVIA_PREPROD_AGA_WEB_PORT="$(AGA_DEMO_WEB_PORT)" \
	AVIA_PREPROD_AGA_OIDC_HOST="$(AGA_DEMO_OIDC_HOST)" \
	AVIA_PREPROD_AGA_WEB_IMAGE="$(AGA_DEMO_WEB_IMAGE)" \
	AVIA_PREPROD_AGA_DEMO_WEB_ORIGIN="$(AGA_DEMO_WEB_ORIGIN)" \
		bash scripts/start-aga-demo.sh

aga-demo-down:
	@DEMO_NODE="$(AGA_DEMO_NODE)" \
	AVIA_AGA_DEMO_STATE_DIR="$(AGA_DEMO_STATE_DIR)" \
		bash scripts/stop-aga-demo.sh

aga-demo-status:
	@DEMO_NODE="$(AGA_DEMO_NODE)" \
	AVIA_AGA_DEMO_STATE_DIR="$(AGA_DEMO_STATE_DIR)" \
		bash scripts/status-aga-demo.sh
