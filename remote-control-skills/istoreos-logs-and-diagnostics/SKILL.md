---
name: istoreos-logs-and-diagnostics
description: iStoreOS 故障诊断最小信息集与分流（opkg/is-opkg、init.d、df/mount、docker/dockerd、istoreenhance、DNS/网络）；用于失败时先采证据再结论。
---

# iStoreOS Logs & Diagnostics

原则：先采证据，再给结论；先分层，再深入；命令块不超过 6 条。

## Min-DIAG（按需收集）

- 基础：`cat /etc/os-release; uname -a; df -h; mount | head -n 80`
- 包管理失败：完整 stderr/stdout + `opkg --help | head`
- 服务异常：`/etc/init.d/<svc> status || true` + `logread | tail -n 200 | grep -iE '<svc>|error|fail' || true`
- Docker：`docker version` + `docker info | head -n 80` + `uci -q show dockerd | head -n 120`
- 加速：`/etc/init.d/istoreenhance status || true` + `uci -q show istoreenhance || true`
- DNS/Registry：按实际镜像加速源域名/地址探测（示例：`nslookup registry.linkease.net || true`）

## 快速分流

- 安装失败：优先确认是 iStore meta 还是 opkg 包，再决定用 `app-meta-<name>` 还是 `opkg install`
- docker pull 失败：先确认 daemon 通，再检查 `registry_mirrors`，必要时转 `istoreos-docker-acceleration-istoreenhance`
- 磁盘满：先定位哪个挂载点满（df），再决定清理/迁移（转 `istoreos-storage-path`）
