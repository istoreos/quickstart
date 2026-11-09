#!/usr/bin/env python3
"""M08 real namespace packet probes; no production UCI/service mutation."""
import argparse,hashlib,json,pathlib,shlex,subprocess,time,uuid
p=argparse.ArgumentParser();p.add_argument("host",choices=["192.168.30.7","192.168.30.93"]);p.add_argument("--binary",required=True);p.add_argument("--go-test",required=True);p.add_argument("--output",required=True);a=p.parse_args()
target="root@"+a.host;base="/tmp/quickstart-multi-lan/m08-"+uuid.uuid4().hex[:8];ns=["qsm8-"+uuid.uuid4().hex[:8]+side for side in ("r","a","b","w")]
result={"host":a.host,"base":base,"namespaces":ns,"passed":False,"checks":[],"artifactSha256":{"netpolicy":hashlib.sha256(pathlib.Path(a.binary).read_bytes()).hexdigest(),"goTest":hashlib.sha256(pathlib.Path(a.go_test).read_bytes()).hexdigest()}}
def remote(command,check=True,timeout=60):
 if isinstance(command,list):command=shlex.join(command)
 r=subprocess.run(["ssh","-o","BatchMode=yes",target,command],capture_output=True,text=True,timeout=timeout)
 if check and r.returncode:raise RuntimeError(command+"\n"+r.stdout+"\n"+r.stderr)
 return r

def inside(index,args,**kw):return remote(["ip","netns","exec",ns[index]]+args,**kw)
def upload(path,name):subprocess.run(["scp","-O","-q",str(path),target+":"+base+"/"+name],check=True,timeout=30)
def tick(check):result["checks"].append(check);print(json.dumps({"passed":check}),flush=True)
def wait_file(name,seconds=20):
 deadline=time.monotonic()+seconds
 while time.monotonic()<deadline:
  r=remote(["cat",base+"/"+name],check=False)
  if r.returncode==0:return r.stdout
  time.sleep(.2)
 raise RuntimeError("timed out waiting for "+name)
def ping(side,family=4):return inside(side,["ping","-"+str(family),"-c","1","-W","1","203.0.113.2" if family==4 else "fd73::2"],check=False,timeout=5).returncode==0
def launch(index,test,extra=()):
 cmd=["ip","netns","exec",ns[index],"env","QUICKSTART_MULTI_LAN_AUX_LIVE_BASE="+base]+list(extra)+[base+"/service.test","-test.run","^"+test+"$","-test.timeout","0","-test.v"]
 remote(shlex.join(cmd)+" >"+shlex.quote(base+"/"+test+"-"+str(index)+".log")+" 2>&1 &")
def run(test,extra=()):
 args=["env","QUICKSTART_MULTI_LAN_AUX_LIVE_BASE="+base]+list(extra)+[base+"/service.test","-test.run","^"+test+"$","-test.v"]
 r=inside(0,args,timeout=60);print(r.stdout,flush=True);return r.stdout

