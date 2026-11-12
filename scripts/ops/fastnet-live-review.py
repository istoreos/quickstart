#!/usr/bin/env python3
"""Reversible complete-device fixture; credentials remain in memory/private env."""
import argparse, functools, hashlib, http.server, json, os, re, shlex, subprocess, tarfile, threading, time, uuid
from pathlib import Path
p=argparse.ArgumentParser()
p.add_argument('--host',choices=['root@192.168.30.7','root@192.168.30.93'],required=True)
p.add_argument('--candidate',type=Path,required=True)
p.add_argument('--backend',type=Path,required=True)
p.add_argument('--web',type=Path,required=True)
p.add_argument('--output',type=Path,required=True)
p.add_argument('--browser',type=Path,default=Path(__file__).with_name('fastnet-live-home.cjs'))
p.add_argument('--milestone',default='FN09')
p.add_argument('--profile',choices=['home','performance','checks'],default='home')
p.add_argument('--bench',type=Path)
p.add_argument('--delay-core-package',type=float,default=0)
p.add_argument('--persistent-reboot',action='store_true',help='private backup and candidate runtime survive a lab-device reboot')
a=p.parse_args();a.output.mkdir(parents=True,exist_ok=True)
stage=('/root/' if a.persistent_reboot else '/tmp/')+'fastnet-live-'+uuid.uuid4().hex[:10];ssh=['ssh','-o','BatchMode=yes','-o','ConnectTimeout=5',a.host]
arch='x86_64' if a.host.endswith('.7') else 'aarch64_generic'
record=json.loads((a.candidate/'candidate.json').read_text());packages=[x for x in record['packages'] if x['arch'] in ['all',arch]];names={x['name'] for x in packages}
report={'milestone':a.milestone,'status':'in_progress','device':arch,'restored':False,'checks':[],'artifacts':{'backend':hashlib.sha256(a.backend.read_bytes()).hexdigest(),'packages':[x['sha256'] for x in packages]}}
sid=None;token=uuid.uuid4().hex;pid=None;tunnel=None;changed=False;server=None
def remote(script,data=None,timeout=90):
 r=subprocess.run(ssh+[script],input=data,capture_output=True,timeout=timeout)
 if r.returncode:raise RuntimeError('remote operation: '+r.stderr.decode(errors='replace')[:500])
 return r.stdout
def ubus(action,value):return json.loads(remote('ubus -S call session '+action+' '+shlex.quote(json.dumps(value))) or b'{}')
def ownblock(b):return b.splitlines() and b.splitlines()[0].removeprefix('Package: ') in names
def invalidate():remote('rm -f /tmp/luci-indexcache /tmp/luci-indexcache.*; rm -rf /tmp/luci-modulecache/')
class Quiet(http.server.SimpleHTTPRequestHandler):
 def do_GET(self):
  if a.delay_core_package and Path(self.path).name.startswith('fastnet_'):time.sleep(a.delay_core_package)
  return super().do_GET()
 def log_message(self,*args):pass
