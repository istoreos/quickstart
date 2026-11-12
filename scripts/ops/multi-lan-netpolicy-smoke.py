#!/usr/bin/env python3
"""Real procd/two-engine smoke test on the two explicitly authorized routers.
Only task-owned empty bridges, files, process instances and BPF hooks are used.
"""
import argparse
import json
import shlex
import subprocess
import time
import uuid
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument("host", choices=["192.168.30.93", "192.168.30.7"])
parser.add_argument("--output", required=True)
parser.add_argument("--backend", choices=["auto", "netlink"], default="auto")
parser.add_argument("--go-test")
args = parser.parse_args()
Path(args.output).parent.mkdir(parents=True, exist_ok=True)
token = uuid.uuid4().hex[:5]
service = "qsnp-test-" + token
base = "/tmp/quickstart-multi-lan/" + service
interfaces = ["qsn" + token + side for side in "abc"]
ports = [18765, 18766, 18767]
binary = "/tmp/quickstart-multi-lan/quickstart-netpolicy"
evidence = {"host": args.host, "service": service, "backend": args.backend, "checks": []}

def remote(command, check=True, timeout=20, input_text=None):
    if isinstance(command, list):
        command = shlex.join(str(value) for value in command)
    result = subprocess.run(["ssh", "-o", "BatchMode=yes", "root@" + args.host, command], capture_output=True, text=True, timeout=timeout, input=input_text)
    if check and result.returncode:
        raise RuntimeError(command + ": " + result.stderr.strip() + result.stdout.strip())
    return result

def rpc(method, document):
    result = remote(["ubus", "-S", "call", "service", method, json.dumps(document)])
    return json.loads(result.stdout or "{}")

def api(index, path, body=None):
    command = ["curl", "-sS", "--max-time", "2", "http://127.0.0.1:%d/api/v1/%s" % (ports[index], path)]
    if body is not None:
        command += ["-H", "Content-Type: application/json", "--data-binary", json.dumps(body)]
    return json.loads(remote(command, timeout=5).stdout)

def data(index, path, body=None):
    envelope = api(index, path, body)
    assert envelope["status"] == "success", envelope
    return envelope["data"]

def healthy(index, previous_epoch=None):
    deadline = time.monotonic() + 20
    last = None
    while time.monotonic() < deadline:
        try:
            last = data(index, "system/health")
            if last["state"] == "healthy" and (previous_epoch is None or last["context"]["epoch"] != previous_epoch):
                return last
        except (RuntimeError, ValueError, AssertionError) as error:
            last = str(error)
        time.sleep(0.15)
    raise RuntimeError("engine readiness timeout: " + str(last))

def specification(index):
    name = "a" if index == 0 else "b"
    return {"command": [binary, "--iface", interfaces[index], "--port", str(ports[index]), "--data-dir", base + "/" + name, "--network-id", "test_" + name, "--instance-id", name, "--enable-traffic", "--tc-backend", args.backend], "respawn": [3600, 1, 5], "term_timeout": 10, "stdout": True, "stderr": True, "netdev": [interfaces[index]]}

specs = {"a": specification(0), "b": specification(1)}

def set_instances(selected):
    rpc("set", {"name": service, "instances": selected})

def fingerprint():
    return remote("sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall").stdout

def bpf_objects(pid):
    result = remote("awk '/^(prog_id|map_id):/ {print $1 $2}' /proc/%d/fdinfo/* 2>/dev/null" % pid)
    return sorted(set(result.stdout.splitlines()))

def mutation(revision, rate, owner="local-router", mac="02:00:00:00:00:55"):
    return {"expectedRevision": revision, "idempotencyKey": uuid.uuid4().hex, "owner": {"product": "quickstart", "instance": owner}, "policies": [{"device": {"kind": "mac", "value": mac}, "rateLimit": {"enabled": True, "uploadBitsPerSecond": rate, "downloadBitsPerSecond": rate * 2}}]}