before=remote("sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall").stdout
try:
 remote(["mkdir","-p",base]);upload(a.binary,"quickstart-netpolicy");upload(a.go_test,"service.test");upload(pathlib.Path(__file__).with_name("multi-lan-traffic-namespace.sh"),"setup.sh")
 remote(["chmod","700",base+"/quickstart-netpolicy",base+"/service.test"])
 remote(["sh",base+"/setup.sh",base]+ns+["bridge"])
 remote(["sh","-c","printf 'M08 isolated netns' > "+shlex.quote(base+"/owner")])
 for index,side in ((1,"a"),(2,"b")):launch(index,"TestMultiLANLiveAuxiliaryHTTPServer",["QUICKSTART_MULTI_LAN_AUX_LIVE_SIDE="+side]);wait_file("http-"+side+".ready")
 links=[]
 for index,side in ((1,"a"),(2,"b")):
  raw=inside(index,["ip","-j","-6","addr","show","dev","client"+side]).stdout
  addresses=json.loads(raw)[0]["addr_info"];link=next(row["local"] for row in addresses if row["scope"]=="link");links.append("QUICKSTART_MULTI_LAN_AUX_LIVE_LINK_"+side.upper()+"="+link)
 run("TestMultiLANLiveAuxiliaryProbe",links)
 for side,v4,v6 in (("a","192.0.2.1","fd71::1"),("b","198.51.100.1","fd72::1")):
  rows=[json.loads(line) for line in remote(["cat",base+"/http-"+side+".jsonl"]).stdout.splitlines()]
  assert len(rows)==3,rows
  assert all(row["method"]=="HEAD" for row in rows)
  assert any(row["remote"].startswith(v4+":") for row in rows),rows
  assert any(row["remote"].startswith("["+v6) for row in rows),rows
 result["managementSourcesVerified"]=True;tick("management_ipv4_ipv6_linklocal_correct_source_and_foreign_target_rejected")
 b_pid=remote(["cat",base+"/engine-b.pid"]).stdout.strip()
 inside(0,["sh","-c",'kill -9 "$(cat '+shlex.quote(base+'/engine-a.pid')+')"'])
 remote(["uci","-c",base+"/config","-t",base+"/uci-state","set","quickstart-netpolicy.a.enabled=0"]);remote(["uci","-c",base+"/config","-t",base+"/uci-state","commit","quickstart-netpolicy"])
 cmd=[base+"/quickstart-netpolicy","--iface","lana","--port","19471","--data-dir",base+"/a","--network-id","test_a","--instance-id","a","--enable-traffic","--traffic-additional-subnets","198.51.100.0/24,fd72::/64"]
 script="#!/bin/sh\nset -eu\n"+shlex.join(cmd)+" >"+shlex.quote(base+"/engine-a-reconnected.log")+" 2>&1 &\necho $! > "+shlex.quote(base+"/engine-a.pid")+"\n"
 remote(["sh","-c","cat > "+shlex.quote(base+"/reload-a")+" <<'QS_SCRIPT'\n"+script+"QS_SCRIPT\n"])
 run("TestMultiLANLiveAuxiliaryConnection")
 assert remote(["cat",base+"/engine-b.pid"]).stdout.strip()==b_pid
 inside(0,["kill","-0",b_pid]);tick("selected_network_connection_committed_and_other_engine_pid_unchanged")
 launch(0,"TestMultiLANLiveAuxiliaryScheduler")
 result["schedule"]=json.loads(wait_file("scheduler.ready"))
 assert ping(1) and ping(2),"initial schedule state not open"
 deadline=time.monotonic()+100
 while time.monotonic()<deadline:
  if not ping(1):break
  time.sleep(1)
 else:raise AssertionError("background schedule did not block without page traffic")
 assert not ping(1,6) and ping(2) and ping(2,6)
 tick("background_schedule_blocks_only_selected_lan_after_page_left")
 deadline=time.monotonic()+80
 while time.monotonic()<deadline:
  if ping(1):break
  time.sleep(1)
 else:raise AssertionError("background schedule did not restore after its window")
 assert ping(1,6) and ping(2) and ping(2,6)
 tick("background_schedule_restores_without_browser_or_cross_lan_side_effect")
 result["passed"]=True
except Exception as error:result["error"]=str(error);raise
finally:
 result["logs"]=remote(["sh","-c","cat "+shlex.quote(base)+"/TestMultiLAN*.log 2>/dev/null"],check=False).stdout[-16000:]
 for name in ns:
  remote("for pid in $(ip netns pids "+shlex.quote(name)+" 2>/dev/null); do kill -9 \"$pid\" 2>/dev/null || true; done",check=False);remote(["ip","netns","del",name],check=False)
 after=remote("sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall").stdout;result["productionConfigBefore"]=before;result["productionConfigAfter"]=after;result["productionConfigUnchanged"]=before==after
 pathlib.Path(a.output).parent.mkdir(parents=True,exist_ok=True);pathlib.Path(a.output).write_text(json.dumps(result,indent=2)+"\n")
 assert before==after,"production configuration changed"
