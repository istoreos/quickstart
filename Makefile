SHELL := /bin/bash

GO ?= go
PROJECT_ROOT := $(CURDIR)
BACKEND_ROOT := $(PROJECT_ROOT)/backend
BUILD_DIR ?= $(PROJECT_ROOT)/bin
LOCAL_BINARY ?= $(BUILD_DIR)/quickstart.amd64

DEPLOY_TARGET ?=
SSH_TARGET ?=
SSH_PORT ?=
SSH_EXTRA_OPTS ?=
SCP_EXTRA_OPTS ?=
REMOTE_BINARY ?=
REMOTE_TMP ?=
REMOTE_BACKUP_DIR ?=
REMOTE_SERVICE ?=
REMOTE_LOG_COMMAND ?=
ROLLBACK_RELEASE ?=

export PROJECT_ROOT BACKEND_ROOT BUILD_DIR GO LOCAL_BINARY
export DEPLOY_TARGET SSH_TARGET SSH_PORT SSH_EXTRA_OPTS SCP_EXTRA_OPTS
export REMOTE_BINARY REMOTE_TMP REMOTE_BACKUP_DIR REMOTE_SERVICE REMOTE_LOG_COMMAND ROLLBACK_RELEASE

.PHONY: help fmt tidy test build build-web build-amd64 build-arm64 build-armv7 release clean
.PHONY: ops-targets ops-show-selected ops-release ops-init-selected ops-preflight-selected ops-deploy-selected ops-verify-selected ops-rollback-selected ops-ui-smoke-selected test-ops test-lan-lab verify-product smoke-lan-device smoke-lan-device-candidate smoke-bandix bandix-sources bandix-source-check

help:
	@printf '%s\n' \
		'Available targets:' \
		'  fmt                    Run go fmt ./...' \
		'  tidy                   Run go mod tidy' \
		'  test                   Run go test ./...' \
		'  build                  Build linux amd64/arm64/armv7 binaries' \
		'  build-web              Build the production LuCI web archive' \
		'  build-amd64            Build the target-device amd64 binary' \
		'  release                Build a clean, reproducible backend + web release and sha256' \
		'  ops-targets            List .it-runner deployment targets' \
		'  ops-show-selected      Show resolved deployment target' \
		'  ops-release            Alias for release' \
		'  ops-init-selected      Prepare remote target directories' \
		'  ops-preflight-selected Check SSH and remote prerequisites' \
		'  ops-deploy-selected    Build, upload, install, restart, and verify quickstart' \
		'  ops-verify-selected    Verify remote quickstart service' \
		'  ops-rollback-selected  Restore a complete remote backup; set ROLLBACK_RELEASE=<directory name>' \
		'  ops-ui-smoke-selected  Run authenticated responsive UI smoke on the selected target' \
		'  test-ops               Validate deployment scripts and task YAML' \
		'  test-lan-lab           Run the local fault-injection and cleanup laboratory' \
		'  verify-product         Validate and regenerate the five-layer product coverage matrix' \
		'  smoke-lan-device       Fail-closed four-host release smoke; critical gateway stays read-only' \
		'  smoke-lan-device-candidate  Validate local candidate artifacts on the four-host topology; ignore old opkg metadata' \
		'  smoke-bandix           Validate Bandix/eBPF on B; set BANDIX_MUTATION_SMOKE=1 for disposable-rule CRUD' \
		'  bandix-sources         Initialize the pinned Bandix source submodules' \
		'  bandix-source-check    Verify Bandix source versions and clean state'

fmt:
	cd $(BACKEND_ROOT) && $(GO) fmt ./...

tidy:
	cd $(BACKEND_ROOT) && $(GO) mod tidy

test:
	cd $(BACKEND_ROOT) && $(GO) test ./...

build: build-amd64 build-arm64 build-armv7

build-web:
	npm --prefix web run builds

build-amd64:
	GOARCH=amd64 ./scripts/ops/build-backend.sh

build-arm64:
	GOARCH=arm64 ./scripts/ops/build-backend.sh

build-armv7:
	GOARCH=arm GOARM=7 ./scripts/ops/build-backend.sh

release:
	./scripts/ops/release.sh

ops-targets:
	./scripts/ops/targets.sh

ops-show-selected:
	./scripts/ops/show-selected.sh

ops-release: release

ops-init-selected:
	./scripts/ops/init.sh

ops-preflight-selected:
	./scripts/ops/preflight.sh

ops-deploy-selected: build-amd64 build-web
	./scripts/ops/deploy.sh

ops-verify-selected:
	./scripts/ops/verify.sh

ops-rollback-selected:
	./scripts/ops/rollback.sh

ops-ui-smoke-selected:
	./scripts/ops/lan-device-ui-smoke.sh

test-ops:
	./scripts/ops/test-ops.sh

test-lan-lab:
	./scripts/ops/lan-device-isolation-lab.test.sh

verify-product:
	node ./scripts/validate-lan-device-coverage.mjs --self-test --write

smoke-lan-device:
	./scripts/ops/lan-device-topology-smoke.sh
	./scripts/ops/lan-device-business-smoke.sh
	./scripts/ops/lan-device-rate-limit-smoke.sh

smoke-lan-device-candidate:
	CANDIDATE_MODE=1 REQUIRE_PACKAGE_COHERENCE=0 ./scripts/ops/lan-device-topology-smoke.sh
	./scripts/ops/lan-device-business-smoke.sh
	./scripts/ops/lan-device-rate-limit-smoke.sh

smoke-bandix:
	./scripts/ops/bandix-runtime-smoke.sh

bandix-sources:
	git submodule update --init --recursive -- third_party/bandix/core third_party/bandix/openwrt third_party/bandix/luci

bandix-source-check:
	./scripts/check-bandix-sources.sh

clean:
	rm -rf $(BUILD_DIR)
