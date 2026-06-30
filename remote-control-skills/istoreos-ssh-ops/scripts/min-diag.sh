#!/bin/sh
# BusyBox/POSIX read-only diagnostics for iStoreOS/OpenWrt.

section() {
	printf '\n===== %s =====\n' "$1"
}

run() {
	section "$1"
	shift
	"$@" 2>&1 || printf 'command failed: %s\n' "$*"
}

have() {
	command -v "$1" >/dev/null 2>&1
}

show_file_redacted() {
	file=$1
	if [ -r "$file" ]; then
		section "$file"
		sed -e '/password/d' \
			-e '/passwd/d' \
			-e '/secret/d' \
			-e '/token/d' \
			-e '/key/d' \
			"$file" 2>&1
	else
		section "$file"
		printf 'not present or not readable\n'
	fi
}

section "identity"
if [ -r /etc/os-release ]; then
	cat /etc/os-release 2>&1
elif [ -r /etc/openwrt_release ]; then
	cat /etc/openwrt_release 2>&1
else
	printf 'no os-release/openwrt_release found\n'
fi

run "uname -a" uname -a

if have ubus; then
	run "ubus call system board" ubus call system board
else
	section "ubus call system board"
	printf 'ubus not found\n'
fi

run "df -h" df -h
run "df -i" df -i
run "mount" mount

if have block; then
	run "block info" block info
else
	section "block info"
	printf 'block not found\n'
fi

if have lsblk; then
	run "lsblk" lsblk
else
	section "lsblk"
	printf 'lsblk not found\n'
fi

section "tool presence"
for tool in uci ubus opkg is-opkg docker dockerd logread service; do
	if have "$tool"; then
		printf '%s: %s\n' "$tool" "$(command -v "$tool")"
	else
		printf '%s: not found\n' "$tool"
	fi
done

if have opkg; then
	run "opkg print-architecture" opkg print-architecture
else
	section "opkg print-architecture"
	printf 'opkg not found\n'
fi

show_file_redacted /etc/config/fstab
show_file_redacted /etc/config/dockerd
show_file_redacted /etc/config/quickstart
show_file_redacted /etc/config/luci
show_file_redacted /etc/config/uhttpd
show_file_redacted /etc/config/rpcd

section "init service status"
for svc in uhttpd rpcd dockerd quickstart; do
	if [ -x "/etc/init.d/$svc" ]; then
		printf '\n--- %s status ---\n' "$svc"
		"/etc/init.d/$svc" status 2>&1 || printf '%s status returned non-zero\n' "$svc"
	else
		printf '\n--- %s status ---\nnot present\n' "$svc"
	fi
done

if have docker; then
	run "docker info" docker info
	run "docker ps -a" docker ps -a
else
	section "docker"
	printf 'docker not found\n'
fi

if have logread; then
	section "logread tail"
	logread 2>&1 | tail -n 80
else
	section "logread tail"
	printf 'logread not found\n'
fi
