#!/usr/bin/env python3
"""Owned netns capacity/soak gate. It never edits root network configuration."""
import argparse,datetime,hashlib,json,pathlib,shlex,signal,subprocess,time,uuid
p=argparse.ArgumentParser()
p.add_argument('host',choices=['192.168.30.7','192.168.30.93'])
p.add_argument('--application');p.add_argument('--binary',required=True);p.add_argument('--go-test',required=True)
p.add_argument('--legacy-test');p.add_argument('--counts',default='1,2,4');p.add_argument('--output',required=True)
p.add_argument('--exercise-rebind',action='store_true');p.add_argument('--soak-hours',type=float,default=24);p.add_argument('--preflight-only',action='store_true')
a=p.parse_args();counts=[int(x) for x in a.counts.split(',')]
if not counts or any(x not in (1,2,4) for x in counts):p.error('counts must be 1,2,4')
if not a.preflight_only and not a.application:p.error('production application binary required')
if not a.preflight_only and not a.legacy_test:p.error('pinned old baseline binary required')
if not a.preflight_only and a.soak_hours<24:p.error('actual soak must be at least 24 hours')
out=pathlib.Path(a.output);out.parent.mkdir(parents=True,exist_ok=True)
target='root@'+a.host
state={'milestone':'M10','host':a.host,'status':'starting','startedAt':datetime.datetime.now(datetime.timezone.utc).isoformat(),'preflightOnly':a.preflight_only,'cases':[],'artifactSha256':{name:hashlib.sha256(pathlib.Path(file).read_bytes()).hexdigest() for name,file in [('engine',a.binary),('goTest',a.go_test),('application',a.application),('legacyTest',a.legacy_test)] if file}}
def save():out.write_text(json.dumps(state,indent=2)+'\n')
def remote(command,check=True,timeout=60):
 result=subprocess.run(['ssh','-o','BatchMode=yes',target,shlex.join(command) if isinstance(command,list) else command],capture_output=True,text=True,timeout=timeout)
 if check and result.returncode:raise RuntimeError(f'{command}: {result.stdout[-3000:]} {result.stderr[-3000:]}')
 return result
def upload(source,destination):subprocess.run(['scp','-O','-q',str(source),target+':'+destination],check=True,timeout=60)
def measure(case,latency=False):
 command=['ip','netns','exec',case['router'],'env','QUICKSTART_MULTI_LAN_CAPACITY_BASE='+case['base']]
 if latency:command+=['QUICKSTART_MULTI_LAN_CAPACITY_LATENCY=1']
 result=remote(command+[case['base']+'/service.test','-test.run','^TestMultiLANCapacityMeasure$','-test.timeout','30m','-test.v'],timeout=1830)
 row=next((line[9:] for line in result.stdout.splitlines() if line.startswith('CAPACITY=')),None)
 if row is None:raise RuntimeError(result.stdout)
 value=json.loads(row);value['wallMonotonic']=time.monotonic();value['lanCount']=case['lanCount'];value['loadProbe']=latency
 with out.with_suffix('.samples.jsonl').open('a') as stream:stream.write(json.dumps(value)+'\n')
 return value

def legacy_latency(case):
 base=case['base'];ns=case['router']
 command='i=0; while [ "$i" -lt 100 ]; do curl -fsS --max-time 10 -o '+shlex.quote(base+'/baseline-response.json')+' -w "%{time_total}\\n" http://127.0.0.1:19482/cgi-bin/luci/istore/lanctrl/v2/devices/; i=$((i+1)); done'
 durations=sorted(float(row)*1000 for row in remote(['ip','netns','exec',ns,'sh','-c',command],timeout=180).stdout.splitlines())
 assert len(durations)==100,('legacy latency samples',len(durations))
 return {'commit':'66589b22f140615db97851b6c3b1eec7aad975d5','samples':100,'p95Ms':durations[94]}

