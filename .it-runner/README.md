# quickstart `.it-runner`

Tasks are intentionally thin and call stable `make` targets. The default target
deploys a complete locally built Quickstart candidate—backend, web assets,
English catalog, and cache version—to `root@192.168.9.215`.

Release artifacts and deployable binaries are written to the project-level `bin/` directory.

## Common loop

```sh
make ops-show-selected
make ops-preflight-selected
make ops-deploy-selected
make ops-verify-selected
make ops-ui-smoke-selected
```

## Overrides

Committed defaults live in:

- `.it-runner/envs/000-defaults.env`
- `.it-runner/meta/servers/quickstart-dev.env`
- `.it-runner/meta/deployments/quickstart-dev.env`

For local overrides, copy `.it-runner/envs/010-local.env.example` to `.it-runner/envs/010-local.env`. That file is ignored by git. Typical overrides:

```sh
SSH_TARGET=root@192.168.9.215
SSH_PORT=22
DEPLOY_TARGET=quickstart-dev
```

The four-host LAN environment is modeled as two selected deployments:

```sh
DEPLOY_TARGET=quickstart-lan-side make ops-preflight-selected
DEPLOY_TARGET=quickstart-lan-side make ops-deploy-selected

# The gateway target is fail-closed and needs explicit operator intent.
DEPLOY_TARGET=quickstart-lan-main ALLOW_CRITICAL_GATEWAY_DEPLOY=1 make ops-deploy-selected
```

Each complete deployment creates one independently addressable directory below
`/root/quickstart-backups/`. Use its basename as `ROLLBACK_RELEASE`. The main
gateway applies the same explicit-allow guard to rollback.

The committed OpenWrt server metadata sets `SCP_EXTRA_OPTS=-O`, because the
device does not provide the SFTP subsystem used by newer `scp` defaults.

## Runtime artifacts

The following paths are local runtime state and are ignored:

- `.it-runner/logs/`
- `.it-runner/states/`
- `.it-runner/cache/`
