#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "${SCRIPT_DIR}/../.." && pwd)"

for script in "${SCRIPT_DIR}"/*.sh; do
    sh -n "$script"
done

sh "${SCRIPT_DIR}/lan-device-load-check.test.sh"
node "${PROJECT_ROOT}/scripts/analyze-lan-device-business-smoke.test.mjs"

business_smoke="${SCRIPT_DIR}/lan-device-business-smoke.sh"
[ -x "$business_smoke" ] || {
    echo "business smoke must be executable: $business_smoke" >&2
    exit 1
}
grep -Fq 'analyze-lan-device-business-smoke.mjs' "$business_smoke" || {
    echo "business smoke must use the public-contract analyzer" >&2
    exit 1
}
if grep -Eq 'curl .*\-(X|d) |/apply/|/plan/|/etc/init\.d/[^ ]+ (restart|reload|stop)|opkg (install|remove|upgrade)' "$business_smoke"; then
    echo "business smoke must remain read-only on the critical gateway" >&2
    exit 1
fi

grep -Fq 'smoke-lan-device:' "${PROJECT_ROOT}/Makefile" && \
    grep -Fq './scripts/ops/lan-device-topology-smoke.sh' "${PROJECT_ROOT}/Makefile" && \
    grep -Fq './scripts/ops/lan-device-business-smoke.sh' "${PROJECT_ROOT}/Makefile" && \
    grep -Fq './scripts/ops/lan-device-rate-limit-smoke.sh' "${PROJECT_ROOT}/Makefile" || {
    echo "Makefile must expose the fail-closed LAN device smoke gate" >&2
    exit 1
}

rate_limit_smoke="${SCRIPT_DIR}/lan-device-rate-limit-smoke.sh"
[ -x "$rate_limit_smoke" ] || {
    echo "rate-limit smoke must be executable: $rate_limit_smoke" >&2
    exit 1
}
grep -Fq '/rate-limit-settings/plan/' "$rate_limit_smoke" && \
    grep -Fq 'configuration drift' "$rate_limit_smoke" || {
    echo "rate-limit smoke must verify read-only planning and zero configuration drift" >&2
    exit 1
}
if grep -Eq '/apply/|/etc/init\.d/[^ ]+ (restart|reload|stop)|opkg (install|remove|upgrade)' "$rate_limit_smoke"; then
    echo "rate-limit smoke must not mutate the critical gateway" >&2
    exit 1
fi

required_files="
${PROJECT_ROOT}/.it-runner/project.yaml
${PROJECT_ROOT}/.it-runner/envs/000-defaults.env
${PROJECT_ROOT}/.it-runner/meta/servers/quickstart-dev.env
${PROJECT_ROOT}/.it-runner/meta/deployments/quickstart-dev.env
${PROJECT_ROOT}/Makefile
"

for file in $required_files; do
    [ -f "$file" ] || {
        echo "missing required file: $file" >&2
        exit 1
    }
done

for task in ops-targets ops-show-selected ops-release ops-init-selected ops-preflight-selected ops-deploy-selected ops-verify-selected ops-rollback-selected ops-ui-smoke-selected test-ops; do
    task_file="${PROJECT_ROOT}/.it-runner/tasks/${task}/task.yaml"
    [ -f "$task_file" ] || {
        echo "missing task: $task" >&2
        exit 1
    }
    grep -q 'version: "1"' "$task_file" || {
        echo "task.version missing: $task_file" >&2
        exit 1
    }
done

"${SCRIPT_DIR}/show-selected.sh" >/dev/null

resolved_target="$(${SCRIPT_DIR}/show-selected.sh)"
printf '%s\n' "$resolved_target" | grep -q '^SCP_EXTRA_OPTS=-O$' || {
    echo "OpenWrt target must use legacy SCP transport (-O)" >&2
    exit 1
}

side_target="$(DEPLOY_TARGET=quickstart-lan-side "${SCRIPT_DIR}/show-selected.sh")"
main_target="$(DEPLOY_TARGET=quickstart-lan-main "${SCRIPT_DIR}/show-selected.sh")"
printf '%s\n' "$side_target" | grep -Fq 'PROTECT_CRITICAL_GATEWAY=0' && \
    printf '%s\n' "$main_target" | grep -Fq 'PROTECT_CRITICAL_GATEWAY=1' || {
    echo "LAN rollout targets must distinguish bypass and critical gateway safety" >&2
    exit 1
}

grep -Fq 'LOCAL_REMOTE_INSTALLER' "${SCRIPT_DIR}/deploy.sh" && \
    grep -Fq 'EXPECTED_EN_SHA' "${SCRIPT_DIR}/deploy.sh" && \
    grep -Fq 'ALLOW_CRITICAL_GATEWAY_DEPLOY' "${SCRIPT_DIR}/deploy.sh" || {
    echo "deploy must use the complete candidate installer and protect the critical gateway" >&2
    exit 1
}
"${SCRIPT_DIR}/install-candidate-remote.test.sh"

ui_smoke="${SCRIPT_DIR}/lan-device-ui-smoke.sh"
[ -x "$ui_smoke" ] && grep -Fq 'session grant' "$ui_smoke" && grep -Fq 'session destroy' "$ui_smoke" || {
    echo "UI smoke must grant and destroy an ephemeral LuCI session" >&2
    exit 1
}
soak_script="${SCRIPT_DIR}/lan-device-topology-soak.sh"
grep -Fq 'b_pid' "$soak_script" && grep -Fq 'b_start_ticks' "$soak_script" && grep -Fq 'b_rss_kb' "$soak_script" || {
    echo "topology soak must sample both Quickstart processes" >&2
    exit 1
}
grep -Fq 'a_fd' "$soak_script" && grep -Fq 'b_fd' "$soak_script" || {
    echo "topology soak must sample file descriptors" >&2
    exit 1
}

smoke_script="${SCRIPT_DIR}/lan-device-topology-smoke.sh"
for token in A_HOST B_HOST C_HOST D_HOST EXPECTED_PACKAGE_VERSION REQUIRE_PACKAGE_COHERENCE CANDIDATE_MODE \
    LOCAL_CANDIDATE_BINARY LOCAL_CANDIDATE_WEB_DIR EXPECTED_EN_SHA 'candidate asset versions agree' \
    'A is LAN gateway with local DHCP authority' 'B is downstream and route editing is fail-closed' \
    'VIP must have exactly one owner' 'C default route uses B' 'D default route uses A'; do
    grep -Fq "$token" "$smoke_script" || {
        echo "topology smoke missing contract: $token" >&2
        exit 1
    }
done
grep -Fq 'CANDIDATE_MODE=1 REQUIRE_PACKAGE_COHERENCE=0' "${PROJECT_ROOT}/Makefile" || {
    echo "candidate smoke must validate local candidate artifacts without package metadata" >&2
    exit 1
}

release_script="${SCRIPT_DIR}/release.sh"
grep -Fq -- '--sort=name' "$release_script" && grep -Fq 'gzip -n' "$release_script" || {
    echo "release archive must be reproducible" >&2
    exit 1
}
grep -Fq 'formal releases require a clean worktree' "$release_script" && \
    grep -Fq 'web/luci-static' "$release_script" && grep -Fq 'MANIFEST' "$release_script" || {
    echo "formal release must be clean and contain backend, web assets, and a manifest" >&2
    exit 1
}
echo "ops contract ok"
