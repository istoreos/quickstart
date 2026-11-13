import {chromium} from 'playwright';
import fs from 'node:fs/promises';
import path from 'node:path';
import assert from 'node:assert/strict';
import {performance} from 'node:perf_hooks';
import {claimReviewRound} from './timeline.mjs';

let raw='';for await(const c of process.stdin) raw+=c;
const cfg=JSON.parse(raw),root=process.env.TUTORIAL_OUTPUT;assert.ok(root,'TUTORIAL_OUTPUT required');
if(cfg.mode==='opening'){
 assert.ok(process.env.TUTORIAL_PARENT_OUTPUT&&root!==process.env.TUTORIAL_PARENT_OUTPUT,'opening revision must be isolated');
 const history=await fs.readFile(path.join(root,'review-history.json'),'utf8').catch(e=>{if(e.code==='ENOENT')return '[]';throw e});
 claimReviewRound(JSON.parse(history.replace(/^\uFEFF/,'')));
}
const home=cfg.base+'/cgi-bin/luci/admin/quickstart';
const out=path.join(root,'capture');await fs.mkdir(out,{recursive:true});
const browser=await chromium.launch();
const report={schemaVersion:1,mode:cfg.mode,checks:[],stages:{},errors:[],capturedAt:new Date().toISOString()};
const privacy=`[data-layout-card="devices"] .canvas-devices,[data-canvas-device] {filter:blur(14px)!important} [data-layout-card="connection"] dd,pre {filter:blur(12px)!important} [data-layout-card="sharing"],[data-layout-card="remote"],[data-layout-card="docker"],[data-layout-card="storage"]{filter:blur(14px)!important}`;

