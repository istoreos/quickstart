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
    grep -Fq './scripts/ops/lan-device-business-smoke.sh' "${PROJECT_ROOT}/Makefile" || {
    echo "Makefile must expose the fail-closed LAN device smoke gate" >&2
    exit 1
}

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

for task in ops-targets ops-show-selected ops-release ops-init-selected ops-preflight-selected ops-deploy-selected ops-verify-selected ops-rollback-selected test-ops; do
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

grep -Fq 'cp '\''${REMOTE_BINARY}'\'' \"${REMOTE_BACKUP_DIR}/quickstart.\${stamp}.bak\"' "${SCRIPT_DIR}/deploy.sh" || {
    echo "remote backup filename must expand the deployment timestamp" >&2
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
for token in A_HOST B_HOST C_HOST D_HOST EXPECTED_PACKAGE_VERSION REQUIRE_PACKAGE_COHERENCE \
    'A is LAN gateway with local DHCP authority' 'B is downstream and route editing is fail-closed' \
    'VIP must have exactly one owner' 'C default route uses B' 'D default route uses A'; do
    grep -Fq "$token" "$smoke_script" || {
        echo "topology smoke missing contract: $token" >&2
        exit 1
    }
done

release_script="${SCRIPT_DIR}/release.sh"
grep -Fq -- '--sort=name' "$release_script" && grep -Fq 'gzip -n' "$release_script" || {
    echo "release archive must be reproducible" >&2
    exit 1
}
echo "ops contract ok"