try:
 active=json.loads(remote('ubus -S call service list \'{"name":"fastnet"}\''));assert not any(i.get('running') for v in active.values() for i in v.get('instances',{}).values()),'another active FastNet owns the test device'
 original_status=remote('cat /usr/lib/opkg/status').decode()
 remote('umask 077; mkdir '+stage+' '+stage+'/backup')
 files={'/etc/config/fastnet','/etc/config/firewall','/etc/config/ucitrack','/usr/share/quickstart/cards.d/fastnet.json','/etc/fastnet/card-state.json','/usr/lib/lua/luci/view/quickstart/main.htm'}
 for x in packages:
  files.update('/'+f for f in x['files'] if not f.startswith('CONTROL/'))
  files.update(f for f in remote('opkg files '+x['name']+' 2>/dev/null || true').decode().splitlines() if f.startswith('/'))
 if a.persistent_reboot:
  assert int(remote("df -k /overlay | tail -1 | awk '{print $4}'"))>24000,'insufficient persistent recovery space'
  files.add('/usr/sbin/quickstart')
 files.update(remote('ls /usr/lib/opkg/info/*fastnet* /etc/rc.d/*fastnet 2>/dev/null || true').decode().splitlines())
 for f in ['/tmp/run/istore-meta/meta/fastnet.json','/var/run/quickstart-installs/app-meta-fastnet.json','/var/run/quickstart-installs/app-meta-fastnet.lock']:files.add(f)
 remote('cat > '+stage+'/paths',('\n'.join(sorted(files))+'\n').encode())
 remote('while IFS= read -r file; do if [ -f "$file" ] || [ -L "$file" ]; then echo "${file#/}"; fi; done < '+stage+'/paths > '+stage+'/existing; tar -czf '+stage+'/backup/files.tar.gz -C / -T '+stage+'/existing')
 directories=['/www/luci-static/quickstart','/etc/quickstart','/tmp/is-root']
 for index,directory in enumerate(directories):
  remote('if [ -d '+directory+' ]; then tar -czf '+stage+'/backup/dir'+str(index)+'.tar.gz -C / '+directory.lstrip('/')+'; else touch '+stage+'/backup/dir'+str(index)+'.absent; fi')
 if a.persistent_reboot:
  remote('cat > '+stage+'/backup/status',original_status.encode())
  recovery='#!/bin/sh\nset -eu\ncd '+stage+"""
/etc/init.d/quickstart stop
/etc/init.d/fastnet stop 2>/dev/null || true
/etc/init.d/fastnet disable 2>/dev/null || true
while IFS= read -r file; do [ ! -f "$file" ] && [ ! -L "$file" ] || rm -f "$file"; done < paths
tar -xzf backup/files.tar.gz -C /
awk 'BEGIN{RS="";ORS=sprintf("%c%c",10,10)}$1!="Package:" || $2!~/^(fastnet|luci-app-fastnet|luci-i18n-fastnet-zh-cn|app-meta-fastnet)$/' /usr/lib/opkg/status > status.restored
awk 'BEGIN{RS="";ORS=sprintf("%c%c",10,10)}$1=="Package:" && $2~/^(fastnet|luci-app-fastnet|luci-i18n-fastnet-zh-cn|app-meta-fastnet)$/' backup/status >> status.restored
cat status.restored > /usr/lib/opkg/status
"""
  for index,directory in enumerate(directories):recovery+='rm -rf '+directory+'; if [ -f backup/dir'+str(index)+'.tar.gz ]; then tar -xzf backup/dir'+str(index)+'.tar.gz -C /; fi\n'
  recovery+='rm -f /tmp/luci-indexcache /tmp/luci-indexcache.*; rm -rf /tmp/luci-modulecache/\n/etc/init.d/quickstart start\nsync\n'
  remote('cat > '+stage+'/RESTORE.sh',recovery.encode());remote('chmod 700 '+stage+'/RESTORE.sh; sync')
 remote('cat > '+stage+'/quickstart',a.backend.read_bytes())
 archive=a.output/'web-candidate.tar.gz'
 with tarfile.open(archive,'w:gz') as tar:
  tar.add(a.web/'luci-static/quickstart',arcname='luci-static/quickstart')
  if a.bench:
   assert (a.bench/'luci-static/quickstart/card-runtime-v1.js').read_bytes()==(a.web/'luci-static/quickstart/card-runtime-v1.js').read_bytes()
   tar.add(a.bench/'luci-static/quickstart/bench',arcname='luci-static/quickstart/bench')
 remote('cat > '+stage+'/web.tar.gz',archive.read_bytes())
 feed=a.candidate/'feed';assert (feed/'Packages').is_file()
 server=http.server.ThreadingHTTPServer(('127.0.0.1',0),functools.partial(Quiet,directory=str(feed)));threading.Thread(target=server.serve_forever,daemon=True).start()
 tunnel=subprocess.Popen(ssh+['-o','ExitOnForwardFailure=yes','-N','-R','127.0.0.1:41609:127.0.0.1:'+str(server.server_port)],stdout=subprocess.DEVNULL,stderr=subprocess.PIPE)
 time.sleep(.5);assert tunnel.poll() is None,'test feed tunnel could not be established'
 assert remote('curl -fsS --max-time 3 http://127.0.0.1:41609/Packages')==(feed/'Packages').read_bytes()
 changed=True
 remote('/etc/init.d/quickstart stop; opkg remove app-meta-fastnet luci-i18n-fastnet-zh-cn luci-app-fastnet fastnet >/dev/null 2>&1 || true')
 # This is an isolated, temporary iStore cache input. No iStore code or system feed changes.
 remote('mkdir -p /tmp/is-root/etc/opkg /tmp/is-root/tmp/opkg-lists; printf "src fn-review http://127.0.0.1:41609/\\n" > /tmp/is-root/etc/opkg/fn-review.conf; opkg print-architecture > /tmp/is-root/etc/opkg/arch.conf; touch /tmp/is-root/.last_force_ts')
 remote('cat > /tmp/is-root/tmp/opkg-lists/fn-review',(feed/'Packages').read_bytes())
 remote('rm -f /usr/share/quickstart/cards.d/fastnet.json /etc/fastnet/card-state.json /var/run/quickstart-installs/app-meta-fastnet.json /var/run/quickstart-installs/app-meta-fastnet.lock; chmod 700 '+stage+'/quickstart; tar -xzf '+stage+'/web.tar.gz -C /www')
 # Home configuration is isolated inside a backed-up directory for this fixture.
 remote('rm -f /etc/quickstart/home-layout.json* /etc/quickstart/home-launcher.json*')
 template=remote('cat /usr/lib/lua/luci/view/quickstart/main.htm').decode();template=re.sub(r'local asset_version = "[^"]+"','local asset_version = "fastnet-'+report['artifacts']['backend'][:12]+'"',template)
 remote('cat > /usr/lib/lua/luci/view/quickstart/main.htm',template.encode());invalidate()
 if a.persistent_reboot:
  remote('mv '+stage+'/quickstart /usr/sbin/quickstart; /etc/init.d/quickstart start; sync')
 else:
  pid=remote('(trap "" HUP; exec '+stage+'/quickstart serve --addr 127.0.0.1:3038 --unix '+stage+'/local.sock) >'+stage+'/backend.log 2>&1 </dev/null & echo $!').decode().strip();assert pid.isdigit()
 for i in range(30):
  if remote('curl -s --max-time 1 -o /dev/null -w "%{http_code}" http://127.0.0.1:3038/cgi-bin/luci/istore/version/ || true').strip()!=b'000':break
  time.sleep(.2)
 else:raise RuntimeError('candidate backend did not listen')
 session=ubus('create',{'timeout':3600});sid=session['ubus_rpc_session'];ubus('set',{'ubus_rpc_session':sid,'values':{'username':'root','token':token}})
 for scope,objects in [('ubus',[['*','*']]),('access-group',[['*','read'],['*','write']]),('uci',[['*','read'],['*','write']])]:ubus('grant',{'ubus_rpc_session':sid,'scope':scope,'objects':objects})
 port=remote('uci -q get uhttpd.main.listen_http').decode().split()[0].rsplit(':',1)[1]
 url='http://'+a.host.split('@')[1]+':'+port
 print('Private candidate ready; exercising real LuCI homepage and package installation',flush=True)
 env={**os.environ,'NODE_PATH':'/tmp/quickstart-playwright/node_modules','DEVICE_URL':url,'DEVICE_SESSION':sid,'DEVICE_CSRF':token,'DEVICE_HOST':a.host,'DEVICE_STAGE':stage,'OUTPUT_DIR':str(a.output),'MILESTONE':a.milestone,'SERVICE_FAULTS':os.environ.get('SERVICE_FAULTS',''),'REBOOT_FIXTURE':'1' if a.persistent_reboot else ''}
 r=subprocess.run(['node',str(a.browser)],env=env,capture_output=True,timeout=1000)
 stdout=r.stdout.decode().replace(sid,'<redacted>').replace(token,'<redacted>');stderr=r.stderr.decode().replace(sid,'<redacted>').replace(token,'<redacted>')
 (a.output/'browser.log').write_text(stdout+stderr)
 assert r.returncode==0,stderr[-500:]
 if a.persistent_reboot:
  boot=json.loads((a.output/'reboot-browser-report.json').read_text());assert boot['status']=='passed' and boot['bootIdBefore']!=boot['bootIdAfter'];job=boot['installationBeforeReboot']
 else:job=json.loads(remote('cat /var/run/quickstart-installs/app-meta-fastnet.json'))
 assert job['state']=='done' and job['exitCode']==0 and job['attempt']==1,job
 report['checks'].append('two live browser tabs produce one real is-opkg installation; durable child journal confirms one attempt')
 raw_state=remote('cat /etc/fastnet/card-state.json 2>/dev/null || true');state=json.loads(raw_state) if raw_state else {'schemaVersion':1,'jobs':[],'latest':None};report['stateFileAbsent']=not bool(raw_state)
 if a.persistent_reboot:
  report['reboot']={k:boot[k] for k in ['bootIdBefore','bootIdAfter','interruptedTaskId']};assert boot['latest']==state['latest']
 elif a.profile=='home':
  browser=json.loads((a.output/'live-browser-report.json').read_text());assert browser['status']=='passed'
  assert browser['publicLatest']==state['latest'],'public measurements differ from same core snapshot'
  task=next(x for x in state['jobs'] if x['task']['id']==browser['taskId']);assert task['task']['state'] in ['succeeded','partial']
  report['sameRun']={'id':browser['taskId'],'taskState':task['task']['state'],'latest':state['latest']}
 else:
  browser=json.loads((a.output/('performance-browser-report.json' if a.profile=='performance' else 'checks-browser-report.json')).read_text());assert browser['status']=='passed';assert len(state['jobs'])<=32;assert all(x['task']['state'] not in ['running','cancel_requested'] for x in state['jobs'])
 (a.output/'core-state.json').write_text(json.dumps(state,ensure_ascii=False,indent=2)+'\n')
 report['checks'].append('browser observations match the same installed core and bounded persisted job history')
 report['status']='passed'