async function stage(name,fn){
 const c=await browser.newContext({viewport:{width:1920,height:1080},deviceScaleFactor:1,locale:'zh-CN',timezoneId:'Asia/Shanghai',colorScheme:'light',recordVideo:{dir:path.join(out,'raw'),size:{width:1920,height:1080},fps:30}});
 await c.addCookies(['sysauth_http','sysauth_https'].map(n=>({name:n,value:cfg.sid,url:cfg.base})));
 await c.addInitScript(({privacy})=>{
  try{localStorage.setItem('quickstart.theme.preference','light')}catch{}
  const install=()=>{if(!document.documentElement)return;const s=document.createElement('style');s.textContent=privacy;document.documentElement.append(s)};
  new MutationObserver(install).observe(document,{childList:true,subtree:false});document.addEventListener('DOMContentLoaded',install);
  document.addEventListener('mousemove',event=>{const e=document.getElementById('tutorial-cursor');if(e){e.style.left=event.clientX+'px';e.style.top=event.clientY+'px'}});
 },{privacy});
 await c.tracing.start({screenshots:true,snapshots:true,sources:false});
 const start=performance.now(),p=await c.newPage();const events=[];
 const mark=(type,data={})=>{const item={type,t:(performance.now()-start)/1000,...data};events.push(item);return item};
 p.on('pageerror',e=>report.errors.push({stage:name,message:e.message}));
 const api=async(url,body)=>{const r=await c.request.fetch(cfg.base+url,{method:body?'POST':'GET',data:body});assert.equal(r.status(),200,url);return r.json()};
 const sync=async()=>{
  await p.evaluate(()=>{const e=document.createElement('div');e.id='tutorial-sync';e.style='position:fixed;inset:0;background:rgb(255,0,255);z-index:2147483647';document.documentElement.append(e)});
  mark('sync-on');await p.waitForTimeout(300);await p.locator('#tutorial-sync').evaluate(e=>e.remove());mark('sync-off');
 };
 const point=async(locator,label)=>{
  assert.equal(await locator.count(),1,label+' must be unique');await locator.scrollIntoViewIfNeeded();await locator.waitFor({state:'visible'});
  const box=await locator.boundingBox();assert.ok(box);
  await p.evaluate(({box,label})=>{let e=document.getElementById('tutorial-focus');if(!e){e=document.createElement('div');e.id='tutorial-focus';document.body.append(e)}e.style=`position:fixed;pointer-events:none;z-index:2147483645;left:${box.x-7}px;top:${box.y-7}px;width:${box.width+14}px;height:${box.height+14}px;border:4px solid #1d70ef;box-shadow:0 0 0 4px #fff8;box-sizing:border-box`;e.setAttribute('aria-hidden','true')},{box,label});
  mark('highlight',{target:label,box});await p.waitForTimeout(600);
  await p.mouse.move(box.x+box.width*.65,box.y+box.height*.65,{steps:18});
  await p.evaluate(({x,y})=>{let e=document.getElementById('tutorial-cursor');if(!e){e=document.createElement('div');e.id='tutorial-cursor';e.innerHTML='<svg width="38" height="44" viewBox="0 0 38 44"><path d="M3 2V34L12 26L20 41L27 37L19 23L32 23Z" fill="white" stroke="#173c74" stroke-width="3"/></svg>';document.body.append(e)}e.style=`position:fixed;pointer-events:none;z-index:2147483646;left:${x}px;top:${y}px`},{x:box.x+box.width*.65,y:box.y+box.height*.65});
 };
 const click=async(locator,label)=>{await point(locator,label);mark('click',{target:label});await locator.click();await p.waitForTimeout(250);await p.locator('#tutorial-focus').evaluate(e=>e.remove()).catch(()=>{});};
 const audio=JSON.parse(await fs.readFile(path.join(root,'audio/manifest.json'),'utf8'));
 const say=async(id,locator,label)=>{
  const item=audio.find(a=>a.id===id);assert.ok(item,'missing audio '+id);const event=mark('narration',{id,duration:item.duration});
  if(locator)await point(locator,label);
  await p.waitForTimeout(Math.max(0,(event.t+item.duration+.15-(performance.now()-start)/1000)*1000));
  if(locator){mark('click',{target:label,narration:id,cueErrorMs:Math.round(((performance.now()-start)/1000-event.t-item.duration)*1000)});await locator.click();await p.locator('#tutorial-focus').evaluate(e=>e.remove()).catch(()=>{});}
  await p.waitForTimeout(400);
 };
 let success=false;
 try{await fn({p,c,api,mark,sync,click,point,say});success=true}
 catch(e){report.failure={stage:name,message:String(e.message).split(cfg.sid).join('<redacted>')};await p.screenshot({path:path.join(out,name+'-failure.png')});throw e}
 finally{
  mark('stage-end');await c.tracing.stop({path:path.join(out,name+'.trace.zip')});const video=p.video();await c.close();await video.saveAs(path.join(out,name+'.webm'));
  report.stages[name]={file:name+'.webm',success,events};await fs.writeFile(path.join(out,'capture-'+cfg.mode+'.json'),JSON.stringify(report,null,2));
 }
}

