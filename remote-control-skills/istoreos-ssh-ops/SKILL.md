---
name: istoreos-ssh-ops
description: Safely access iStoreOS/OpenWrt routers over SSH for system identification, read-only diagnostics, disk/rootfs/overlay/Docker/LuCI/opkg/service triage, and routing to specialized skills such as storage path, logs, LuCI recovery, package manager, service manager, Docker basics, Docker data-root migration, and backup/restore workflows.
---

# iStoreOS SSH Ops

## Operating Model

Use this skill when Codex needs shell access to an iStoreOS or OpenWrt router over SSH. Start with read-only identification and diagnostics, then route focused work to the specialized skill that owns the problem area.

Require the user to provide the router host/IP, SSH username, and authentication method at runtime. Never store credentials, private keys, passwords, host fingerprints, or connection strings in this skill, repository files, shell history, logs, or generated config files.

## Safety Rules

Default to read-only commands. Do not change firewall, network, opkg, UCI, filesystem, Docker, service, or boot configuration unless the user explicitly asks for that effect and confirms the exact command set.

Do not perform long-term writes to `/overlay`. Avoid creating persistent files under `/etc`, `/root`, `/usr`, `/overlay`, `/opt`, or Docker data directories during diagnostics. If a temporary helper script is required, place it in `/tmp`, show its contents, execute it, and remove it before finishing.

Treat these as danger operations and require explicit confirmation immediately before execution:

- reboot, poweroff, reset, sysupgrade, firstboot, factory reset
- editing UCI configs or running `uci commit`
- `opkg install`, `opkg remove`, package source changes, or package upgrades
- service enable/disable/restart/reload/stop/start
- firewall, DHCP, DNS, Wi-Fi, WAN, LAN, or VLAN changes
- disk partitioning, formatting, mounting changes, fstab changes, RAID changes
- deleting, moving, or overwriting user data
- Docker daemon configuration changes, Docker data-root migration, container recreation, image pruning, or volume deletion
- LuCI/uhttpd/rpcd authentication or web access changes

Before a confirmed write, state the command, target path or service, expected device effect, rollback path when one exists, and why the change is needed. After a write, run read-only verification and report the result.

## SSH Connection Flow

1. Ask for the router host/IP, username, port if not `22`, and whether the session should use password auth, an SSH key, or an existing SSH agent.
2. Connect with a host-specific command. Examples:

```bash
ssh root@192.168.1.1
ssh -p 2222 root@192.168.1.1
ssh -i "$KEY_PATH" root@192.168.1.1
```

3. If the host key is unknown or changed, stop and ask the user to verify the fingerprint out of band. Do not bypass host key checking for convenience.
4. Once connected, run system identification before deciding on any triage path.
5. Prefer one-shot commands or a short interactive session. Keep credentials out of shell history and generated files.

## System Identification

Run these read-only commands first:

```sh
cat /etc/os-release 2>/dev/null || cat /etc/openwrt_release 2>/dev/null
uname -a
ubus call system board 2>/dev/null
df -h
df -i
mount
block info 2>/dev/null
command -v uci; command -v ubus; command -v opkg; command -v is-opkg
```

If more detail is needed, upload or paste `scripts/min-diag.sh` to `/tmp/min-diag.sh`, run `sh /tmp/min-diag.sh`, capture the output, then remove `/tmp/min-diag.sh`.

## Minimal Diagnostics

Use the bundled diagnostic script for first-pass read-only triage:

```bash
scp scripts/min-diag.sh root@192.168.1.1:/tmp/min-diag.sh
ssh root@192.168.1.1 'sh /tmp/min-diag.sh; rm -f /tmp/min-diag.sh'
```

The script collects OS, kernel, filesystem, mount, block device, LuCI, opkg, Docker, QuickStart, and service status information. It does not write to router state. It redacts obvious secret-like config values from displayed UCI files.

## Triage Priorities

Check root filesystem and overlay space early. Small rootfs is normal on many router images, but a full `/overlay` commonly breaks opkg, LuCI sessions, service starts, Docker writes, and config commits. Disk discovery should distinguish the immutable rootfs, writable overlay, external disks, mounted data paths, and Docker data-root.

Use `references/triage-map.md` to map symptoms to the right follow-up skill.

## Routing To Specialized Skills

- Use `$istoreos-storage-path` for disk discovery, rootfs/overlay interpretation, mount points, fstab, external storage paths, and "where should data live?" questions.
- Use `$istoreos-logs-and-diagnostics` for logread, dmesg, crash loops, kernel errors, service logs, and deeper read-only diagnostic collection.
- Use `$istoreos-luci-recovery` for LuCI web UI failures, uhttpd/rpcd access problems, login/session trouble, or restoring web management access.
- Use `$istoreos-package-manager` for opkg health, package feeds, package install/remove/upgrade planning, and dependency conflicts.
- Use `$istoreos-service-manager` for init scripts, service enablement, service restart/reload plans, and process supervision.
- Use `$istoreos-docker-basics` for Docker daemon status, containers, images, volumes, compose basics, and non-migration Docker triage.
- Use `$istoreos-docker-data-root-migrate` when Docker data-root is on `/overlay`, overlay is filling because of Docker, or Docker storage needs migration to external disk.
- Use `$istoreos-backup-restore` before risky changes, config export/import, backup verification, restore planning, reset, sysupgrade, or disaster recovery.

When the diagnosis crosses skill boundaries, keep SSH access and safety rules from this skill, then load the specialized skill for the domain-specific procedure.
