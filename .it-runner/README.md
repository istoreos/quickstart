# quickstart `.it-runner`

Tasks are intentionally thin and call stable `make` targets. The default target deploys the latest locally built quickstart backend binary to `root@192.168.30.7`.

Release artifacts and deployable binaries are written to the project-level `bin/` directory.

## Common loop

```sh
make ops-show-selected
make ops-preflight-selected
make ops-deploy-selected
make ops-verify-selected
```

## Overrides

Committed defaults live in:

- `.it-runner/envs/000-defaults.env`
- `.it-runner/meta/servers/quickstart-dev.env`
- `.it-runner/meta/deployments/quickstart-dev.env`

For local overrides, copy `.it-runner/envs/010-local.env.example` to `.it-runner/envs/010-local.env`. That file is ignored by git. Typical overrides:

```sh
SSH_TARGET=root@192.168.30.7
SSH_PORT=22
DEPLOY_TARGET=quickstart-dev
```

## Runtime artifacts

The following paths are local runtime state and are ignored:

- `.it-runner/logs/`
- `.it-runner/states/`
- `.it-runner/cache/`