def validate(value,case,strict=False):
 for name,process in value['processes'].items():
  limit=128*1024 if name=='backend.pid' else 64*1024
  assert 0<process['rssKiB']<=limit,(name,'RSS budget',process)
  if name!='backend.pid':
   assert process['bpfMaps']>0 and 0<process['bpfBytes']<=32*1024*1024,('BPF budget or unavailable evidence',process)
 for api in value['api']:
  assert api['devices']==256,('device admission count',api)
  if api['endpoint']=='device-traffic':assert api['activeDevices']==256,('device packet counters',api)
  assert api['state']=='ready',('sample or inventory health',api)
  assert api['maxSampleAgeSeconds']<=30,('sample freshness',api)
  if strict:assert api['samples']>=100 and api['p95Ms']<=1500,('latency',api)
 for file,count in value['metrics']['writesLastHour'].items():assert count<=60,('history writes in last hour',file,count)
 previous=case.get('last')
 if previous:
  elapsed=value['cpuSampleMonotonicSeconds']-previous['cpuSampleMonotonicSeconds']
  ticks=sum(value['processes'][key]['ticks']-old['ticks'] for key,old in previous['processes'].items())
  value['observedCpuPercent']=ticks/elapsed # Linux USER_HZ is 100 on both targets.
  if strict or previous.get('loadProbe'):
   value['cpuInterval']='active_api_load'
  elif elapsed>=30:
   value['cpuInterval']='steady_sampling';value['combinedCpuPercent']=value['observedCpuPercent']
   assert value['combinedCpuPercent']<=25,('combined CPU budget',value['combinedCpuPercent'])
 if previous and 'warm' in case and value['wallMonotonic']-previous['wallMonotonic']>=30:
  current={item['networkId']:item for item in value['api'] if item['endpoint']=='device-traffic'}
  old={item['networkId']:item for item in previous['api'] if item['endpoint']=='device-traffic'}
  normalized=[]
  for index in range(1,case['lanCount']+1):
   network='capacity'+str(index);delta=current[network]['uploadBytes']-old[network]['uploadBytes']
   assert delta>0,('counter stopped or reset',network,delta)
   normalized.append(delta/(28+96*index))
  assert max(normalized)<=min(normalized)*1.25,('counter attribution payload ratio',normalized)
 if 'warm' in case:
  for key,process in value['processes'].items():assert process['rssKiB']-case['warm']['processes'][key]['rssKiB']<=32*1024,('steady RSS growth',key,process)
 case['last']=value

def packet_guard(case):
 results=[]
 for index in range(1,case['lanCount']+1):
  for family,address in [(4,'203.0.113.2'),(6,'fd90::2')]:
   minute,second=map(int,remote(['date','+%M:%S']).stdout.strip().split(':'))
   # Avoid both the scheduler's settling interval and a probe crossing into
   # the next minute. Always check the other LANs, including during transitions.
   if index==1 and (minute in (10,11) and second<15 or minute in (9,10) and second>=55):continue
   expected=index!=1 or minute!=10
   namespace=case['router']+'-c'+str(index)
   reply=remote(['ip','netns','exec',namespace,'ping','-'+str(family),'-c','1','-W','1',address],False)
   end_minute,end_second=map(int,remote(['date','+%M:%S']).stdout.strip().split(':'))
   if index==1 and (end_minute!=minute or end_minute in (10,11) and end_second<15):continue
   assert (reply.returncode==0)==expected,('unexpected scheduled packet result',index,family,expected,minute,second,end_minute,end_second,reply.stdout,reply.stderr)
   results.append({'networkId':'capacity'+str(index),'family':family,'expectedAllowed':expected,'actualAllowed':reply.returncode==0,'startMinute':minute,'startSecond':second,'endMinute':end_minute,'endSecond':end_second})
 case['lastPacketGuard']=results
 case['packetGuardChecks']=case.get('packetGuardChecks',0)+len(results)

