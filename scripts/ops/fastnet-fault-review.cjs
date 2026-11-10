const {chromium}=require('playwright'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path')
const out=process.env.OUTPUT_DIR,base=process.env.PREVIEW_URL||'http://127.0.0.1:5180'
if(!out)throw Error('OUTPUT_DIR required');fs.mkdirSync(out,{recursive:true})
;(async()=>{
 const b=await chromium.launch(),report={milestone:'FN10',status:'in_progress',checks:[]}
 async function fixture(){const c=await b.newContext(),p=await c.newPage();await p.goto(base+'/design/external-card?hidden=1&external=present&tools=0');await p.locator('[data-toggle-external]').waitFor();await p.evaluate(()=>{
  const original=window.fetch;window.__fnFault='';window.__fnCalls=[];window.__fnAborts=0
  window.fetch=async(input,options={})=>{
   const pathname=new URL(String(input),location.href).pathname,action=pathname.split('/').at(-1);if(!pathname.includes('/fastnet/card/'))return original(input,options)
   window.__fnCalls.push({action,time:performance.now(),body:options.body?JSON.parse(options.body):null})
   const fault=window.__fnFault
   if(action==='summary'&&fault==='slow')return new Promise((resolve,reject)=>{options.signal.addEventListener('abort',()=>{window.__fnAborts++;reject(new DOMException('Aborted','AbortError'))},{once:true})})
   if(action==='summary'&&fault==='redirect'){const r=new Response('');Object.defineProperty(r,'type',{value:'opaqueredirect'});return r}
   if(action==='summary'&&fault==='persist'){const s=await(await original(input,options)).json();s.service.persistenceError='write_failed';s.lastTask={id:'persist-run',state:'partial',issues:{ipv6:'probe_failed'}};return Response.json(s)}
   if(action==='summary'&&['403','404','503'].includes(fault))return new Response('{}',{status:Number(fault),headers:{'Content-Type':'application/json'}})
   if(action==='summary'&&fault==='json')return new Response('{broken',{headers:{'Content-Type':'application/json'}})
   if(action==='summary'&&fault==='bandwidth'){const s=await(await original(input,options)).json();s.latest.bandwidth={downloadBps:'oops',uploadBps:1,measuredAt:new Date().toISOString()};return Response.json(s)}
   if(action==='task'&&fault==='poll')return new Response('{}',{status:503,headers:{'Content-Type':'application/json'}})
   if(action==='summary'&&fault==='restart'){const s=await(await original(input,options)).json();s.instanceId='restarted-service';s.lastTask={...s.task,state:'interrupted',finishedAt:new Date().toISOString()};s.task=null;return Response.json(s)}
   if(action==='run'&&fault==='lost-503'){window.__fnFault='';await original(input,options);return Response.json({error:'upstream'},{status:503})}
   return original(input,options)
  }
 });await p.locator('[data-toggle-external]').click();await p.locator('[data-fastnet-card] [data-fastnet-test]:not(:disabled)').waitFor();return {c,p}}
 const reload=async p=>{await p.locator('.qs-card-menu>summary').click();await p.locator('.qs-card-menu button').nth(1).click()}
 const run=async(id,fn)=>{try{await fn();report.checks.push({id,status:'passed'})}catch(e){report.checks.push({id,status:'failed',failure:e.message})}}
 for(const fault of ['403','404','503','json','slow','bandwidth','redirect'])await run('read-'+fault,async()=>{
  const {c,p}=await fixture();try{const stamp=await p.locator('[data-fastnet-card] time').getAttribute('datetime');await p.evaluate(v=>window.__fnFault=v,fault);const start=Date.now();await reload(p);await p.locator('[data-fastnet-card] [role=alert]').waitFor();if(fault==='slow')assert.ok(Date.now()-start>=7900&&Date.now()-start<=9000);assert.equal(await p.locator('[data-fastnet-card] time').getAttribute('datetime'),stamp);assert.match(await p.locator('[data-fastnet-card] .qs-card-metrics').innerText(),/950/);await p.evaluate(()=>window.__fnFault='');await p.locator('[data-fastnet-card] [role=alert] button').click();await p.waitForTimeout(100);assert.equal(await p.evaluate(()=>window.__fnCalls.filter(c=>c.action==='run').length),0,'read retry unexpectedly starts a speed test')}finally{await c.close()}
 })
 await run('persistence-error-with-partial-result',async()=>{const{c,p}=await fixture();try{await p.evaluate(()=>window.__fnFault='persist');await reload(p);await p.getByText(/结果保存失败/).waitFor();await p.getByText(/部分检测未完成/).waitFor();assert.equal(await p.evaluate(()=>window.__quickstartCardFixture.stats.runs),0)}finally{await c.close()}})
 await run('write-503-retains-idempotency',async()=>{const{c,p}=await fixture();try{await p.evaluate(()=>window.__fnFault='lost-503');await p.locator('[data-fastnet-test]').click();await p.locator('[role=alert] button').click();await p.waitForTimeout(100);const keys=await p.evaluate(()=>window.__fnCalls.filter(c=>c.action==='run').map(c=>c.body.idempotencyKey));assert.equal(keys.length,2);assert.equal(keys[0],keys[1])}finally{await c.close()}})
 await run('three-failures-pause-sequential-poll',async()=>{const{c,p}=await fixture();try{await p.evaluate(()=>window.__fnFault='poll');await p.locator('[data-fastnet-test]').click();await p.getByText(/已暂停查询/).waitFor();const calls=await p.evaluate(()=>window.__fnCalls.filter(c=>c.action==='task'));assert.equal(calls.length,3);for(let i=1;i<3;i++)assert.ok(calls[i].time-calls[i-1].time>=1995);await p.waitForTimeout(2500);assert.equal(await p.evaluate(()=>window.__fnCalls.filter(c=>c.action==='task').length),3);assert.equal(await p.evaluate(()=>window.__quickstartCardFixture.stats.runs),1)}finally{await c.close()}})
 await run('hide-resume-same-task',async()=>{const{c,p}=await fixture();try{await p.locator('[data-fastnet-test]').click();await p.waitForFunction(()=>!!document.querySelector('[data-fastnet-card]')?.dataset.taskId);const id=await p.locator('[data-fastnet-card]').getAttribute('data-task-id');await p.locator('[data-toggle-external]').click();const n=await p.evaluate(()=>window.__fnCalls.length);await p.waitForTimeout(2200);assert.equal(await p.evaluate(()=>window.__fnCalls.length),n);await p.locator('[data-toggle-external]').click();await p.waitForFunction(id=>document.querySelector('[data-fastnet-card]')?.dataset.taskId===id,id);assert.equal(await p.evaluate(()=>window.__quickstartCardFixture.stats.runs),1)}finally{await c.close()}})
 await run('restart-retains-results-no-auto-run',async()=>{const{c,p}=await fixture();try{await p.locator('[data-fastnet-test]').click();await p.evaluate(()=>window.__fnFault='restart');await reload(p);await p.getByText(/任务已中断/).waitFor();assert.match(await p.locator('.qs-card-metrics').innerText(),/950/);assert.equal(await p.evaluate(()=>window.__quickstartCardFixture.stats.runs),1)}finally{await c.close()}})
 report.status=report.checks.every(c=>c.status==='passed')?'passed':'failed';await b.close();fs.writeFileSync(path.join(out,'fault-browser-report.json'),JSON.stringify(report,null,2)+'\n');console.log(JSON.stringify(report));if(report.status!=='passed')process.exitCode=1
})().catch(e=>{console.error(e);process.exitCode=1})
