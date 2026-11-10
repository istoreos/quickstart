// Real controller measurements and production CardShell/component lifecycle fixture.
const {chromium}=require('playwright'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path'),{spawn}=require('node:child_process'),crypto=require('node:crypto')
const out=process.env.OUTPUT_DIR,base=process.env.DEVICE_URL,sid=process.env.DEVICE_SESSION,token=process.env.DEVICE_CSRF,host=process.env.DEVICE_HOST
if(!out||!base||!sid||!token||!host)throw Error('Private device fixture required')
const mean=a=>a.reduce((s,v)=>s+v,0)/a.length,p95=a=>[...a].sort((a,b)=>a-b)[Math.ceil(a.length*.95)-1],sleep=ms=>new Promise(r=>setTimeout(r,ms))
const sampleLua=`local fs=require'nixio.fs';local n=require'nixio';local j=require'luci.jsonc'
local function snapshot()
 local c={};for v in (fs.readfile('/proc/stat'):match('cpu%s+([^\\n]+)')):gmatch('%d+') do c[#c+1]=tonumber(v) end
 local processes={};local count=0
 for pid in fs.dir('/proc') do if pid:match('^%d+$') then
  local comm=(fs.readfile('/proc/'..pid..'/comm') or ''):gsub('%s+$','')
  if comm=='FastNet' or comm=='quickstart' or comm=='uhttpd' then
   local rss=tonumber((fs.readfile('/proc/'..pid..'/status') or ''):match('VmRSS:%s+(%d+)')) or 0
   processes[#processes+1]={pid=pid,comm=comm,rssKiB=rss};count=count+1
  end
 end end
 return {cpu=c,processes=processes,count=count}
end
local before=snapshot();n.nanosleep(60);local after=snapshot();print(j.stringify({before=before,after=after}))`
async function sample(){
 const began=performance.now();const raw=await new Promise((resolve,reject)=>{const child=spawn('ssh',['-o','BatchMode=yes','-o','ConnectTimeout=5',host,'lua -']),chunks=[];let error='';child.stdout.on('data',b=>chunks.push(b));child.stderr.on('data',b=>error+=b);child.on('error',reject);child.on('close',code=>code?reject(Error(error)):resolve(Buffer.concat(chunks).toString()));child.stdin.end(sampleLua)})
 const elapsedMs=performance.now()-began;assert.ok(elapsedMs>=59900,'resource interval shorter than 60 seconds');const v=JSON.parse(raw),total=x=>x.slice(0,8).reduce((a,b)=>a+b,0),delta=total(v.after.cpu)-total(v.before.cpu),idle=(v.after.cpu[3]+v.after.cpu[4])-(v.before.cpu[3]+v.before.cpu[4])
 return {...v,elapsedMs,cpuPercent:100*(delta-idle)/delta,rssKiB:mean([v.before.processes.reduce((s,p)=>s+p.rssKiB,0),v.after.processes.reduce((s,p)=>s+p.rssKiB,0)])}
}
;(async()=>{
 const b=await chromium.launch({args:['--js-flags=--expose-gc']}),report={milestone:'FN10',status:'in_progress',checks:[],requests:[],samples:[],budgets:{}}
 const c=await b.newContext({ignoreHTTPSErrors:true,viewport:{width:900,height:800}});await c.addCookies(['sysauth_http','sysauth_https'].map(name=>({name,value:sid,url:base})))
 const p=await c.newPage(),prefix='/cgi-bin/luci/admin/services/fastnet/card/'
 p.on('request',r=>{const u=new URL(r.url());if(u.pathname.includes('/fastnet/'))report.requests.push({path:u.pathname,time:Date.now(),method:r.method()})})
 try{
  await p.goto(base+'/cgi-bin/luci/admin/quickstart');await p.locator('[data-home-settings]').waitFor();assert.equal(report.requests.length,0)
  await p.locator('[data-home-settings]').click();await p.locator('[data-open-library]').click();await p.locator('[data-choice="plugin.fastnet"]').click();await p.locator('[data-confirm-card-install]').click();await p.locator('[data-confirm-choice]').waitFor({timeout:180000});assert.equal(report.requests.length,0);report.checks.push('uninstalled and installed-but-unadded request no FastNet JS/API')
  async function request(action,body){const start=performance.now(),r=body?await c.request.post(base+prefix+action+'?token='+encodeURIComponent(token),{data:body,timeout:8000}):await c.request.get(base+prefix+action,{timeout:8000});const raw=await r.body();assert.ok(r.ok(),action+' '+r.status());return {value:JSON.parse(raw.toString()),bytes:raw.length,ms:performance.now()-start}}
  // Explicit fixture setup: the arm test device retains a disabled user configuration.
  await new Promise((resolve,reject)=>{const child=spawn('ssh',['-o','BatchMode=yes',host,"uci -q set fastnet.@fastnet[0].enabled=1; uci commit fastnet; /etc/init.d/fastnet restart"]);child.on('error',reject);child.on('close',code=>code?reject(Error('core fixture enable failed')):resolve())});
  let ready=false;for(let i=0;i<50;i++){if((await request('summary')).value.service.state==='ready'){ready=true;break}await sleep(100)}assert.ok(ready,'explicitly enabled fixture core did not become ready');
  let maximumSummary=0,maximumTask=0;const gets=[],runs=[],stops=[]
  for(let i=0;i<35;i++){const r=await request('summary');maximumSummary=Math.max(maximumSummary,r.bytes);if(i>=5)gets.push(r.ms)}
  for(let i=0;i<35;i++){
   const r=await request('run',{idempotencyKey:crypto.randomBytes(16).toString('hex')});if(i>=5)runs.push(r.ms);maximumTask=Math.max(maximumTask,r.bytes)
   const began=performance.now();await request('cancel',{id:r.value.id});let current
   do{current=await request('task?id='+r.value.id);if(current.value.state==='cancelled')break;await sleep(20)}while(performance.now()-began<2000)
   assert.equal(current.value.state,'cancelled');stops.push(performance.now()-began);maximumTask=Math.max(maximumTask,current.bytes)
  }
  const registry=await c.request.get(base+'/cgi-bin/luci/istore/home/v1/cards/');const registryBytes=(await registry.body()).length
  report.budgets.B03={summaryP95ms:p95(gets),runP95ms:p95(runs),warmup:5,samples:30,summarySamplesMs:gets,runSamplesMs:runs};assert.ok(p95(gets)<=500);assert.ok(p95(runs)<=1000)
  report.budgets.B04={maximumSummaryBytes:maximumSummary,maximumTaskBytes:maximumTask,registryBytes};assert.ok(maximumSummary<=8192&&maximumTask<=4096&&registryBytes<=65536)
  report.budgets.B10={realStarts:35,maximumStopMs:Math.max(...stops)};assert.ok(Math.max(...stops)<=2000)
  await c.addInitScript(()=>{
   const timers=new Set(),observers=new Set(),listeners=new Map();let pending=0,queries=0
   const set=window.setTimeout,clear=window.clearTimeout
   window.setTimeout=function(fn,ms,...args){let id=set(()=>{timers.delete(id);typeof fn==='function'?fn(...args):window.eval(fn)},ms);timers.add(id);return id}
   window.clearTimeout=function(id){timers.delete(id);return clear(id)}
   for(const kind of ['IntersectionObserver','ResizeObserver','MutationObserver']){const original=window[kind];window[kind]=class extends original{constructor(...args){super(...args);observers.add(this)}disconnect(){observers.delete(this);return super.disconnect()}}}
   const add=EventTarget.prototype.addEventListener,remove=EventTarget.prototype.removeEventListener
   EventTarget.prototype.addEventListener=function(type,fn,...options){if(this===document&&type==='visibilitychange')listeners.set(fn,1);return add.call(this,type,fn,...options)}
   EventTarget.prototype.removeEventListener=function(type,fn,...options){if(this===document&&type==='visibilitychange')listeners.delete(fn);return remove.call(this,type,fn,...options)}
   const fetch=window.fetch;window.fetch=async(...args)=>{const url=String(args[0]);if(url.includes('/fastnet/card/')){pending++;queries++;try{return await fetch(...args)}finally{pending--}}return fetch(...args)}
   window.__fnMetrics=()=>({timers:timers.size,observers:observers.size,listeners:listeners.size,pending,queries})
  })
  await p.goto(base+'/luci-static/quickstart/bench/index.html');await p.locator('[data-toggle-card]').waitFor()
  const toggle=()=>p.locator('[data-toggle-card]').click(),mounted=()=>p.locator('[data-fastnet-card] [data-fastnet-test]:not(:disabled)').waitFor(),unmounted=()=>p.locator('[data-fastnet-card]').waitFor({state:'hidden'})
  await toggle();await mounted();await toggle();await unmounted();await sleep(100)
  const cd=await c.newCDPSession(p);await cd.send('HeapProfiler.enable');await cd.send('HeapProfiler.collectGarbage');const before=await cd.send('Runtime.getHeapUsage'),scope=await p.evaluate(()=>window.__fnMetrics())
  for(let i=0;i<50;i++){await toggle();await mounted();await toggle();await unmounted()}
  await sleep(100);await cd.send('HeapProfiler.collectGarbage');const after=await cd.send('Runtime.getHeapUsage'),end=await p.evaluate(()=>window.__fnMetrics())
  for(const k of ['timers','observers','listeners','pending'])assert.equal(end[k],scope[k],k+' leaked')
  assert.equal(await p.locator('.qs-card-shell').count(),0,'card DOM leaked');const growth=after.usedSize-before.usedSize;assert.ok(growth<=2*1024*1024,JSON.stringify({growth,before,after}));report.budgets.B08={mountCycles:50,scope,end,heapGrowthBytes:growth};report.checks.push('50 production shell/component mount cycles release timers/observers/listeners/requests; GC heap within 2 MiB')
  const modules=await p.evaluate(()=>performance.getEntriesByType('resource').filter(e=>/fastnet\/quickstart\/card\.[a-f0-9]+\.js/.test(e.name)).map(e=>({path:new URL(e.name).pathname,wire:e.transferSize,encoded:e.encodedBodySize})))
  assert.equal(modules.length,1);report.budgets.B07={successfulImports:1,modules}
  // Warm both the service and the already imported card before paired measurements.
  await sleep(30000)
  for(let pair=0;pair<3;pair++){
   const values={}
   for(const state of pair===1?['visible','hidden']:['hidden','visible']){
    const shown=await p.locator('[data-fastnet-card]').count()>0
    if(shown!==(state==='visible'))await toggle();if(state==='visible')await mounted();else await unmounted()
    const count=await p.evaluate(()=>window.__fnMetrics().queries),r=await sample();assert.equal(await p.evaluate(()=>window.__fnMetrics().queries),count,'idle card polled')
    values[state]=r;report.samples.push({pair:pair+1,state,...r});fs.writeFileSync(path.join(out,'performance-progress.json'),JSON.stringify({completed:report.samples.length,total:6,samples:report.samples},null,2)+'\n')
   }
   assert.equal(values.visible.before.count,values.hidden.before.count);assert.equal(values.visible.after.count,values.hidden.after.count)
  }
  const delta=key=>[0,1,2].map(i=>report.samples.find(v=>v.pair===i+1&&v.state==='visible')[key]-report.samples.find(v=>v.pair===i+1&&v.state==='hidden')[key])
  report.budgets.B05={idleSecondsPerVisibleInterval:60,intervals:3,periodicRequests:0};report.budgets.B09={warmupSeconds:30,pairs:3,secondsEach:60,cpuDeltasPercentagePoints:delta('cpuPercent'),meanCpuDelta:mean(delta('cpuPercent')),rssDeltasKiB:delta('rssKiB'),meanRSSDeltaKiB:mean(delta('rssKiB')),newPersistentProcesses:0,measurement:'whole-device /proc/stat CPU; RSS of same FastNet + Quickstart + uhttpd PIDs; isolated production CardShell fixture'}
  assert.ok(report.budgets.B09.meanCpuDelta<=1);assert.ok(report.budgets.B09.meanRSSDeltaKiB<=8192)
  report.status='passed'
 }catch(e){report.status='failed';report.failure=e.message.split(token).join('<redacted>').split(sid).join('<redacted>');throw e}
 finally{await b.close();fs.writeFileSync(path.join(out,'performance-browser-report.json'),JSON.stringify(report,null,2)+'\n')}
 console.log(JSON.stringify({status:report.status,budgets:report.budgets}))
})().catch(e=>{console.error(e.message);process.exitCode=1})