def exercise_rebind(case):
 if case['lanCount']<2:return
 base=case['base'];router=case['router']
 def inside(args,check=True):return remote(['ip','netns','exec',router]+args,check)
 def api(path,body=None,port=19481):
  command=['curl','-sS','--max-time','10','-H','X-Forwarded-Sid: 11111111111111111111111111111111','-H','X-Forwarded-Token: m10-isolated']
  if body is not None:command+=['-H','Content-Type: application/json','--data-binary',json.dumps(body)]
  command+=['http://127.0.0.1:'+str(port)+path]
  response=inside(command,False)
  if response.returncode==7 and port==19501:
   log=remote(['cat',base+'/engine-1.log']).stdout
   assert 'interface identity changed; stopping this instance' in log,('unexpected engine exit',log[-1000:])
   return {'status':'unavailable','data':{'state':'unavailable'}}
  assert response.returncode==0,(response.returncode,response.stderr)
  return json.loads(response.stdout)
 scopes=json.loads(remote(['cat',base+'/scopes.json']).stdout);scope=scopes[0];iface=scope['device'];prefix=iface[:-1]
 native=lambda path,body=None,index=1:api('/api/v1/'+path,body,19500+index)
 other=native('system/health',index=2)['data'];old_index=native('system/health')['data']['context']['interfaceIndex']
 owner={'product':'quickstart','instance':'m10-rebind'}
 request={'expectedRevision':0,'idempotencyKey':'rebind-seed','owner':owner,'policies':[{'device':{'kind':'mac','value':'02:00:00:00:00:01'},'rateLimit':{'enabled':True,'uploadBitsPerSecond':3000000,'downloadBitsPerSecond':6000000}}]}
 plan=native('policies/plan',request)['data'];assert native('policies/apply',{'request':request,'plan':plan})['status']=='success'
 stale=dict(request,expectedRevision=1,idempotencyKey='rebind-stale');old_plan=native('policies/plan',stale)['data']
 groups=api('/cgi-bin/luci/istore/lanctrl/v2/device-groups/?network='+scope['networkId'])['result']
 version=groups['version'];mac=inside(['cat','/sys/class/net/'+iface+'/address']).stdout.strip()
 runtime_file=base+'/rootfs/opt/runtime.json';runtime=json.loads(remote(['cat',runtime_file]).stdout)
 def up(value):
  runtime['interface'][0]['up']=value
  data=json.dumps(runtime)
  remote(['sh','-c','printf %s '+shlex.quote(data)+' > '+shlex.quote(runtime_file)])
 before=remote(['sha256sum',base+'/rootfs/etc/config/firewall']).stdout
 remote(['ip','-n',router,'link','set',iface,'down']);up(False)
 rejected=api('/cgi-bin/luci/istore/lanctrl/v2/device-policy/?network='+scope['networkId'],{'deviceId':'mac:02:00:00:00:00:01','kind':'access','access':{'networkAccess':False}})
 assert rejected.get('error') or rejected.get('result',{}).get('error'),('down link accepted write',rejected)
 assert remote(['sha256sum',base+'/rootfs/etc/config/firewall']).stdout==before
 remote(['ip','-n',router,'link','delete',iface])
 assert native('policies/apply',{'request':stale,'plan':old_plan})['status']!='success','stale native plan survived device deletion'
 remote(['ip','-n',router,'link','add',iface,'type','bridge']);remote(['ip','-n',router,'link','set',iface,'address',mac])
 remote(['ip','-n',router,'addr','add',scope['ipv4Prefixes'][0],'dev',iface]);remote(['ip','-n',router,'-6','addr','add',scope['ipv6Prefixes'][0],'dev',iface,'nodad'])
 remote(['ip','-n',router,'link','set',prefix+'p1','master',iface]);remote(['ip','-n',router,'link','set',iface,'up'])
 # Static test neighbors are recreated with the new kernel link identity.
 command='j=1; while [ "$j" -le 256 ]; do mac=$(printf "02:00:00:00:%02x:%02x" "$((j/256))" "$((j%256))"); ip neigh replace "10.201.$((j/250)).$((j%250+2))" lladdr "$mac" nud permanent dev '+shlex.quote(iface)+'; ip -6 neigh replace "fd91::$(printf %x "$((j+2))")" lladdr "$mac" nud permanent dev '+shlex.quote(iface)+'; j=$((j+1)); done'
 inside(['sh','-c',command]);up(True)
 stale_group=api('/cgi-bin/luci/istore/lanctrl/v2/device-groups/?network='+scope['networkId'],{'action':'set_device_policy','deviceId':'mac:02:00:00:00:00:01','devicePolicy':None,'expectedVersion':version})
 assert stale_group.get('result',{}).get('error',{}).get('code')=='conflict',('stale group version survived rebind',stale_group)
 assert native('system/health')['data']['state']!='healthy','old engine still reported healthy after link rebind'
 pid=int(remote(['cat',base+'/engine-1.pid']).stdout);remote(['kill','-TERM',str(pid)],False)
 deadline=time.monotonic()+10
 while remote(['kill','-0',str(pid)],False).returncode==0:
  if time.monotonic()>deadline:raise AssertionError('selected engine did not exit')
  time.sleep(.2)
 args=[base+'/quickstart-netpolicy','--iface',iface,'--port','19501','--data-dir',base+'/engine-1','--network-id','capacity1','--instance-id','capacity1','--enable-traffic','--traffic-additional-subnets','10.201.0.0/23,10.202.0.0/23,10.203.0.0/23,10.204.0.0/23,fd91::/64,fd92::/64,fd93::/64,fd94::/64']
 command=shlex.join(['ip','netns','exec',router]+args)+' > '+shlex.quote(base+'/engine-1-rebound.log')+' 2>&1 & echo $! > '+shlex.quote(base+'/engine-1.pid')
 remote(['sh','-c',command]);deadline=time.monotonic()+30
 while True:
  try:
   current=native('system/health')['data']
   if current['state']=='healthy':break
  except Exception:pass
  if time.monotonic()>deadline:raise AssertionError('rebound engine not ready')
  time.sleep(.3)
 assert current['context']['interfaceIndex']!=old_index
 stored=native('policies')['data'];assert stored['revision']==1 and len(stored['policies'])==1
 unchanged=native('system/health',index=2)['data'];assert unchanged['pid']==other['pid'] and unchanged['context']==other['context']
 case['hotplug']={'passed':True,'oldInterfaceIndex':old_index,'newInterfaceIndex':current['context']['interfaceIndex'],'otherInstanceUnchanged':True,'oldPlanRejected':True,'oldGroupVersionRejected':True,'policiesRestored':True}

