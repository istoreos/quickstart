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
REMOTE_BINARY ?=
REMOTE_TMP ?=
REMOTE_BACKUP_DIR ?=
REMOTE_SERVICE ?=
REMOTE_LOG_COMMAND ?=
ROLLBACK_RELEASE ?=

export PROJECT_ROOT BACKEND_ROOT BUILD_DIR GO LOCAL_BINARY
export DEPLOY_TARGET SSH_TARGET SSH_PORT SSH_EXTRA_OPTS
export REMOTE_BINARY REMOTE_TMP REMOTE_BACKUP_DIR REMOTE_SERVICE REMOTE_LOG_COMMAND ROLLBACK_RELEASE

.PHONY: help fmt tidy test build build-amd64 build-arm64 build-armv7 release clean
.PHONY: ops-targets ops-show-selected ops-release ops-init-selected ops-preflight-selected ops-deploy-selected ops-verify-selected ops-rollback-selected test-ops

help:
	@printf '%s\n' \
		'Available targets:' \
		'  fmt                    Run go fmt ./...' \
		'  tidy                   Run go mod tidy' \
		'  test                   Run go test ./...' \
		'  build                  Build linux amd64/arm64/armv7 binaries' \
		'  build-amd64            Build the target-device amd64 binary' \
		'  release                Build release tarball and sha256' \
		'  ops-targets            List .it-runner deployment targets' \
		'  ops-show-selected      Show resolved deployment target' \
		'  ops-release            Alias for release' \
		'  ops-init-selected      Prepare remote target directories' \
		'  ops-preflight-selected Check SSH and remote prerequisites' \
		'  ops-deploy-selected    Build, upload, install, restart, and verify quickstart' \
		'  ops-verify-selected    Verify remote quickstart service' \
		'  ops-rollback-selected  Restore a remote backup; set ROLLBACK_RELEASE=<file>' \
		'  test-ops               Validate deployment scripts and task YAML'

fmt:
	cd $(BACKEND_ROOT) && $(GO) fmt ./...

tidy:
	cd $(BACKEND_ROOT) && $(GO) mod tidy

test:
	cd $(BACKEND_ROOT) && $(GO) test ./...

build: build-amd64 build-arm64 build-armv7

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

ops-deploy-selected: build-amd64
	./scripts/ops/deploy.sh

ops-verify-selected:
	./scripts/ops/verify.sh

ops-rollback-selected:
	./scripts/ops/rollback.sh

test-ops:
	./scripts/ops/test-ops.sh

clean:
	rm -rf $(BUILD_DIR)
