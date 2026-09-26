#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(CDPATH= cd -- "${SCRIPT_DIR}/../.." && pwd)"

for script in "${SCRIPT_DIR}"/*.sh; do
    sh -n "$script"
done

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

release_script="${SCRIPT_DIR}/release.sh"
grep -Fq -- '--sort=name' "$release_script" && grep -Fq 'gzip -n' "$release_script" || {
    echo "release archive must be reproducible" >&2
    exit 1
}
echo "ops contract ok"
