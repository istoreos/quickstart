#!/usr/bin/env python3
"""Exercise pinned old/new client and engine combinations on the x86 test VM."""
import json
import shlex
import subprocess
import time
import uuid
from pathlib import Path

host = "root@192.168.30.7"
token = uuid.uuid4().hex[:5]
service = "qsnpc-" + token
base = "/tmp/quickstart-multi-lan/" + service
interfaces = ["qsc" + token + side for side in "ab"]
ports = [19465, 19466]
results = {"host": host, "legacyQuickstartCommit": "66589b22f140615db97851b6c3b1eec7aad975d5", "legacyEngineCommit": "54b5338ad8e03f46ba04387eb232aeb676ff68d2", "checks": []}

def remote(command, input_text=None, check=True):
    if isinstance(command, list): command = shlex.join(map(str, command))
    result = subprocess.run(["ssh", "-o", "BatchMode=yes", host, command], input=input_text, text=True, capture_output=True, timeout=25)
    if check and result.returncode: raise RuntimeError(result.stderr + result.stdout)
    return result

def rpc(method, payload):
    return remote(["ubus", "-S", "call", "service", method, json.dumps(payload)])

def run_current(index, network, instance, unsupported=False):
    directory = base + "/client-" + str(index) + "-" + network
    remote(["mkdir", "-p", directory])
    config = "config quickstart-netpolicy '%s'\n option enabled '1'\n option network '%s'\n option interface '%s'\n option port '%d'\n option data_dir '%s/%d'\n" % (instance, network, interfaces[index], ports[index], base, index)
    remote(["tee", directory + "/quickstart-netpolicy"], config)
    result = remote(["env", "QUICKSTART_MULTI_LAN_NATIVE_LIVE_CONFIG=" + directory, "QUICKSTART_MULTI_LAN_NATIVE_NETWORK=" + network, "QUICKSTART_MULTI_LAN_NATIVE_DEVICE=" + interfaces[index], "QUICKSTART_MULTI_LAN_EXPECT_UNSUPPORTED=" + ("1" if unsupported else "0"), "/tmp/quickstart-multi-lan/service.test", "-test.v", "-test.run", "^TestMultiLANNativeLiveBinding$"])
    return result.stdout

before = remote("sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall").stdout
try:
    remote(["mkdir", "-p", base])
    specs = {}
    for index, interface in enumerate(interfaces):
        assert remote(["curl", "-sS", "--max-time", "1", "http://127.0.0.1:%d/" % ports[index]], check=False).returncode != 0
        remote(["ip", "link", "add", interface, "type", "bridge"])
        remote(["ip", "addr", "add", ["192.0.2.1/24", "198.51.100.1/24"][index], "dev", interface])
        remote(["ip", "link", "set", interface, "up"])
        binary = "/tmp/quickstart-multi-lan/" + ("old/quickstart-netpolicy" if index == 0 else "quickstart-netpolicy")
        command = [binary, "--iface", interface, "--port", str(ports[index]), "--data-dir", base + "/" + str(index), "--enable-traffic"]
        if index == 1: command += ["--network-id", "office", "--instance-id", "office"]
        specs["legacy" if index == 0 else "office"] = {"command": command, "stdout": True, "stderr": True, "term_timeout": 10}
    rpc("set", {"name": service, "instances": specs})
    for port in ports:
        deadline = time.monotonic() + 20
        while True:
            reply = remote(["curl", "-sS", "--max-time", "2", "http://127.0.0.1:%d/api/v1/system/health" % port], check=False)
            if reply.returncode == 0 and json.loads(reply.stdout)["data"]["state"] == "healthy": break
            if time.monotonic() > deadline: raise RuntimeError("compatibility engine not ready")
            time.sleep(0.1)
    results["newClientOldEngine"] = run_current(0, "lan", "main")
    results["checks"].append("new Quickstart registry/provider works with original main v1 engine")
    result = remote(["env", "QUICKSTART_MULTI_LAN_LEGACY_BASE_URL=http://127.0.0.1:19466", "/tmp/quickstart-multi-lan/service.legacy-client.test", "-test.v", "-test.run", "^TestMultiLANLegacyClientLiveBinding$"])
    results["oldClientNewEngine"] = result.stdout
    results["checks"].append("original Quickstart provider works with new v1 engine")
    results["newClientNewEngine"] = run_current(1, "office", "office")
    results["checks"].append("new client and new additional-LAN engine work together")
    results["legacyAdditionalLAN"] = run_current(0, "office", "main", True)
    results["checks"].append("old engine cannot masquerade as an additional-LAN instance")
finally:
    rpc("delete", {"name": service})
    for interface in interfaces: remote(["ip", "link", "del", interface], check=False)
    results["productionConfigUnchanged"] = before == remote("sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall").stdout
    output = Path("docs/archive/evidence/multi-lan/M06/compatibility.json")
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps(results, indent=2) + "\n")
assert results["productionConfigUnchanged"]
print(json.dumps({"passed": len(results["checks"]), "evidence": str(output)}))