before = fingerprint()
try:
    # Fail before mutation if an unrelated process already uses the test ports.
    for port in ports:
        probe = remote(["curl", "-sS", "--max-time", "1", "http://127.0.0.1:%d/" % port], check=False)
        assert probe.returncode != 0, "test port already occupied"
    remote(["mkdir", "-p", base])
    for index, interface in enumerate(interfaces):
        remote(["ip", "link", "add", interface, "type", "bridge"])
        remote(["ip", "addr", "add", ["192.0.2.1/24", "198.51.100.1/24", "203.0.113.1/24"][index], "dev", interface])
        remote(["ip", "link", "set", interface, "up"])
    set_instances(specs)
    first, second = healthy(0), healthy(1)
    assert first["context"]["networkId"] == "test_a" and second["context"]["networkId"] == "test_b"
    assert first["pid"] != second["pid"]
    first_objects, second_objects = bpf_objects(first["pid"]), bpf_objects(second["pid"])
    assert first_objects and second_objects and not set(first_objects) & set(second_objects), "engines share BPF objects"
    evidence["initial"] = [first, second]
    evidence["bpfObjects"] = [first_objects, second_objects]
    a_request, b_request = mutation(0, 10_000_000), mutation(0, 20_000_000)
    a_plan = data(0, "policies/plan", a_request)
    assert api(1, "policies/apply", {"plan": a_plan, "request": a_request})["status"] != "success"
    assert data(1, "policies")["revision"] == 0
    data(0, "policies/apply", {"plan": a_plan, "request": a_request})
    b_plan = data(1, "policies/plan", b_request)
    data(1, "policies/apply", {"plan": b_plan, "request": b_request})
    assert data(0, "policies")["policies"][0]["rateLimit"]["uploadBitsPerSecond"] == 10_000_000
    assert data(1, "policies")["policies"][0]["rateLimit"]["uploadBitsPerSecond"] == 20_000_000
    evidence["checks"].append("same MAC independent policy and foreign plan rejection")
    foreign = mutation(1, 5_000_000, "other-owner", "02:00:00:00:00:66")
    data(0, "policies/apply", {"plan": data(0, "policies/plan", foreign), "request": foreign})
    update = mutation(2, 12_000_000)
    data(0, "policies/apply", {"plan": data(0, "policies/plan", update), "request": update})
    assert len(data(0, "policies")["policies"]) == 2
    evidence["checks"].append("owner replacement retains other owner")
    set_instances(specs)
    assert healthy(0)["pid"] == first["pid"] and healthy(1)["pid"] == second["pid"]
    evidence["checks"].append("unchanged procd reload preserves both processes")
    # Noncooperating invocations must fail before attaching an additional BPF object.
    collisions = [(interfaces[0], ports[2], base + "/collision"), (interfaces[2], ports[2], base + "/a"), (interfaces[2], ports[1], base + "/collision")]
    for interface, port, directory in collisions:
        attempt = remote([binary, "--iface", interface, "--port", str(port), "--data-dir", directory, "--enable-traffic"], check=False)
        assert attempt.returncode != 0, "resource conflict was accepted"
    child = interfaces[0] + ".10"
    remote(["ip", "link", "add", "link", interfaces[0], "name", child, "type", "vlan", "id", "10"])
    try:
        remote(["ip", "addr", "add", "198.18.0.1/24", "dev", child])
        remote(["ip", "link", "set", child, "up"])
        attempt = remote([binary, "--iface", child, "--port", str(ports[2]), "--data-dir", base + "/child", "--enable-traffic"], check=False)
        assert attempt.returncode != 0, "parent/child attachment accepted"
    finally:
        remote(["ip", "link", "del", child], check=False)
    assert bpf_objects(second["pid"]) == second_objects
    evidence["checks"].append("interface, parent/child, directory and port conflicts do not change healthy BPF objects")
    remote(["kill", "-KILL", str(first["pid"])])
    recovered = healthy(0, first["context"]["epoch"])
    assert data(0, "policies")["revision"] == 3
    assert healthy(1)["pid"] == second["pid"] and bpf_objects(second["pid"]) == second_objects
    evidence["checks"].append("hard crash restart reclaims own hooks and restores independent policies")
    set_instances({"b": specs["b"]})
    time.sleep(0.4)
    assert healthy(1)["pid"] == second["pid"]
    assert remote(["curl", "-sS", "--max-time", "1", "http://127.0.0.1:%d/api/v1/system/health" % ports[0]], check=False).returncode != 0
    set_instances(specs)
    resumed = healthy(0, recovered["context"]["epoch"])
    assert data(0, "policies")["revision"] == 3
    evidence["checks"].append("single named instance stop/start preserves other process and data")
    remote(["ip", "link", "del", interfaces[0]])
    remote(["ip", "link", "add", interfaces[0], "type", "bridge"])
    remote(["ip", "addr", "add", "192.0.2.1/24", "dev", interfaces[0]])
    remote(["ip", "link", "set", interfaces[0], "up"])
    set_instances(specs)
    recreated = healthy(0, resumed["context"]["epoch"])
    assert recreated["context"]["interfaceIndex"] != resumed["context"]["interfaceIndex"]
    assert healthy(1)["pid"] == second["pid"] and bpf_objects(second["pid"]) == second_objects
    evidence["checks"].append("interface recreation replaces only the affected runtime")
    evidence["final"] = [recreated, healthy(1)]
    if args.go_test:
        config_dir = base + "/client-config"
        remote(["mkdir", "-p", config_dir])
        config = "config quickstart-netpolicy 'a'\n option enabled '1'\n option network 'test_a'\n option interface '%s'\n option port '%s'\n option data_dir '%s/a'\n" % (interfaces[0], ports[0], base)
        remote(["tee", config_dir + "/quickstart-netpolicy"], input_text=config)
        result = remote(["env", "QUICKSTART_MULTI_LAN_NATIVE_LIVE_CONFIG=" + config_dir, "QUICKSTART_MULTI_LAN_NATIVE_NETWORK=test_a", "QUICKSTART_MULTI_LAN_NATIVE_DEVICE=" + interfaces[0], args.go_test, "-test.v", "-test.run", "^TestMultiLANNativeLiveBinding$"])
        evidence["quickstartProvider"] = result.stdout
        evidence["checks"].append("production Quickstart registry/provider applies and verifies against the real additional LAN instance")
finally:
    rpc("delete", {"name": service})
    for interface in interfaces:
        remote(["ip", "link", "del", interface], check=False)
    evidence["productionConfigUnchanged"] = before == fingerprint()
    Path(args.output).write_text(json.dumps(evidence, indent=2) + "\n")
assert evidence["productionConfigUnchanged"]
print(json.dumps({"host": args.host, "passed": len(evidence["checks"]), "evidence": args.output}))
