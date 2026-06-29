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
echo "ops contract ok"
