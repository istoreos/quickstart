# Bandix upstream sources

Bandix is tracked as three pinned Git submodules so development, CI and release
review use the same source as the deployed test package.

| Directory | Upstream | Pin | Deployed role |
| --- | --- | --- | --- |
| `core/` | `timsaya/bandix` | `v0.12.10` | Rust userspace service and eBPF programs |
| `openwrt/` | `timsaya/openwrt-bandix` | `v0.12.10` | OpenWrt package, UCI config and procd init script |
| `luci/` | `timsaya/luci-app-bandix` | `v0.12.11` | LuCI UI and rpcd-to-local-HTTP adapter |

Initialize after cloning Quickstart:

```sh
git submodule update --init --recursive
make bandix-source-check
```

The gitlinks are the version lock. Do not configure these submodules to follow
an upstream branch automatically. Review upstream changes, test an explicit
commit, then update the gitlink in a normal Quickstart change.

The upstream repositories retain their own licenses and notices. Keep them
intact when building or redistributing artifacts.

See [`docs/bandix-source-walkthrough.md`](../../docs/bandix-source-walkthrough.md)
for architecture, data flow, review findings and the recommended reading order.
