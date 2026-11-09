#!/usr/bin/env python3
"""Collect a read-only network baseline; temporary root sessions only on test hosts."""
import argparse
import json
import subprocess
from datetime import datetime, timezone
from pathlib import Path

TEST_HOSTS = {"root@192.168.30.93", "root@192.168.30.7"}
READ_ONLY_HOST = "root@192.168.30.1"


def remote(host, script):
    reply = subprocess.run(
        ["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", host, "sh", "-s"],
        input=script, text=True, capture_output=True, timeout=90, check=True,
    )
    return reply.stdout


def collect(host):
    if host not in TEST_HOSTS | {READ_ONLY_HOST}:
        raise ValueError("host is outside the user-authorized test inventory")
    fingerprint = "sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall; pidof dnsmasq || true"
    before = remote(host, fingerprint)
    board = json.loads(remote(host, "ubus call system board"))
    interfaces = json.loads(remote(host, "ubus call network.interface dump"))
    result = {
        "host": host, "role": "read_only" if host == READ_ONLY_HOST else "test",
        "board": board,
        "interfaces": [{key: interface.get(key) for key in (
            "interface", "device", "l3_device", "proto", "up", "ipv4-address",
            "ipv6-address", "ipv6-prefix-assignment",
        )} for interface in interfaces.get("interface", [])],
        "resources": remote(host, "grep -E '^Mem(Total|Available):' /proc/meminfo; "
                            "for p in $(pidof quickstart); do grep -E '^VmRSS:|^Threads:' /proc/$p/status; done"),
    }
    if host in TEST_HOSTS:
        # A task-owned, short-lived session; never record its token in evidence.
        script = r'''
set -eu
sid="$(ubus -S call session create '{"timeout":120}' | jsonfilter -e '@.ubus_rpc_session')"
test -n "$sid"
trap 'ubus -S call session destroy "{\"ubus_rpc_session\":\"$sid\"}" >/dev/null' EXIT HUP INT TERM
ubus -S call session set "{\"ubus_rpc_session\":\"$sid\",\"values\":{\"username\":\"root\",\"token\":\"multi-lan-baseline\"}}" >/dev/null
ubus -S call session grant "{\"ubus_rpc_session\":\"$sid\",\"scope\":\"ubus\",\"objects\":[[\"*\",\"*\"]]}" >/dev/null
for endpoint in router-context devices gateway-targets lan-dhcp-settings network-rules device-groups device-traffic floating-gateway; do
    printf '%s\t' "$endpoint"
    curl -sS --max-time 8 -H "X-Forwarded-Sid: $sid" -H "X-Forwarded-Token: multi-lan-baseline" "http://127.0.0.1:3038/cgi-bin/luci/istore/lanctrl/v2/$endpoint/" || true
    printf '\n'
done
'''
        endpoints = {}
        for line in remote(host, script).splitlines():
            if "\t" not in line:
                continue
            name, body = line.split("\t", 1)
            try:
                response = json.loads(body)
            except json.JSONDecodeError:
                endpoints[name] = {"available": False, "reason": "non_json_response"}
                continue
            data = response.get("result")
            summary = {"available": data is not None, "success": response.get("success", 0)}
            if isinstance(data, dict):
                summary["fields"] = sorted(data)
                summary["counts"] = {k: len(v) for k, v in data.items() if isinstance(v, list)}
                for key in ("lan", "dhcpAuthority", "topologyPosition", "editable", "readOnlyReason", "health"):
                    if key in data:
                        summary[key] = data[key]
            endpoints[name] = summary
        result["endpoints"] = endpoints
    after = remote(host, fingerprint)
    result["networkConfigAndDnsmasqUnchanged"] = before == after
    if before != after:
        raise RuntimeError(f"network baseline changed during observation on {host}")
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--host", action="append", choices=sorted(TEST_HOSTS | {READ_ONLY_HOST}))
    args = parser.parse_args()
    hosts = args.host or sorted(TEST_HOSTS)
    report = {"collectedAt": datetime.now(timezone.utc).isoformat(), "hosts": [collect(h) for h in hosts]}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"hosts": len(hosts), "networkConfigAndDnsmasqUnchanged": True, "output": str(args.output)}))


if __name__ == "__main__":
    main()
