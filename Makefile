.DEFAULT_GOAL := help

DEMO_HOST ?= 127.0.0.1
DEMO_PORT ?= 4173
DEMO_STATE_DIR ?= apps/web/.local/demo
DEMO_PID_FILE ?= $(DEMO_STATE_DIR)/vite.pid
DEMO_LOG_FILE ?= $(DEMO_STATE_DIR)/vite.log
DEMO_NODE ?= node
DEMO_VITE_ENTRY ?= apps/web/node_modules/vite/bin/vite.js
LOCAL_PROFILE ?= full
LOCAL_PROJECT ?= aviasurveil360-task-make
LOCAL_TARGET ?= namibia/demo

.PHONY: help harness-check harness-maintenance qualification-bootstrap-check demo-up demo-down demo-status local-up local-down local-status local-check

help:
	@printf '%s\n' \
		'demo-up      Start the local browser demo at http://$(DEMO_HOST):$(DEMO_PORT)' \
		'demo-down    Stop the browser demo started by demo-up' \
		'demo-status  Show whether the browser demo is responding' \
		'local-up     Start the target-bound connected local stack' \
		'local-down   Stop the target-bound connected local stack and erase task-owned state' \
		'local-status Show connected local stack health' \
		'local-check  Run connected local runtime checks' \
		'qualification-bootstrap-check Run PostgreSQL foundation/roster/catalog replay and permission checks' \
		'harness-check Validate repository-native harness routes and semantic smoke assertions' \
		'harness-maintenance Run local harness maintenance without certification writes'

harness-check:
	node tests/harness-docs-smoke.test.js

harness-maintenance: harness-check
	@echo "verified locally: Surveil harness maintenance completed; certification remains authority-gated"

qualification-bootstrap-check:
	@bash scripts/test-qualification-bootstrap.sh

demo-up:
	@set -eu; \
	mkdir -p "$(DEMO_STATE_DIR)"; \
	if [ -f "$(DEMO_PID_FILE)" ] && kill -0 "$$(cat "$(DEMO_PID_FILE)")" 2>/dev/null; then \
		echo "Demo already running (PID $$(cat "$(DEMO_PID_FILE)")): http://$(DEMO_HOST):$(DEMO_PORT)"; exit 0; \
	fi; \
	rm -f "$(DEMO_PID_FILE)"; \
	if curl --fail --silent --output /dev/null "http://$(DEMO_HOST):$(DEMO_PORT)/"; then \
		echo "Demo already responding at http://$(DEMO_HOST):$(DEMO_PORT)"; exit 0; \
	fi; \
	AVIA_BUILD_PROFILE=demo nohup "$(DEMO_NODE)" "$(DEMO_VITE_ENTRY)" --host "$(DEMO_HOST)" --port "$(DEMO_PORT)" --strictPort >"$(DEMO_LOG_FILE)" 2>&1 < /dev/null & \
	pid=$$!; printf '%s\n' "$$pid" >"$(DEMO_PID_FILE)"; \
	for attempt in $$(seq 1 40); do if curl --fail --silent --output /dev/null "http://$(DEMO_HOST):$(DEMO_PORT)/"; then echo "Demo running at http://$(DEMO_HOST):$(DEMO_PORT)"; exit 0; fi; sleep 0.25; done; \
	echo "Demo did not become ready; see $(DEMO_LOG_FILE)" >&2; exit 1

demo-down:
	@if [ -f "$(DEMO_PID_FILE)" ]; then pid="$$(cat "$(DEMO_PID_FILE)")"; kill "$$pid" 2>/dev/null || true; rm -f "$(DEMO_PID_FILE)"; echo "Stopped demo process $$pid"; else echo "No demo process owned by demo-up is recorded"; fi

demo-status:
	@curl --fail --silent --output /dev/null "http://$(DEMO_HOST):$(DEMO_PORT)/" && echo "Demo responding at http://$(DEMO_HOST):$(DEMO_PORT)"

local-up:
	@AVIA_LOCAL_PROJECT="$(LOCAL_PROJECT)" AVIA_LOCAL_PROFILE="$(LOCAL_PROFILE)" AVIA_LOCAL_TARGET="$(LOCAL_TARGET)" bash scripts/local-stack.sh up "$(LOCAL_PROFILE)"

local-down:
	@AVIA_LOCAL_PROJECT="$(LOCAL_PROJECT)" AVIA_LOCAL_PROFILE="$(LOCAL_PROFILE)" AVIA_LOCAL_TARGET="$(LOCAL_TARGET)" bash scripts/local-stack.sh down "$(LOCAL_PROFILE)"

local-status:
	@AVIA_LOCAL_PROJECT="$(LOCAL_PROJECT)" AVIA_LOCAL_PROFILE="$(LOCAL_PROFILE)" AVIA_LOCAL_TARGET="$(LOCAL_TARGET)" bash scripts/local-stack.sh status "$(LOCAL_PROFILE)"

local-check:
	@AVIA_LOCAL_PROJECT="$(LOCAL_PROJECT)" AVIA_LOCAL_PROFILE="$(LOCAL_PROFILE)" AVIA_LOCAL_TARGET="$(LOCAL_TARGET)" bash scripts/local-stack.sh check "$(LOCAL_PROFILE)"
