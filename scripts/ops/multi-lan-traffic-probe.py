#!/usr/bin/env python3
"""Real dual-LAN IPv4/IPv6 packet gate; all links/firewall state live in netns."""
import argparse, hashlib, json, pathlib, shlex, subprocess, time, uuid
p=argparse.ArgumentParser()
p.add_argument("host",choices=["192.168.30.7","192.168.30.93"])
p.add_argument("--binary",required=True);p.add_argument("--go-test",required=True)
p.add_argument("--legacy-enforcement-only",action="store_true")
p.add_argument("--legacy-lifecycle-only",action="store_true")
p.add_argument("--legacy-quota-only",action="store_true");p.add_argument("--legacy-missing-runtime",action="store_true");p.add_argument("--business-only",action="store_true");p.add_argument("--attachment",choices=["ethernet","bridge","vlan"],default="ethernet");p.add_argument("--output",required=True);p.add_argument("--preflight-only",action="store_true")
a=p.parse_args();target="root@"+a.host
live_device="lana"
base="/tmp/quickstart-multi-lan/m07-"+uuid.uuid4().hex[:8]
ns=["qsm7-"+uuid.uuid4().hex[:8]+"-"+side for side in ("r","a","b","w")]
results={"host":a.host,"base":base,"namespaces":ns,"measurements":[],"passed":False,"attachment":a.attachment,"businessOnly":a.business_only,"artifactSha256":{"netpolicy":hashlib.sha256(pathlib.Path(a.binary).read_bytes()).hexdigest(),"goTest":hashlib.sha256(pathlib.Path(a.go_test).read_bytes()).hexdigest()}}
def remote(command,check=True,timeout=60):
 if isinstance(command,list):command=shlex.join(command)
 r=subprocess.run(["ssh","-o","BatchMode=yes",target,command],text=True,capture_output=True,timeout=timeout)
 if check and r.returncode:raise RuntimeError(f"remote command failed ({r.returncode}): {command}\n{r.stdout}\n{r.stderr}")
 return r

def inside(index,args,**kw):return remote(["ip","netns","exec",ns[index]]+args,**kw)
def upload(source,dest):subprocess.run(["scp","-O","-q",str(source),target+":"+dest],check=True,timeout=30)
def phase(name,network="test_a",rate=None):
 env=["env","QUICKSTART_MULTI_LAN_TRAFFIC_LIVE_BASE="+base,"QUICKSTART_MULTI_LAN_TRAFFIC_LIVE_PHASE="+name,"QUICKSTART_MULTI_LAN_TRAFFIC_LIVE_NETWORK="+network]
 env+=["QUICKSTART_MULTI_LAN_TRAFFIC_LIVE_DEVICE="+live_device]
 if rate is not None:env+=["QUICKSTART_MULTI_LAN_TRAFFIC_LIVE_RATE="+str(rate)]
 result=inside(0,env+[base+"/service.test","-test.run","^TestMultiLANLiveTrafficPolicy$","-test.v"],timeout=60)
 for line in result.stdout.splitlines():
  if line.startswith("LEGACY="):results.setdefault("legacyPhases",[]).append(json.loads(line[7:]))
 print(json.dumps({"phase":name,"network":network,"result":result.stdout.strip()}),flush=True)
 return result.stdout

def api(side,path="/api/traffic/devices"):
 text=inside(0,["curl","-fsS","--max-time","4","http://127.0.0.1:"+str(19471+side)+path]).stdout
 return json.loads(text)["data"]
def ping(side=1,family=4,allowed=True):
 address="203.0.113.2" if family==4 else "fd73::2"
 result=inside(side,["ping","-"+str(family),"-c","2","-W","1",address],check=False,timeout=8)
 assert (result.returncode==0)==allowed,(side,family,allowed,result.stdout,result.stderr)
def measure(side,family,reverse=False,rate=100,duration=30,warmup=5,address=None):
 address=address or ("203.0.113.2" if family==4 else "fd73::2")
 cmd=["iperf3","-"+str(family),"-c",address,"-u","-b",str(rate)+"M","-l","1200","-t",str(duration),"-O",str(warmup),"-J"]
 if reverse:cmd+=["-R"]
 r=inside(side,cmd,timeout=duration+warmup+20)
 data=json.loads(r.stdout)
 if "error" in data:raise RuntimeError(data["error"])
 end=data["end"]
 received=end.get("sum_received") or end.get("sum")
 if not received or "bits_per_second" not in received:
  received=end["streams"][0].get("udp")
 value=received["bits_per_second"]/1e6
 row={"side":side,"family":family,"reverse":reverse,"offeredMbit":rate,"seconds":duration,"warmupSeconds":warmup,"receivedMbit":value,"address":address}
 results["measurements"].append(row);print(json.dumps(row),flush=True)
 return value

