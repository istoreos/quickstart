#!/usr/bin/env python3
"""Reboot acceptance for the production NetPolicy init on the writable x86 VM.
A persistent watchdog restores original config if the controller is interrupted.
"""
import argparse
import hashlib
import json
import shlex
import subprocess
import time
import uuid
from pathlib import Path

parser=argparse.ArgumentParser()
parser.add_argument('--binary')
parser.add_argument('--output',default='docs/archive/evidence/multi-lan/M06/reboot.json')
args=parser.parse_args()
host = "root@192.168.30.7"
token = uuid.uuid4().hex[:5]
base = "/root/.quickstart-multi-lan-boot-" + token
names = ["qsb" + token + side for side in "ab"]
ports = [19765, 19766]
watchdog = "qsnp-recover-" + token
result = {"host": host, "checks": []}

def remote(command, data=None, check=True, timeout=25):
    if isinstance(command, list): command = shlex.join(map(str, command))
    value = subprocess.run(["ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=3", host, command], input=data, text=True, capture_output=True, timeout=timeout)
    if check and value.returncode: raise RuntimeError(value.stderr + value.stdout)
    return value

def put(path, text): remote(["tee", path], text)
def api(index, path, body=None):
    command = ["curl", "-sS", "--max-time", "2", "http://127.0.0.1:%d/api/v1/%s" % (ports[index], path)]
    if body is not None: command += ["-H", "Content-Type: application/json", "--data-binary", json.dumps(body)]
    value = json.loads(remote(command).stdout)
    assert value["status"] == "success", value
    return value["data"]
def wait_engines():
    deadline = time.monotonic() + 45
    while time.monotonic() < deadline:
        try:
            values = [api(index, "system/health") for index in range(2)]
            if all(value["state"] == "healthy" for value in values): return values
        except (RuntimeError, ValueError, AssertionError): pass
        time.sleep(0.25)
    raise RuntimeError("production init did not start both instances")

def hashes(): return remote("sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall").stdout

# This VM has no existing native installation. Never replace an existing service.
remote("test ! -e /etc/init.d/quickstart-netpolicy && test ! -e /etc/config/quickstart-netpolicy && test ! -e /opt/quickstart-netpolicy")
before = hashes()
remote(["mkdir", "-p", base + "/payload"])
remote(["chmod", "700", base])
remote(["cp", "-p", "/etc/config/network", base + "/network.original"])
if args.binary:
    subprocess.run(["scp","-O","-q",args.binary,host+":"+base+"/payload/quickstart-netpolicy"],check=True,timeout=30)
else:
    remote(["cp", "/tmp/quickstart-multi-lan/quickstart-netpolicy", base + "/payload/quickstart-netpolicy"])
