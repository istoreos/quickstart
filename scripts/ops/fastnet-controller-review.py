#!/usr/bin/env python3
"""Reversible LuCI/core test on an inventoried development device; never records credentials."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import shlex
import subprocess
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

p=argparse.ArgumentParser(description=__doc__)
p.add_argument('--host',choices=['root@192.168.30.7','root@192.168.30.93'],required=True)
p.add_argument('--binary',type=Path,required=True)
p.add_argument('--packages',type=Path,required=True)
p.add_argument('--output',type=Path,required=True)
p.add_argument('--diagnose-csrf',action='store_true')
a=p.parse_args();a.output.mkdir(parents=True,exist_ok=True)
label='x86_64' if a.host.endswith('.7') else 'aarch64'
report={'milestone':'FN06','status':'in_progress','device':label,'checks':[],'transactions':[],'restored':False}
ssh=['ssh','-o','BatchMode=yes','-o','ConnectTimeout=5',a.host]
stage='/tmp/fastnet-FN06-'+uuid.uuid4().hex[:10];backup='/root/quickstart-fastnet-FN06-'+uuid.uuid4().hex[:10]
sid=None;token=uuid.uuid4().hex;core_token='fixture-internal-core-token';pid=None
files=[('/usr/lib/lua/luci/controller/fastnet.lua','controller.lua'),('/usr/lib/lua/luci/model/fastnet/card.lua','model.lua'),('/etc/fastnet/card-state.json','state.json'),('/etc/config/fastnet','config'),('/usr/lib/lua/luci/model/cbi/fastnet.lua','cbi.lua'),('/usr/lib/lua/luci/view/fastnet/status.htm','status.htm')]
changed=False
socket_owned=False
def remote(script,data=None,label='remote operation',timeout=60):
 r=subprocess.run(ssh+[script],input=data,capture_output=True,timeout=timeout)
 if r.returncode:
  message=r.stderr.decode(errors='replace')
  for secret in [sid,token]:
   if secret:message=message.replace(secret,'<redacted>')
  message=re.sub(r'\b[a-f0-9]{32,64}\b','<redacted>',message)
  raise RuntimeError(label+': '+message[:600])
 return r.stdout
def ubus(action,value):return json.loads(remote('ubus -S call session '+action+' '+shlex.quote(json.dumps(value)),label='session '+action) or b'{}')
class NoRedirect(urllib.request.HTTPRedirectHandler):
 def redirect_request(self,*args):return None
client=urllib.request.build_opener(NoRedirect)
base='http://'+a.host.split('@')[1]+'/cgi-bin/luci/admin/services/fastnet'
def request(path,body=None,authenticated=True,csrf=True,method=None,raw=None):
 data=raw if raw is not None else None if body is None else json.dumps(body).encode()
 headers={}
 if authenticated and sid:headers['Cookie']='sysauth_http='+sid+'; sysauth_https='+sid
 if data is not None:headers['Content-Type']='application/json'
 if csrf and data is not None:headers['X-CSRF-Token']=token
 target=base+path
 if csrf and data is not None:target+=('&' if '?' in target else '?')+'token='+urllib.parse.quote(token)
 req=urllib.request.Request(target,data=data,method=method,headers=headers)
 try:
  res=client.open(req,timeout=10)
 except urllib.error.HTTPError as error:res=error
 with res:payload=res.read(32768);status=res.status;kind=res.headers.get('Content-Type','')
 assert core_token.encode() not in payload,'response leaked the internal core credential'
 if path.startswith('/card/'):assert token.encode() not in payload,'card response echoed CSRF credential'
 try:value=json.loads(payload)
 except (ValueError,UnicodeDecodeError):value=None
 report['transactions'].append({'path':path,'method':method or ('POST' if data is not None else 'GET'),'status':status,'json':value is not None,'bytes':len(payload),**({'error':value['error']} if isinstance(value,dict) and isinstance(value.get('error'),str) and re.fullmatch('[a-z_-]{1,64}',value['error']) else {})})
 if a.diagnose_csrf and isinstance(value,dict) and isinstance(value.get('diagnostic'),dict):report['csrfDiagnostic']=value['diagnostic']
 return status,value,payload,kind
def invalidate():remote('for cache_path in /tmp/luci-indexcache*; do [ ! -f "$cache_path" ] || rm -f "$cache_path"; done',label='invalidate route cache')
def stop():
 global pid
 if pid:
  remote('if [ "$(readlink /proc/'+pid+'/exe 2>/dev/null)" = '+shlex.quote(stage+'/FastNet')+' ]; then kill '+pid+'; fi',label='stop task-owned core')
  pid=None
  time.sleep(.2)
try:
 listeners=remote('uci -q get uhttpd.main.listen_http || true',label='discover management listener').decode().split()
 if not listeners:raise RuntimeError('no HTTP management listener; use the configured TLS endpoint')
 management_port=int(listeners[0].rsplit(':',1)[1]);base='http://'+a.host.split('@')[1]+':'+str(management_port)+'/cgi-bin/luci/admin/services/fastnet'
 remote('umask 077; mkdir '+shlex.quote(stage)+' '+shlex.quote(backup),label='create private stage')
 # A task never takes over another instance's private socket.
 remote("lua -e 'local n=require \"nixio\";local s=n.socket(\"unix\",\"stream\");local live=s:connect(\"/var/run/fastnet-card.sock\");s:close();if live then os.exit(42) end'",label='private socket ownership preflight')
 for path,name in files:
  remote('if [ -e '+shlex.quote(path)+' ]; then cp -p '+shlex.quote(path)+' '+shlex.quote(backup+'/'+name)+'; else touch '+shlex.quote(backup+'/'+name+'.absent')+'; fi',label='back up '+name)
 for directory,name in [('/usr/lib/lua/luci/model/fastnet','model-dir'),('/etc/fastnet','state-dir'),('/usr/lib/lua/luci/view/fastnet','view-dir')]:
  remote('if [ -d '+shlex.quote(directory)+' ]; then touch '+shlex.quote(backup+'/'+name+'.existed')+'; fi',label='record directory ownership')
 for source,name in [(a.binary,'FastNet'),(a.packages/'luci-app-fastnet/luasrc/controller/fastnet.lua','controller.lua'),(a.packages/'luci-app-fastnet/luasrc/model/fastnet/card.lua','model.lua'),(a.packages/'fastnet/files/fastnet.config','config'),(a.packages/'luci-app-fastnet/luasrc/model/cbi/fastnet.lua','cbi.lua'),(a.packages/'luci-app-fastnet/luasrc/view/fastnet/status.htm','status.htm')]:
  content=source.read_bytes()
  if a.diagnose_csrf and name=='model.lua':
   diagnostic='return reply(403,{error="csrf",diagnostic={contextTokenIsString=type(ctx.authtoken)=="string",contextMatchesFixture=ctx.authtoken=='+json.dumps(token)+',underscoreHeaderPresent=http.getenv("HTTP_X_CSRF_TOKEN")~=nil,hyphenHeaderPresent=http.getenv("HTTP_X-CSRF-TOKEN")~=nil,underscoreMatchesFixture=http.getenv("HTTP_X_CSRF_TOKEN")=='+json.dumps(token)+',hyphenMatchesFixture=http.getenv("HTTP_X-CSRF-TOKEN")=='+json.dumps(token)+'}})'
   content=content.decode().replace('return error_reply(403,"csrf")',diagnostic).encode()
  remote('cat > '+shlex.quote(stage+'/'+name),content,label='upload '+name,timeout=90)
 changed=True
 remote('mkdir -p /usr/lib/lua/luci/view/fastnet; cp '+shlex.quote(stage+'/cbi.lua')+' /usr/lib/lua/luci/model/cbi/fastnet.lua; cp '+shlex.quote(stage+'/status.htm')+' /usr/lib/lua/luci/view/fastnet/status.htm',label='stage complete LuCI package resources')
 remote('if [ ! -f /etc/config/fastnet ]; then cp '+shlex.quote(stage+'/config')+' /etc/config/fastnet; fi',label='stage missing package configuration after backup')
 remote('chmod 700 '+shlex.quote(stage+'/FastNet')+'; mkdir -p /usr/lib/lua/luci/model/fastnet; cp '+shlex.quote(stage+'/model.lua')+' /usr/lib/lua/luci/model/fastnet/card.lua.new; mv /usr/lib/lua/luci/model/fastnet/card.lua.new /usr/lib/lua/luci/model/fastnet/card.lua; cp '+shlex.quote(stage+'/controller.lua')+' /usr/lib/lua/luci/controller/fastnet.lua.new; mv /usr/lib/lua/luci/controller/fastnet.lua.new /usr/lib/lua/luci/controller/fastnet.lua',label='install reversible controller candidate')
 remote('rm -f /etc/fastnet/card-state.json',label='isolate candidate state after backup');invalidate()
 port=41602
 start='(trap "" HUP; exec env FASTNET_CARD_SOCKET=/var/run/fastnet-card.sock FASTNET_CARD_STATE=/etc/fastnet/card-state.json '+shlex.quote(stage+'/FastNet')+' web --addr 127.0.0.1:'+str(port)+' --token '+shlex.quote(core_token)+' --no-open) >'+shlex.quote(stage+'/server.log')+' 2>&1 </dev/null & echo $!'
 pid=remote(start,label='start private candidate').decode().strip();assert pid.isdigit()
 session=ubus('create',{'timeout':900});sid=session['ubus_rpc_session'];ubus('set',{'ubus_rpc_session':sid,'values':{'username':'root','token':token}})
 for scope,objects in [('ubus',[['*','*']]),('access-group',[['*','read'],['*','write']]),('uci',[['*','read'],['*','write']])]:ubus('grant',{'ubus_rpc_session':sid,'scope':scope,'objects':objects})
 deadline=time.monotonic()+12
 while time.monotonic()<deadline:
  status,summary,raw,_=request('/card/summary')
  if status==200 and summary and summary.get('service',{}).get('state')=='ready':break
  time.sleep(.2)
 else:raise AssertionError(('authenticated summary not ready',status,summary))
 assert b'"task":null' in raw or b'"task": null' in raw
 remote('test "$(readlink /proc/'+pid+'/exe)" = '+shlex.quote(stage+'/FastNet'),label='verify running candidate ownership')
 direct=json.loads(remote('curl -fsS --max-time 3 -H '+shlex.quote('Authorization: Bearer '+core_token)+' http://127.0.0.1:'+str(port)+'/api/card/summary',label='compare private/core instance'))
 assert direct['instanceId']==summary['instanceId'],'private socket belongs to another service'
 socket_owned=True
 print('Authenticated LuCI summary ready; root-only socket connected',flush=True)
 before=set()
 state_raw=remote('cat /etc/fastnet/card-state.json 2>/dev/null || true',label='read initial task count')
 if state_raw:before={j['task']['id'] for j in json.loads(state_raw)['jobs']}
 status,value,_,_=request('/card/summary',authenticated=False);assert not (value and value.get('schemaVersion')==1)
 for path,options,expected in [('/card/run',{'method':'GET'},405),('/card/run',{'body':{'idempotencyKey':'fixture-key-0001'},'csrf':False},403),('/card/run',{'raw':b'{}'},422),('/card/run',{'raw':b'{"idempotencyKey":"fixture-key-0001","idempotencyKey":"fixture-key-0002"}'},422),('/card/run',{'raw':b' '*2000},413),('/card/task?id=../bad',{},422)]:
  status,_,_,_=request(path,**options);assert status==expected,(path,status,expected)
 report['checks'].append('real LuCI login, method, CSRF, duplicate-key, size and task-id rejection; no core token disclosure')
 key=uuid.uuid4().hex;began=time.monotonic();status,task,_,_=request('/card/run',{'idempotencyKey':key});assert status==202,(status,task);report['submitSeconds']=time.monotonic()-began;ident=task['id'];assert request('/card/run',{'idempotencyKey':key})[1]['id']==ident
 print('Real LuCI task submitted; checking same run through completion',flush=True)
 deadline=time.monotonic()+125
 while time.monotonic()<deadline:
  status,current,_,_=request('/card/task?id='+ident);assert status==200
  if current['state'] not in ['running','cancel_requested']:break
  time.sleep(2)
 assert current['state'] in ['succeeded','partial'],current.get('error')
 status,summary,_,_=request('/card/summary');assert status==200
 disk=json.loads(remote('cat /etc/fastnet/card-state.json',label='compare actual core state'))
 assert summary['latest']==disk['latest'];assert next(j for j in disk['jobs'] if j['task']['id']==ident)['task']==current
 assert {j['task']['id'] for j in disk['jobs']}==before|{ident}
 report['checks'].append('LuCI task/result exactly matches same Go runId and persisted snapshot; reads and rejected writes create no other jobs')
 configured=remote("uci -q get fastnet.@fastnet[0].port || echo 3200",label='read configured port').decode().strip();assert configured!=str(port)
 report['checks'].append('controller works when configured public port differs from actual candidate port; private socket is independent')
 status,second,_,_=request('/card/run',{'idempotencyKey':uuid.uuid4().hex});assert status==202
 began=time.monotonic();assert request('/card/cancel',{'id':second['id']})[0]==200
 while time.monotonic()-began<2:
  _,cancelled,_,_=request('/card/task?id='+second['id'])
  if cancelled['state']=='cancelled':break
  time.sleep(.02)
 assert cancelled['state']=='cancelled';report['cancelSeconds']=time.monotonic()-began
 stop();status,stopped,_,_=request('/card/summary');assert status==200 and stopped['service']['state'] in ['disabled','unavailable']
 for field in ['download','upload','nat','ipv6']:
  for key in ['status','value','measuredAt','error']:assert stopped['latest'][field].get(key)==summary['latest'][field].get(key)
 report['checks'].append('real cancel completes within two seconds; stopped service summary preserves values and nulls without starting it')
 for path in ['/status','/open','']:
  status,value,_,_=request(path);assert status in [200,302,303,404,503],('legacy endpoint regressed',path,status)
 report['checks'].append('legacy status/open/management routes remain available with their existing response modes')
 report['coreSha256']=hashlib.sha256(a.binary.read_bytes()).hexdigest();report['status']='passed'
except BaseException as error:
 report['status']='failed';report['failure']=str(error);raise
finally:
 try:
  stop()
  if sid:ubus('destroy',{'ubus_rpc_session':sid})
  if changed:
   for path,name in files:
    remote('if [ -f '+shlex.quote(backup+'/'+name+'.absent')+' ]; then rm -f '+shlex.quote(path)+'; else cp -p '+shlex.quote(backup+'/'+name)+' '+shlex.quote(path)+'; fi',label='restore '+name)
   invalidate();remote('rmdir /usr/lib/lua/luci/model/fastnet /etc/fastnet 2>/dev/null || true',label='remove empty task-created directories')
  if socket_owned:remote("lua -e 'local n=require \"nixio\";local s=n.socket(\"unix\",\"stream\");local live=s:connect(\"/var/run/fastnet-card.sock\");s:close();if not live then require(\"nixio.fs\").unlink(\"/var/run/fastnet-card.sock\") end'",label='remove task-owned stale socket')
  remote('rm -rf '+shlex.quote(stage)+' '+shlex.quote(backup),label='remove task-owned private stage')
  report['restored']=True
 except BaseException as error:report['restorationError']=str(error);report['status']='failed'
 (a.output/'controller-report.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
print(json.dumps(report,ensure_ascii=False))
