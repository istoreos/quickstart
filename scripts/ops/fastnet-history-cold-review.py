#!/usr/bin/env python3
"""Verify stopped-service snapshots and cold reload on the owned .7 appliance."""
import argparse,hashlib,json,math,secrets,shlex,subprocess,time,urllib.request
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--output',type=Path,required=True);a=p.parse_args();a.output.parent.mkdir(parents=True,exist_ok=True)
def remote(command):return subprocess.check_output(['ssh','-o','BatchMode=yes','-o','ConnectTimeout=5','root@192.168.30.7',command])
def ubus(action,value):return json.loads(remote('ubus -S call session '+action+' '+shlex.quote(json.dumps(value))) or b'{}')
def core(path):return json.loads(remote('curl --fail -s --max-time 5 http://127.0.0.1:3200/api/'+path))
def same_values(expected,actual):
 for key,value in expected.items():
  observed=actual[key];assert observed['runId']==value['runId'];assert observed['status']==value['status']
  if isinstance(value.get('value'),(int,float)):assert math.isclose(observed['value'],value['value'],rel_tol=1e-12,abs_tol=1e-5)
  else:assert observed.get('value')==value.get('value')
sid=None;stopped=False;report={'status':'running','scope':'one FastNet stop/start only; no OS reboot or configuration/package changes'}
try:
 before=core('history/summary');assert not before['activeRuns'];assert json.loads(remote('curl -fsS --unix-socket /var/run/fastnet-card.sock http://localhost/api/card/summary'))['task'] is None
 state=remote('cat /etc/fastnet/card-state.json');report['stateBytes']=len(state);report['sha256Before']=hashlib.sha256(state).hexdigest();assert len(state)<=128*1024
 sid=ubus('create',{'timeout':120})['ubus_rpc_session'];ubus('set',{'ubus_rpc_session':sid,'values':{'username':'root','token':secrets.token_hex(16)}})
 for scope,objects in [('ubus',[['*','*']]),('access-group',[['*','read'],['*','write']]),('uci',[['*','read'],['*','write']])]:ubus('grant',{'ubus_rpc_session':sid,'scope':scope,'objects':objects})
 remote('/etc/init.d/fastnet stop');stopped=True
 request=urllib.request.Request('http://192.168.30.7:10000/cgi-bin/luci/admin/services/fastnet/card/summary',headers={'Cookie':'sysauth_http='+sid})
 with urllib.request.urlopen(request,timeout=10) as response:offline=json.load(response)
 assert offline['service']['state']=='unavailable';same_values(before['latestResults'],offline['latestResults']);report['stoppedSnapshot']='passed'
 remote('/etc/init.d/fastnet start');stopped=False
 for attempt in range(25):
  try:after=core('history/summary');break
  except subprocess.CalledProcessError:time.sleep(.2)
 else:raise AssertionError('FastNet did not become ready')
 same_values(before['latestResults'],after['latestResults']);assert after['historyCount']==before['historyCount'];assert not after['activeRuns']
 report['sha256After']=hashlib.sha256(remote('cat /etc/fastnet/card-state.json')).hexdigest();assert report['sha256After']==report['sha256Before'];assert remote('stat -c %a /etc/fastnet/card-state.json').strip()==b'600'
 report.update(status='passed',historyCount=after['historyCount'],coldSnapshot='passed',version=core('version')['version'],journalMode='0600')
finally:
 if stopped:remote('/etc/init.d/fastnet start')
 if sid:ubus('destroy',{'ubus_rpc_session':sid})
 a.output.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
print(json.dumps(report))
