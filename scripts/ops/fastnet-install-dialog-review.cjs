// Production installation component through ui-design or the reversible LuCI fixture.
const {chromium}=require('playwright'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path')
const out=process.env.OUTPUT_DIR,device=!!process.env.DEVICE_URL,base=process.env.DEVICE_URL||'http://127.0.0.1:5180',sid=process.env.DEVICE_SESSION
fs.mkdirSync(out,{recursive:true})
;(async()=>{
 const browser=await chromium.launch(),report={status:'in_progress',layer:device?'actual authenticated LuCI homepage':'ui-design with production host and simulated service',checks:[],errors:[]}
 async function open({width=1440,pref='light',english=false,mode='uninstalled',failedChunk=false}={}){
  const c=await browser.newContext({ignoreHTTPSErrors:true,viewport:{width,height:1000},colorScheme:'dark',hasTouch:width<500}),requests=[]
  if(device)await c.addCookies(['sysauth_http','sysauth_https'].map(name=>({name,value:sid,url:base})))
  await c.addInitScript(({pref,english})=>{localStorage.setItem('quickstart.theme.preference',pref);if(english)Object.defineProperty(window,'vue_lang',{configurable:true,get:()=> 'en',set:()=>{}})},{pref,english})
  if(failedChunk)await c.route(/FastnetInstallDialog[^/]*\.js/,r=>r.fulfill({status:404,body:'missing'}))
  const p=await c.newPage();p.on('pageerror',e=>report.errors.push(e.message));p.on('request',r=>requests.push({path:new URL(r.url()).pathname,method:r.method()}))
  await p.goto(base+(device?'/cgi-bin/luci/admin/quickstart':'/?tools=0'+(mode==='error'?'&scene=install-error':'')));await p.locator('[data-home-settings]:not(:disabled)').waitFor()
  assert.equal(requests.filter(r=>r.path.includes('FastnetInstallDialog')).length,0,'install dialog must be lazy')
  if(!['uninstalled','error'].includes(mode))await p.evaluate(mode=>{const original=window.fetch;window.fetch=(input,options)=>{const url=String(input);if(url.includes('/app/check/')){if(mode==='checking')return new Promise((_,reject)=>options?.signal?.addEventListener('abort',()=>reject(new DOMException('Aborted','AbortError'))));return Promise.resolve(new Response(JSON.stringify({result:{status:mode==='update_required'?'installed':'uninstalled',installation:mode==='installing'?'installing':mode==='failed'?'failed':mode==='unknown'?'unknown':'completed'}}),{headers:{'Content-Type':'application/json'}}))}return original(input,options)}},mode)
  await p.locator('[data-home-settings]').click();await p.locator('[data-open-library]').click();await p.locator('[data-choice="plugin.fastnet"]').click();await p.locator('section.network-dialog').waitFor();if(!failedChunk){await p.locator('[data-fastnet-install-dialog]').waitFor();if(mode!=='checking')await p.waitForFunction(()=>!document.querySelector('[data-fastnet-install-dialog]')?.textContent.includes('正在确认')&&!document.querySelector('[data-fastnet-install-dialog]')?.textContent.includes('Checking the plugin'))}
  return {c,p,requests}
 }
 try{
  for(const pref of ['light','dark','system'])for(const width of [320,390,1440])for(const english of [false,true]){
   const {c,p,requests}=await open({pref,width,english});const d=p.locator('section.network-dialog'),primary=p.locator('[data-confirm-card-install]');await primary.waitFor({state:'visible'});await p.waitForFunction(()=>!document.querySelector('[data-confirm-card-install]')?.disabled)
   const metrics=await d.evaluate(e=>{const r=e.getBoundingClientRect(),b=e.querySelector('[data-confirm-card-install]'),s=getComputedStyle(b),buttons=[...e.querySelectorAll('.fastnet-install-actions button')].map(b=>{const r=b.getBoundingClientRect();return {x:r.x,right:r.right,h:r.height}});return {left:r.left,right:r.right,width:innerWidth,scroll:document.documentElement.scrollWidth,panel:getComputedStyle(e).backgroundColor,primary:s.backgroundColor,radius:s.borderRadius,buttons}})
   assert.ok(metrics.left>=0&&metrics.right<=width+1&&metrics.scroll<=width+1,JSON.stringify(metrics));assert.notEqual(metrics.primary,metrics.panel);assert.notEqual(metrics.radius,'0px');assert.ok(metrics.buttons.every(b=>b.h>=(width<500?44:40)));assert.ok(metrics.buttons[0].right+4<=metrics.buttons[1].x)
   assert.equal(requests.filter(r=>r.path.includes('/app/install/')).length,0);assert.equal(requests.filter(r=>/fastnet\/quickstart\/card\./.test(r.path)).length,0)
   await d.screenshot({path:path.join(out,`${pref}-${width}-${english?'en':'zh'}.png`)});await p.keyboard.press('Tab');const focus=await p.evaluate(()=>{const s=getComputedStyle(document.activeElement);return s.outlineStyle});assert.notEqual(focus,'none');await p.keyboard.press('Escape');await d.waitFor({state:'hidden'});assert.equal(await p.locator('[data-choice="plugin.fastnet"]').evaluate(e=>e===document.activeElement),true);report.checks.push({pref,width,english,metrics});await c.close()
  }
  if(!device){
   for(const mode of ['checking','installing','unknown','failed','update_required']){
    const {c,p}=await open({mode,pref:'dark',width:390});const d=p.locator('section.network-dialog');if(mode==='unknown')assert.equal(await p.locator('[data-confirm-card-install]').count(),0);if(mode==='checking')assert.equal(await p.locator('[data-confirm-card-install]').isDisabled(),true);await d.screenshot({path:path.join(out,`state-${mode}.png`)});report.checks.push({mode});await c.close()
   }
   const x=await open({mode:'error'});await x.p.locator('[data-confirm-card-install]:not(:disabled)').click();await x.p.locator('section.network-dialog [role=alert]').waitFor();await x.p.locator('section.network-dialog').screenshot({path:path.join(out,'state-error.png')});assert.equal(await x.p.evaluate(()=>window.__quickstartCardFixture.writes.length),0);assert.equal(await x.p.locator('[data-fastnet-card]').count(),0);await x.p.locator('[data-recheck-card-install]').click();await x.p.locator('[data-confirm-card-install]:not(:disabled)').waitFor();await x.p.locator('[data-confirm-card-install]').click();await x.p.locator('[data-confirm-choice]').waitFor();assert.equal(await x.p.evaluate(()=>window.__quickstartCardFixture.stats.runs),0);assert.equal(await x.p.evaluate(()=>window.__quickstartCardFixture.writes.length),0);report.checks.push({mode:'failed-install-retry-without-layout-write-or-test'});await x.c.close()
   const bad=await open({failedChunk:true});await bad.p.locator('section.network-dialog [role=alert]').waitFor();await bad.p.locator('section.network-dialog').getByRole('button',{name:'返回',exact:true}).click();await bad.p.locator('[data-choice="plugin.fastnet"]').waitFor();assert.equal(bad.requests.filter(r=>r.path.includes('/app/install/')).length,0);report.checks.push({mode:'host-dialog-chunk-failure-has-recovery'});await bad.c.close()
  }
  const x=await open();await x.p.locator('[data-confirm-card-install]:not(:disabled)').click();await x.p.locator('[data-confirm-choice]').waitFor({timeout:180000});assert.equal(await x.p.locator('[data-fastnet-card]').count(),0);assert.equal(x.requests.filter(r=>r.path.endsWith('/fastnet/card/run')).length,0);if(!device)assert.equal(await x.p.evaluate(()=>window.__quickstartCardFixture.writes.length),0);report.checks.push({mode:'successful-install-still-requires-placement-and-never-tests'});await x.c.close()
  assert.deepEqual(report.errors,[]);report.status='passed'
 }catch(e){report.status='failed';report.failure=String(e.message).replace(/(sysauth_(?:http|https)=)[^;\s]+/g,'$1<redacted>');throw e}
 finally{await browser.close();fs.writeFileSync(path.join(out,device?'checks-browser-report.json':'dialog-browser-report.json'),JSON.stringify(report,null,2)+'\n')}
 console.log(JSON.stringify({status:report.status,checks:report.checks.length}))
})().catch(e=>{console.error(e.message.replace(/(sysauth_(?:http|https)=)[^;\s]+/g,'$1<redacted>'));process.exitCode=1})