try{
 if(cfg.mode==='opening'){
  let original,lastWritten;
  try{await stage('opening',async({p,api,mark,sync,click})=>{
   await p.goto(home);await p.locator('[data-home-settings]:not(:disabled)').waitFor();
   original=await api('/cgi-bin/luci/istore/home/v1/layout/');assert.equal(original.result.state,'saved');
   await fs.writeFile(path.join(out,'opening-layout-before.json'),JSON.stringify(original,null,2));
   const before=await api('/cgi-bin/luci/admin/services/fastnet/card/summary');
   const receipt=JSON.parse(await fs.readFile(path.join(process.env.TUTORIAL_PARENT_OUTPUT,'capture/measurement.json'),'utf8'));
   assert.deepEqual(before.latestResults,receipt.after.latestResults,'recorded results changed: do not splice another measurement');
   const value=structuredClone(original.result.value);
   for(const row of value.rows)for(const col of row.columns)col.cards=col.cards.map(id=>id==='plugin.fastnet'?null:id);
   value.rows.unshift({id:'tutorial-fastnet',template:'single',height:380,heightMode:'content',columns:[{weight:100,cards:['plugin.fastnet'],split:50}]});
   const saved=await api('/cgi-bin/luci/istore/home/v1/layout/',{expectedVersion:original.result.version,value});assert.equal(saved.success,0);lastWritten=saved.result.version;
   await p.reload();await p.locator('[data-home-settings]:not(:disabled)').waitFor();
   const card=p.locator('[data-layout-card="plugin.fastnet"]');await card.locator('[data-fastnet-card]').waitFor();
   await card.locator('.qs-card-metrics > div').first().locator('strong').filter({hasText:/921\s*Mbps/}).waitFor();
   await p.evaluate(()=>window.scrollTo(0,0));await p.waitForTimeout(800);
   const box=await card.boundingBox();assert.ok(box.y>60&&box.y+box.height<1000,'card and homepage chrome must fit');
   await sync();mark('home-ready',{box,downloadMbps:before.latestResults.download.value/1e6});
   await p.screenshot({path:path.join(out,'opening-home.png')});await p.waitForTimeout(6500);
   await click(card.locator('[data-fastnet-test]'),'open-fastnet');
   const frame=p.frameLocator('iframe[title="FastNet 完整测速界面"]');
   await frame.getByRole('button',{name:'开始全链路体检',exact:true}).waitFor({timeout:20000});
   const dialog=p.getByRole('dialog',{name:'外网测速',exact:true});assert.equal(await dialog.count(),1);
   await p.locator('.fn-web-loading').waitFor({state:'hidden'});mark('fastnet-ready',{dialogBox:await dialog.boundingBox()});
   await p.screenshot({path:path.join(out,'opening-fastnet.png')});await p.waitForTimeout(5000);
   await click(dialog.locator('[data-network-dialog-close]'),'close-fastnet');await dialog.waitFor({state:'hidden'});
   for(const selector of ['#tutorial-cursor','#tutorial-focus'])await p.locator(selector).evaluateAll(es=>es.forEach(e=>e.remove()));
   await card.locator('[data-fastnet-card]').waitFor();assert.match(await card.locator('.qs-card-metrics > div').first().innerText(),/921\s*Mbps/);
   mark('home-returned',{box:await card.boundingBox()});await p.screenshot({path:path.join(out,'opening-return.png')});await p.waitForTimeout(9000);
   const after=await api('/cgi-bin/luci/admin/services/fastnet/card/summary');assert.deepEqual(after.latestResults,before.latestResults);
   report.checks.push('homepage and card fit together; card opens real FastNet; home displays rounded 921 Mbps from the same 921.4 measurement; no test run');
  });}finally{
   if(original&&lastWritten){const c=await browser.newContext();await c.addCookies([{name:'sysauth_http',value:cfg.sid,url:cfg.base}]);
    try{const current=await(await c.request.get(cfg.base+'/cgi-bin/luci/istore/home/v1/layout/')).json();assert.equal(current.result.version,lastWritten,'concurrent layout change; do not overwrite');const restored=await(await c.request.post(cfg.base+'/cgi-bin/luci/istore/home/v1/layout/',{data:{expectedVersion:lastWritten,value:original.result.value}})).json();assert.equal(restored.success,0);assert.deepEqual(restored.result.value,original.result.value);report.restored=true}finally{await c.close()}}
  }
 }
 if(cfg.mode==='installed'){
  await stage('installed',async({p,mark,sync,point})=>{
   await p.goto(cfg.base+'/cgi-bin/luci/admin/store');const card=p.locator('.app-container').filter({has:p.locator('.app-title[title="fastnet"]')});await card.waitFor();
   await card.scrollIntoViewIfNeeded();assert.match(await card.locator('.app-version').innerText(),/0\.8\.9/);await sync();await point(card.locator('.app-version'),'installed-version');mark('installed-ready');await p.waitForTimeout(8000);await p.screenshot({path:path.join(out,'installed.png')});report.checks.push('real software center reports FastNet 0.8.9 after update');
  });
 }
 if(cfg.mode==='install'){
  assert.equal(cfg.installationLab,true,'installation requires the configured backed-up lab');
  await stage('install',async({p,api,mark,sync,click})=>{
   await p.goto(cfg.base+'/cgi-bin/luci/admin/store');
   const card=p.locator('.app-container').filter({has:p.locator('.app-title[title="fastnet"]')});
   await card.waitFor();assert.equal(await card.count(),1);const before=await card.locator('.app-version').innerText();
   await card.scrollIntoViewIfNeeded();await sync();await p.waitForTimeout(2000);
   await click(card.getByText('更新',{exact:true}),'update-fastnet');mark('install-requested',{before});
   await p.waitForTimeout(1600);await p.screenshot({path:path.join(out,'install-start.png')});
   await fs.writeFile(path.join(out,'install-initial-text.txt'),await p.locator('body').innerText());
   // Only confirm the app-specific update if this firmware presents a dialog.
   const dialog=p.getByRole('dialog');
   if(await dialog.count()){
    const confirm=dialog.getByRole('button',{name:/^(确定|确认|安装|更新)$/});
    if(await confirm.count()===1)await click(confirm,'confirm-fastnet-update');
   }
   let after='';
   for(let i=0;i<100;i++){
    const r=await cFetchInstalled(p);const item=Array.isArray(r)?r.find(x=>x.name==='fastnet'||x.name==='app-meta-fastnet'||x.id==='fastnet'):null;
    const version=item?.version||item?.Version||'';
    if(version&&!before.includes(version)&&!version.includes(before.replace(/^v/,''))){after=version;break}
    await p.waitForTimeout(1500);
   }
   await fs.writeFile(path.join(out,'install-final-text.txt'),await p.locator('body').innerText());
   mark('install-observation-finished',{after});await p.screenshot({path:path.join(out,'install-finish.png')});await p.waitForTimeout(5000);
   report.install={before,observedVersion:after};
   // Package and service checks are performed over SSH after the UI task ends.
   report.checks.push('real software-center FastNet update command dispatched on backed-up ARM64 lab');
  });
 }
 if(cfg.mode==='measure'){
  await stage('run',async({p,api,mark,sync,click})=>{
   await p.goto(home);await p.locator('[data-home-settings]').waitFor();
   const before=await api('/cgi-bin/luci/admin/services/fastnet/card/summary');assert.equal(before.service.state,'ready');assert.ok(!before.task||!['running','queued'].includes(before.task.state));
   const target=await api('/cgi-bin/luci/admin/services/fastnet/embed?format=json&standalone=1');
   await p.goto(target.url);const begin=p.getByRole('button',{name:'开始全链路体检',exact:true});await begin.waitFor();await sync();await p.waitForTimeout(1600);
   await click(begin,'begin-test');await p.locator('.quick-experience[data-phase="running"]').waitFor({timeout:15000});mark('test-running');
   await p.locator('.quick-experience[data-phase="complete"]').waitFor({timeout:240000});mark('test-complete');
   const values=await p.locator('.device-panel .metric-value').allTextContents();assert.ok(values.some(v=>/Mbps|Gbps|MB\/s|KB\/s/.test(v)),JSON.stringify(values));
   const after=await api('/cgi-bin/luci/admin/services/fastnet/card/summary');assert.notDeepEqual(after.latestResults,before.latestResults);
   await fs.writeFile(path.join(out,'measurement.json'),JSON.stringify({before,after,deviceMetrics:values,browserMetrics:await p.locator('.browser-panel .metric-value').allTextContents()},null,2));
   await p.screenshot({path:path.join(out,'results.png')});await p.waitForTimeout(16000);
   report.checks.push('one real full-chain test completed and saved changed results');
  });
 }
 if(cfg.mode==='home'){
  let original,lastWritten;
  try{await stage('add',async({p,api,mark,sync,click,say})=>{
   await p.goto(home);await p.locator('[data-home-settings]').waitFor();
   original=await api('/cgi-bin/luci/istore/home/v1/layout/');assert.equal(original.result.state,'saved');
   await fs.writeFile(path.join(out,'layout-before.json'),JSON.stringify(original,null,2));
   const value=structuredClone(original.result.value);for(const row of value.rows)for(const col of row.columns)col.cards=col.cards.map(id=>id==='plugin.fastnet'?null:id);
   const initialized=await api('/cgi-bin/luci/istore/home/v1/layout/',{expectedVersion:original.result.version,value});assert.equal(initialized.success,0);lastWritten=initialized.result.version;
   await p.reload();await p.locator('[data-home-settings]:not(:disabled)').waitFor();await sync();await p.waitForTimeout(800);
   await say('add-01',p.locator('[data-home-settings]'),'customize');
   await say('add-02',p.locator('[data-open-library]'),'add-module');
   await say('add-03',p.locator('[data-choice="plugin.fastnet"]'),'choose-fastnet');await p.locator('[data-confirm-choice]').waitFor();
   await say('add-04',p.locator('[data-confirm-choice]'),'place-fastnet');
   await say('add-05',p.locator('[data-apply]'),'finish-layout');
   if(await p.locator('[data-confirm-finish]').isVisible()){await say('add-06',p.locator('[data-confirm-finish]'),'confirm-finish')}
   await p.locator('[data-cancel]').waitFor({state:'hidden'});const saved=await api('/cgi-bin/luci/istore/home/v1/layout/');lastWritten=saved.result.version;
   assert.equal(saved.result.value.rows.flatMap(r=>r.columns.flatMap(c=>c.cards)).filter(x=>x==='plugin.fastnet').length,1);
   await p.reload();const card=p.locator('[data-layout-card="plugin.fastnet"]');await card.scrollIntoViewIfNeeded();await p.locator('[data-fastnet-card]').waitFor();mark('card-persisted');
   for(let i=0;i<3;i++){await card.evaluate(e=>e.scrollIntoView({block:'center',behavior:'instant'}));await p.waitForTimeout(400)}
   const box=await card.boundingBox();assert.ok(box.y>=0&&box.y+box.height<=1081,'card must be completely in frame');mark('card-framed',{box});
   if(await p.locator('#tutorial-cursor').count())await p.locator('#tutorial-cursor').evaluate(e=>e.remove());await say('add-07');
   await p.screenshot({path:path.join(out,'card.png')});await p.waitForTimeout(10000);mark('card-hold-end');
   report.checks.push('one FastNet card saved and still present after reload');
  });}finally{
  // The capture is closed before restoration; never overwrite a concurrent writer.
  if(original){
   const c=await browser.newContext();await c.addCookies([{name:'sysauth_http',value:cfg.sid,url:cfg.base}]);
   try{const current=await(await c.request.get(cfg.base+'/cgi-bin/luci/istore/home/v1/layout/')).json();assert.equal(current.result.version,lastWritten,'concurrent layout change; manual recovery required');const restored=await(await c.request.post(cfg.base+'/cgi-bin/luci/istore/home/v1/layout/',{data:{expectedVersion:current.result.version,value:original.result.value}})).json();assert.equal(restored.success,0);assert.deepEqual(restored.result.value,original.result.value);report.restored=true}finally{await c.close()}
  }}
 }
 if(cfg.mode==='beta'){
  await stage('beta',async({p,mark,sync,point})=>{
   await p.goto('https://fw.koolcenter.com/iStoreOS/alpha/quickstart/1113-0.17.1-rc20261028-r2/');await p.getByRole('link',{name:'install_quickstart.sh',exact:true}).waitFor();await sync();
   await point(p.getByRole('link',{name:'install_quickstart.sh',exact:true}),'beta-installer');mark('beta-guide');await p.waitForTimeout(15000);await p.screenshot({path:path.join(out,'beta.png')});report.checks.push('real official BETA directory and installer available');
  });
 }
 report.status='passed';
}catch(e){report.status='failed';console.error(JSON.stringify(report.failure||{message:e.message}));process.exitCode=1}
finally{await browser.close();await fs.writeFile(path.join(out,'capture-'+cfg.mode+'.json'),JSON.stringify(report,null,2));console.log(JSON.stringify({status:report.status,mode:cfg.mode,checks:report.checks,restored:report.restored}))}

async function cFetchInstalled(p){return p.evaluate(()=>fetch('/cgi-bin/luci/admin/store/installed').then(r=>r.json()))}
