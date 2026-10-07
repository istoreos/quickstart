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
.PHONY: ops-targets ops-show-selected ops-release ops-init-selected ops-preflight-selected ops-deploy-selected ops-verify-selected ops-rollback-selected ops-ui-smoke-selected test-ops test-lan-lab test-lan-identity-address test-lan-capability-context test-lan-route test-lan-transaction-migration test-lan-groups-schedule test-lan-traffic-quota test-lan-ipv6 test-lan-advanced-tools test-lan-soak-analyzer verify-product verify-lan-device-business smoke-lan-device smoke-lan-device-candidate smoke-bandix bandix-sources bandix-source-check

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
		'  test-lan-identity-address  Verify device identity, hostname, address history, and DHCP recovery' \
		'  test-lan-capability-context Verify DHCP authority, local capability degradation, and native service actions' \
		'  test-lan-route         Verify path precedence, gateway/DNS pairing, diagnostics, and rollback' \
		'  test-lan-transaction-migration Verify migration and task transaction recovery boundaries' \
		'  test-lan-groups-schedule Verify policy precedence, batch reload bounds, and schedules' \
		'  test-lan-traffic-quota Verify quota actions, lazy history, and 24h resource budgets' \
		'  test-lan-ipv6         Verify native MAC policy covers IPv4/IPv6 and real IPv6 throughput' \
		'  test-lan-advanced-tools Verify safe probes, bounded webhooks, and policy import recovery' \
		'  test-lan-soak-analyzer Verify the 24h topology soak release gate' \
		'  verify-product         Validate and regenerate the five-layer product coverage matrix' \
		'  verify-lan-device-business  Run the deterministic LAN-device business completion gate' \
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

test-lan-identity-address:
	./scripts/ops/lan-device-identity-address-smoke.sh

test-lan-capability-context:
	./scripts/ops/lan-device-capability-context-smoke.sh

test-lan-route:
	./scripts/ops/lan-device-route-smoke.sh

test-lan-transaction-migration:
	./scripts/ops/lan-device-transaction-migration-smoke.sh

test-lan-groups-schedule:
	./scripts/ops/lan-device-groups-schedule-smoke.sh

test-lan-traffic-quota:
	./scripts/ops/lan-device-traffic-quota-smoke.sh

test-lan-ipv6:
	./scripts/ops/lan-device-ipv6-smoke.sh

test-lan-advanced-tools:
	./scripts/ops/lan-device-advanced-tools-smoke.sh

test-lan-soak-analyzer:
	./scripts/ops/analyze-lan-device-topology-soak.test.sh

verify-product:
	node ./scripts/validate-lan-device-coverage.mjs --self-test --write

# This target deliberately excludes the wall-clock soak and remote mutation
# smokes. It answers whether the frozen product scope is implemented and
# internally consistent; long-running release observation remains a separate
# gate and authenticated manual checks are documented in docs/.
verify-lan-device-business:
	node ./scripts/validate-lan-device-coverage.mjs --self-test
	cd $(BACKEND_ROOT) && $(GO) test ./...
	node --test web/test/*.test.mjs
	npm --prefix web run tsc
	./scripts/ops/test-ops.sh
	./scripts/ops/lan-device-isolation-lab.test.sh
	./scripts/ops/lan-device-identity-address-smoke.sh
	./scripts/ops/lan-device-capability-context-smoke.sh
	./scripts/ops/lan-device-route-smoke.sh
	./scripts/ops/lan-device-transaction-migration-smoke.sh
	./scripts/ops/lan-device-groups-schedule-smoke.sh
	./scripts/ops/lan-device-traffic-quota-smoke.sh
	./scripts/ops/lan-device-ipv6-smoke.sh
	./scripts/ops/lan-device-advanced-tools-smoke.sh

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