except BaseException as e:
 report['status']='failed';report['failure']=str(e)
 try:
  core=remote('cat /etc/fastnet/card-state.json 2>/dev/null || true');
  if core:(a.output/'failure-core-state.json').write_bytes(core)
  log=remote('cat '+stage+'/backend.log 2>/dev/null || true').decode(errors='replace');(a.output/'backend.log').write_text(log.replace(token,'<redacted>').replace(sid or token,'<redacted>'))
 except BaseException:pass
 raise
finally:
 try:
  if changed:
   # A failed browser may leave installation in progress; do not remove its files mid-install.
   for i in range(90):
    alive=remote('if [ -f /var/run/quickstart-installs/app-meta-fastnet.json ]; then cat /var/run/quickstart-installs/app-meta-fastnet.json; fi')
    if b'"state":"running"' not in alive:break
    time.sleep(1)
   else:raise RuntimeError('installation still running; private backup retained at '+stage)
   remote('/etc/init.d/fastnet stop 2>/dev/null || true; /etc/init.d/fastnet disable 2>/dev/null || true')
   if a.persistent_reboot:remote('/etc/init.d/quickstart stop')
   if pid:remote('if [ "$(readlink /proc/'+pid+'/exe 2>/dev/null)" = '+shlex.quote(stage+'/quickstart')+' ]; then kill '+pid+'; fi')
   current=remote('cat /usr/lib/opkg/status').decode();blocks=[b for b in current.strip().split('\n\n') if not ownblock(b)]+[b for b in original_status.strip().split('\n\n') if ownblock(b)]
   remote('while IFS= read -r file; do [ ! -f "$file" ] && [ ! -L "$file" ] || rm -f "$file"; done < '+stage+'/paths; rm -f /usr/lib/opkg/info/fastnet.* /usr/lib/opkg/info/app-meta-fastnet.* /usr/lib/opkg/info/luci-app-fastnet.* /usr/lib/opkg/info/luci-i18n-fastnet-zh-cn.*; tar -xzf '+stage+'/backup/files.tar.gz -C /')
   remote('cat > /usr/lib/opkg/status',('\n\n'.join(blocks)+'\n\n').encode())
   for index,directory in enumerate(directories):remote('rm -rf '+directory+'; if [ -f '+stage+'/backup/dir'+str(index)+'.tar.gz ]; then tar -xzf '+stage+'/backup/dir'+str(index)+'.tar.gz -C /; fi')
   invalidate();remote('/etc/init.d/quickstart start');report['restored']=True
  if sid and not a.persistent_reboot:ubus('destroy',{'ubus_rpc_session':sid})
  if report['restored'] or not changed:remote('rm -rf '+stage)
 except BaseException as e:report['restorationError']=str(e);report['status']='failed'
 if tunnel:tunnel.terminate();tunnel.wait(timeout=10)
 if server:server.shutdown();server.server_close()
 (a.output/'live-device-report.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({'status':report['status'],'restored':report['restored'],'checks':report['checks']}))