def counter(side):
 deadline=time.monotonic()+30
 while True:
  data=api(side)
  row=next((r for r in data["d"] if r["mac"].lower()=="02:00:00:00:00:55"),None)
  if row is not None:
   return {"wan":row["w_tx_b"]+row["w_rx_b"],"lan":row["l_tx_b"]+row["l_rx_b"],"epoch":data["counterEpoch"]}
  if time.monotonic()>deadline:raise AssertionError(("device sample did not become ready",data))
  # Neighbor discovery and the traffic poller run independently. Exercise the
  # actual client path while waiting; warming up is never a measured zero.
  ping(side+1,4)
  time.sleep(1)

before=remote("sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall").stdout
try:
 remote(["mkdir","-p",base]);upload(a.binary,base+"/quickstart-netpolicy");upload(a.go_test,base+"/service.test")
 setup=pathlib.Path(__file__).with_name("multi-lan-traffic-namespace.sh");upload(setup,base+"/setup.sh")
 remote(["chmod","700",base+"/quickstart-netpolicy",base+"/service.test"])
 remote(["sh",base+"/setup.sh",base]+ns+[a.attachment])
 deadline=time.monotonic()+25
 while True:
  try:
   health=[api(i,"/api/v1/system/health") for i in (0,1)]
   assert all(h["state"]=="healthy" for h in health)
   results["health"]=health;break
  except Exception:
   if time.monotonic()>deadline:
    raise RuntimeError(remote(["sh","-c","cat "+shlex.quote(base)+"/engine-*.log"],check=False).stdout)
   time.sleep(.25)
 phase("fixtures");inside(0,["sh",base+"/load-firewall"])
 for family in (4,6):ping(1,family);ping(2,family)
 phase("sample");phase("sample","test_b")
 if a.legacy_quota_only:
  actions=("legacy-startup",) if a.legacy_lifecycle_only else (("legacy-enforce",) if a.legacy_enforcement_only else ("legacy-cancel","legacy-reset","legacy-notify","legacy-group-cancel"))
  for action in actions:
   phase("legacy-seed")
   if a.legacy_missing_runtime:
    file=base+"/legacy-quota.json";document=json.loads(remote(["cat",file]).stdout)
    document["quotaRuntime"].pop("mac:02:00:00:00:00:55",None)
    remote(["sh","-c","printf %s "+shlex.quote(json.dumps(document))+" > "+shlex.quote(file)])
    results["legacyRuntimeMissing"]=True
   if a.legacy_enforcement_only:
    file=base+"/legacy-quota.json";document=json.loads(remote(["cat",file]).stdout)
    device="mac:02:00:00:00:00:55"
    document["daily"][device]=[{"start":document["quotaRuntime"][device]["periodStart"],"uploadBytes":200,"downloadBytes":0}]
    remote(["sh","-c","printf %s "+shlex.quote(json.dumps(document))+" > "+shlex.quote(file)])
    results["legacyContinuousEnforcement"]=True
   phase(action)
   phase("legacy-verify")
   inside(0,["sh",base+"/load-firewall"])
   for family in (4,6):ping(1,family,False);ping(2,family)
   if a.legacy_lifecycle_only:
    original_ipv6=remote(["ip","-n",ns[0],"-6","addr","show","dev","lana"]).stdout
    global_ipv6=[line.split()[1] for line in original_ipv6.splitlines() if "inet6 " in line and "scope global" in line]
    remote(["ip","-n",ns[0],"link","set","lana","down"])
    remote(["ip","-n",ns[0],"link","set","lana","name","lana_new"])
    remote(["ip","-n",ns[0],"link","set","lana_new","up"])
    for prefix in global_ipv6:remote(["ip","-n",ns[0],"-6","addr","replace",prefix,"dev","lana_new","nodad"])
    results["reboundIPv6Addresses"]=global_ipv6
    inside(1,["ping","-6","-c","1","-W","1","fd71::1"])
    uci=["uci","-c",base+"/config","-t",base+"/uci-state"]
    remote(uci+["del_list","firewall.local.device=lana"])
    remote(uci+["add_list","firewall.local.device=lana_new"])
    remote(uci+["commit","firewall"])
    live_device="lana_new"
    phase("legacy-rebind");phase("legacy-verify")
    for family in (4,6):ping(1,family,False);ping(2,family)
    results["legacyManualStartupAndRebinding"]=True
   phase("legacy-recover")
   for family in (4,6):ping(1,family);ping(2,family)
  results["legacyQuotaReview"]="preserved across cancellation/reset; explicit restore only; other LAN open"
 elif not a.preflight_only:
  if not a.business_only:
   baseline={}
   for side in (1,2):
    for family in (4,6):baseline[(side,family)]=measure(side,family)
   for side,rate in ((1,3),(2,7)):phase("rate","test_a" if side==1 else "test_b",rate)
   for side,rate in ((1,3),(2,7)):
    for family in (4,6):
     for reverse in (False,True):
      desired=rate*(2 if reverse else 1)
      assert baseline[(side,family)]>=3*desired,("insufficient unlimited baseline",baseline,desired)
      actual=measure(side,family,reverse,rate=desired*3)
      assert .75*desired<=actual<=1.25*desired,("rate outside fixed tolerance",side,family,reverse,actual,desired)
  phase("rate","test_a",0);phase("rate","test_b",0)
  time.sleep(2);initial=[counter(i) for i in (0,1)]
  for family,address in ((4,"198.51.100.55"),(6,"fd72::55")):measure(1,family,rate=30,duration=4,warmup=1,address=address)
  time.sleep(2);after=[counter(i) for i in (0,1)]
  for old,new in zip(initial,after):
   assert new["wan"]-old["wan"]<=65536,("internal traffic counted as WAN",old,new)
   assert new["lan"]-old["lan"]>=1024*1024,("internal packet path did not execute",old,new)
  results["internalTraffic"]={"before":initial,"after":after}
  phase("block")
  for family in (4,6):ping(1,family,False);ping(2,family)
  phase("allow")
  for family in (4,6):ping(1,family)
  phase("quota-start");measure(1,4,rate=20,duration=2,warmup=1);time.sleep(2);phase("quota-record")
  for family in (4,6):ping(1,family,False);ping(2,family)
  old_epoch=counter(0)["epoch"]
  inside(0,["sh","-c",'kill -9 "$(cat '+shlex.quote(base+'/engine-a.pid')+')"'])
  # Reuse the exact owned interface/data directory; process restart preserves policy.
  cmd=[base+"/quickstart-netpolicy","--iface","lana","--port","19471","--data-dir",base+"/a","--network-id","test_a","--instance-id","a","--enable-traffic","--traffic-additional-subnets","198.51.100.0/24,fd72::/64"]
  remote(shlex.join(["ip","netns","exec",ns[0]]+cmd)+" >"+shlex.quote(base+"/engine-a-restart.log")+" 2>&1 &")
  time.sleep(4);assert api(0)["counterEpoch"]!=old_epoch
  phase("quota-record")
  for family in (4,6):ping(1,family,False);ping(2,family)
  phase("block");phase("quota-cancel")
  for family in (4,6):ping(1,family,False);ping(2,family)
  phase("allow")
  for family in (4,6):ping(1,family)
  phase("quota-start")
  for family in (4,6):ping(1,family,False);ping(2,family)
  phase("quota-next-period")
  for family in (4,6):ping(1,family);ping(2,family)
  results["quotaPacketsVerified"]=True
  results["quotaPeriodPacketsVerified"]=True
 results["passed"]=True
except Exception as error:
 results["error"]=str(error);raise
finally:
 results["logs"]=remote(["sh","-c","cat "+shlex.quote(base)+"/engine-*.log 2>/dev/null"],check=False).stdout[-20000:]
 for namespace in ns:
  # Kill only processes in namespaces owned by this invocation.
  remote("for pid in $(ip netns pids "+shlex.quote(namespace)+" 2>/dev/null); do kill -9 \"$pid\" 2>/dev/null || true; done",check=False)
  remote(["ip","netns","del",namespace],check=False)
 after=remote("sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall").stdout
 results["productionConfigBefore"]=before
 results["productionConfigAfter"]=after
 results["productionConfigUnchanged"]=before==after
 pathlib.Path(a.output).parent.mkdir(parents=True,exist_ok=True)
 pathlib.Path(a.output).write_text(json.dumps(results,indent=2)+"\n")
 assert before==after,"production config changed"
