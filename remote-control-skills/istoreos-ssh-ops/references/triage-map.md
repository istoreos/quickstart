# iStoreOS/OpenWrt SSH Triage Map

Use `$istoreos-ssh-ops` for SSH connection, system identification, safety gates, and first-pass read-only diagnostics. Then route to the narrowest specialized skill.

| Symptom or Finding | Route To | First Read-Only Checks |
| --- | --- | --- |
| `/overlay` is full or near full | `$istoreos-storage-path` | `df -h`, `df -i`, `mount`, `du -xhd1 /overlay 2>/dev/null` only after confirming it is acceptable to run a heavier scan |
| Rootfs looks tiny or read-only | `$istoreos-storage-path` | `df -h`, `mount`, `/etc/os-release`, `ubus call system board` |
| Need to identify disks, partitions, filesystems, or mount points | `$istoreos-storage-path` | `block info`, `mount`, `lsblk` if present, `/etc/config/fstab` |
| Docker data appears to live under `/overlay` | `$istoreos-docker-data-root-migrate` | `/etc/config/dockerd`, `docker info`, `df -h`, `mount` |
| Docker daemon, containers, images, or volumes need basic triage | `$istoreos-docker-basics` | `/etc/init.d/dockerd status`, `docker ps -a`, `docker info`, `logread` tail |
| LuCI is unreachable, login fails, or web UI is broken | `$istoreos-luci-recovery` | `/etc/init.d/uhttpd status`, `/etc/init.d/rpcd status`, `/etc/config/uhttpd`, `/etc/config/rpcd`, `/etc/config/luci`, `logread` tail |
| opkg commands fail, feeds are broken, or packages are missing | `$istoreos-package-manager` | `command -v opkg`, `opkg print-architecture`, `df -h`, `df -i`, `/etc/opkg` inspection only if needed |
| Service is stopped, missing, or crash-looping | `$istoreos-service-manager` | `/etc/init.d/<service> status`, `ps`, `logread` tail |
| Kernel, boot, hardware, or recurring error messages need analysis | `$istoreos-logs-and-diagnostics` | `logread`, `dmesg`, `ubus call system board`, `uname -a` |
| A risky fix, reset, sysupgrade, disk operation, or config edit is being considered | `$istoreos-backup-restore` | Identify backup targets and existing config state before any write |

## Disk And Overlay Notes

Small rootfs is expected on many iStoreOS/OpenWrt images because the base image is read-only. The writable area is usually `/overlay`; when it fills, symptoms often appear unrelated: LuCI sessions fail, opkg cannot install, services cannot write state, Docker fails pulls, and UCI commits fail.

Do not assume an external disk is used just because it is present. Verify the mounted path, filesystem, fstab entry, and service-specific data path. For Docker, confirm the active Docker root directory with `docker info` and compare it to `/etc/config/dockerd`.