def cleanup(case):
 for namespace in case['namespaces']:
  # Kill only PIDs proven to belong to one of this run's disposable namespaces.
  result=remote(['ip','netns','pids',namespace],False)
  pids=[part for part in result.stdout.split() if part.isdigit()]
  if pids:remote(['kill','-TERM']+pids,False)
 time.sleep(1)
 for namespace in case['namespaces']:
  result=remote(['ip','netns','pids',namespace],False);pids=[part for part in result.stdout.split() if part.isdigit()]
  if pids:remote(['kill','-KILL']+pids,False)
  remote(['ip','netns','delete',namespace],False)
 case['cleaned']=True

def stop_run(signum,frame):raise KeyboardInterrupt('runner stopped')
signal.signal(signal.SIGINT,stop_run)
signal.signal(signal.SIGTERM,stop_run)
fingerprint=remote('sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall').stdout
state['rootConfigurationBefore']=fingerprint
try:
 save()
 space=remote(['df','-k','/tmp']).stdout
 available=int(space.splitlines()[-1].split()[3])
 state['testEnvironment']={'tmpAvailableBeforeKiB':available,'minimumFixtureHeadroomKiB':1024*1024}
 assert available>=1024*1024,('insufficient space for isolated test artifacts; archive inactive owned fixtures first',space)
 for count in counts:
  identity=uuid.uuid4().hex[:6];base='/tmp/quickstart-multi-lan/m10-'+identity;router='qsm10-'+identity;wan=router+'-w';prefix='qc'+identity
  case={'lanCount':count,'base':base,'router':router,'namespaces':[router,wan]+[router+'-c'+str(i) for i in range(1,count+1)]};state['cases'].append(case);save()
  remote(['mkdir','-p',base]);upload(a.binary,base+'/quickstart-netpolicy');upload(a.go_test,base+'/service.test');
  if a.application:upload(a.application,base+'/application')
  upload(pathlib.Path(__file__).with_name('multi-lan-capacity-namespace.sh'),base+'/setup.sh');remote(['chmod','700',base+'/quickstart-netpolicy',base+'/service.test'])
  remote(['sh',base+'/setup.sh',base,router,wan,prefix,str(count),'1' if a.application else '0'],timeout=180)
  if count==1 and a.legacy_test:
   upload(a.legacy_test,base+'/legacy.test');remote(['chmod','700',base+'/legacy.test'])
   command=shlex.join(['ip','netns','exec',router,'env','QUICKSTART_MULTI_LAN_CAPACITY_BASE='+base,base+'/legacy.test','-test.run','^TestMultiLANLegacyBaselineHTTP$','-test.timeout','26h'])
   remote(['sh','-c',command+' > '+shlex.quote(base+'/baseline.log')+' 2>&1 &'])
  print(json.dumps({'started':count,'base':base}),flush=True)
  deadline=time.monotonic()+60
  while remote(['test','-f',base+'/backend.ready'],False).returncode:
   if time.monotonic()>deadline:raise RuntimeError(remote(['cat',base+'/backend.log'],False).stdout)
   time.sleep(1)
  # Kernel sampling and userspace neighbor admission are asynchronous.
  deadline=time.monotonic()+60
  while True:
   value=measure(case)
   try:validate(value,case);break
   except AssertionError:
    if time.monotonic()>deadline:raise
    time.sleep(3)
  if a.exercise_rebind:
   exercise_rebind(case)
   case.pop('last',None)
   deadline=time.monotonic()+60
   while True:
    value=measure(case)
    try:validate(value,case);break
    except AssertionError:
     if time.monotonic()>deadline:raise
     time.sleep(3)
  packet_guard(case);case['first']=value;save()
 state['status']='warming_up' if not a.preflight_only else 'preflight';save()
 warmup=0 if a.preflight_only else 30*60
 start=time.monotonic()
 while time.monotonic()-start<warmup:
  time.sleep(min(60,warmup-(time.monotonic()-start)))
  for case in state['cases']:
   validate(measure(case),case);packet_guard(case)
  state['warmupElapsedSeconds']=time.monotonic()-start;save()
 for case in state['cases']:
  value=measure(case,True);validate(value,case,True);case['warm']=value
  if case['lanCount']==1 and a.legacy_test:
   baseline=legacy_latency(case);case['legacyBaseline']=baseline
   current=next(item for item in value['api'] if item['endpoint']=='devices')['p95Ms']
   assert current<=baseline['p95Ms']+max(50,baseline['p95Ms']*.2),('single LAN regression',baseline,current)
  save()
 if not a.preflight_only:
  # All cohorts completed the same warmup concurrently; keep the largest for soak.
  largest=max(state['cases'],key=lambda case:case['lanCount'])
  for case in state['cases']:
   if case is not largest:cleanup(case)
  state['status']='soaking';state['soakStartedAt']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
  start=time.monotonic()
  while time.monotonic()-start<a.soak_hours*3600:
   time.sleep(min(60,a.soak_hours*3600-(time.monotonic()-start)))
   value=measure(largest);validate(value,largest);packet_guard(largest)
   elapsed=time.monotonic()-start
   state['soakElapsedSeconds']=elapsed;save()
  state['status']='passed'
 else:state['status']='preflight_passed'
except BaseException as error:
 state['status']='failed';state['error']=repr(error);raise
finally:
 for case in state['cases']:
  if not case.get('cleaned'):
   try:cleanup(case)
   except Exception as error:case['cleanupError']=repr(error)
 final=remote('sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall',False).stdout
 state['rootConfigurationAfter']=final
 state['rootConfigurationUnchanged']=final==fingerprint
 if final!=fingerprint:state['status']='failed';state['error']='root configuration fingerprint changed'
 state['finishedAt']=datetime.datetime.now(datetime.timezone.utc).isoformat();save()
 print(json.dumps({'status':state['status'],'output':str(out)}),flush=True)