result["artifactSha256"]=remote(["sha256sum",base+"/payload/quickstart-netpolicy"]).stdout.split()[0]
init_source = Path("/projects/workspace-linkease-ubuntu/openwrt-apps/quickstart-netpolicy/packaging/quickstart-netpolicy.init").read_text()
put(base + "/quickstart-netpolicy.init", init_source)
recovery = """#!/bin/sh
set -u
/etc/init.d/quickstart-netpolicy stop 2>/dev/null || true
/etc/init.d/quickstart-netpolicy disable 2>/dev/null || true
if [ -f '%s/network.test.sha' ] && sha256sum -c '%s/network.test.sha' >/dev/null 2>&1; then
 cp -p '%s/network.original' /etc/config/network
else
%s
 uci commit network
fi
ubus call network reload '{}' >/dev/null 2>&1 || true
rm -f /etc/init.d/quickstart-netpolicy /etc/config/quickstart-netpolicy /opt/quickstart-netpolicy/current
rmdir /opt/quickstart-netpolicy 2>/dev/null || true
/etc/init.d/%s disable 2>/dev/null || true
rm -f /etc/init.d/%s
printf 'recovered\n' > '%s/recovered'
""" % (base, base, base, "\n".join(" uci -q delete network.%s || true\n uci -q delete network.%s_dev || true" % (name, name) for name in names), watchdog, watchdog, base)
put(base + "/recover.sh", recovery)
put("/etc/init.d/" + watchdog, "#!/bin/sh /etc/rc.common\nSTART=99\nstart() { ( sleep 180; [ -e '" + base + "/controller-present' ] || sh '" + base + "/recover.sh' ) >/dev/null 2>&1 & }\n")
remote(["chmod", "700", base + "/recover.sh", "/etc/init.d/" + watchdog])
remote(["/etc/init.d/" + watchdog, "enable"])
try:
    batch = []
    for index, name in enumerate(names):
        address = ["192.0.2.1", "198.51.100.1"][index]
        batch += ["set network.%s_dev=device" % name, "set network.%s_dev.name='%s'" % (name,name), "set network.%s_dev.type='bridge'" % name, "set network.%s_dev.bridge_empty='1'" % name, "set network.%s=interface" % name, "set network.%s.proto='static'" % name, "set network.%s.device='%s'" % (name,name), "set network.%s.ipaddr='%s'" % (name,address), "set network.%s.netmask='255.255.255.0'" % name, "set network.%s.force_link='1'" % name]
    batch += ["commit network"]
    remote(["uci", "batch"], "\n".join(batch) + "\n")
    remote("sha256sum /etc/config/network > " + shlex.quote(base + "/network.test.sha"))
    remote(["ubus", "call", "network", "reload", "{}"])
    configuration = "config quickstart-netpolicy 'main'\n option enabled '0'\n"
    for index, name in enumerate(names):
        configuration += "config quickstart-netpolicy 'boot_%s'\n option enabled '1'\n option network '%s'\n option interface '%s'\n option port '%d'\n option data_dir '%s/data/%d'\n" % ("a" if index == 0 else "b",name,name,ports[index],base,index)
    put("/etc/config/quickstart-netpolicy", configuration)
    remote(["mkdir", "-p", "/opt/quickstart-netpolicy"])
    remote(["ln", "-s", base + "/payload", "/opt/quickstart-netpolicy/current"])
    remote(["cp", base + "/quickstart-netpolicy.init", "/etc/init.d/quickstart-netpolicy"])
    remote(["chmod", "755", "/etc/init.d/quickstart-netpolicy"])
    remote(["/etc/init.d/quickstart-netpolicy", "enable"])
    remote(["/etc/init.d/quickstart-netpolicy", "start"])
    first = wait_engines()
    for index in range(2):
        mutation = {"expectedRevision":0,"idempotencyKey":"boot-test","owner":{"product":"quickstart","instance":"boot-test"},"policies":[{"device":{"kind":"mac","value":"02:00:00:00:00:55"},"rateLimit":{"enabled":True,"uploadBitsPerSecond":(index+1)*10000000,"downloadBitsPerSecond":(index+1)*20000000}}]}
        api(index,"policies/apply",{"plan":api(index,"policies/plan",mutation),"request":mutation})
    result["beforeBoot"] = first
    # Keep recovery armed across the reboot until both instances are verified.
    remote("reboot", check=False)
    time.sleep(3)
    deadline = time.monotonic() + 100
    while time.monotonic() < deadline:
        if remote("true",check=False,timeout=6).returncode == 0: break
        time.sleep(1)
    else: raise RuntimeError("VM did not return after reboot; watchdog remains armed")
    second = wait_engines()
    for index in range(2):
        assert second[index]["context"]["epoch"] != first[index]["context"]["epoch"]
        stored = api(index,"policies")
        assert stored["revision"] == 1 and len(stored["policies"]) == 1
        assert stored["policies"][0]["rateLimit"]["uploadBitsPerSecond"] == (index+1)*10000000
    remote(["touch", base + "/controller-present"])
    result["afterBoot"] = second
    result["checks"].append("production init resolves two logical netifd networks and starts named instances")
    result["checks"].append("whole-machine reboot restores both independent same-MAC policies with fresh epochs")
finally:
    recovery_result = remote(["sh", base + "/recover.sh"], check=False)
    result["recoveryExitCode"] = recovery_result.returncode
    result["productionConfigRestored"] = before == hashes()
    result["backupDirectory"] = base
    Path(args.output).write_text(json.dumps(result,indent=2) + "\n")
assert result["productionConfigRestored"] and result["recoveryExitCode"] == 0
print(json.dumps({"passed":len(result["checks"]),"evidence":args.output}))
