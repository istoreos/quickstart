// Check the restored Quick Experience in the real frontend and host iframe.
// Preview runs use the existing explicit fixtures; live mode does not run WAN.
const {chromium}=require('playwright'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path')
const out=process.env.OUTPUT_DIR,sid=process.env.DEVICE_SESSION,live=Boolean(sid)
fs.mkdirSync(out,{recursive:true})
;(async()=>{
 const browser=await chromium.launch(),report={status:'running',checks:[],layouts:[],errors:[]}
 const context=await browser.newContext({viewport:{width:1440,height:1000},ignoreHTTPSErrors:true,colorScheme:'light',reducedMotion:'reduce'})
 if(live)await context.addCookies(['sysauth_http','sysauth_https'].map(name=>({name,value:sid,url:'http://192.168.30.7'})))
 await context.addInitScript(()=>{
  localStorage.setItem('quickstart.theme.preference','system');localStorage.setItem('fastnet.appearance','system')
  // The design server's standalone page uses the same explicit mock bridge.
  if(opener)window.__quickstartCardFixture=opener.__quickstartCardFixture
 })
 await context.route('https://ai.koolcenter.com/**',route=>{
  if(live)return route.continue()
  const nat=new URL(route.request().url()).pathname.includes('/nat'),data=nat?{source:'ai-portal-nat',type:'nat-result',natLabel:'Full Cone'}:{source:'ai-portal-ipv6',type:'ipv6-result',scoreStrict:10,statusText:'可用'}
  return route.fulfill({contentType:'text/html; charset=utf-8',body:`<script>setTimeout(()=>parent.postMessage(${JSON.stringify(data)},'*'),100)</script>`})
 })
 const page=await context.newPage();page.on('pageerror',e=>report.errors.push(e.message))
 const home=live?'http://192.168.30.7:10000/cgi-bin/luci/admin/quickstart':'http://127.0.0.1:5180/?external=present&tools=0'
 const absent='.diagnostic-rail,.quick-journey,.quick-completion,.quick-experience .system-status-line'
 async function verify(frame,surface,width){
  const quick=frame.locator('.quick-experience'),start=quick.locator('.start-btn')
  await quick.waitFor();assert.equal(await start.count(),1);assert.equal(await start.isEnabled(),true)
  assert.match(await start.innerText(),/↗\s*开始全链路体检/)
  assert.equal(await frame.locator(absent).count(),0)
  assert.equal(await frame.getByRole('button',{name:/再次体检|查看技术详情/}).count(),0)
  assert.equal(await frame.locator('.log-card').count(),1);assert.equal(await frame.locator('.json-card details').getAttribute('open'),null)
  const layout=await quick.evaluate(e=>{
   const control=e.querySelector('.control-panel'),results=e.querySelector('.device-browser-container')
   return {width:innerWidth,resultsGap:results.getBoundingClientRect().top-control.getBoundingClientRect().bottom,resultsTop:results.getBoundingClientRect().top+scrollY,overflow:document.documentElement.scrollWidth>innerWidth+1}
  })
  assert.equal(layout.overflow,false);assert.ok(layout.resultsGap>=0&&layout.resultsGap<=24);report.layouts.push({surface,...layout})
  for(const theme of ['light','dark']){
   await frame.page().emulateMedia({colorScheme:theme});await frame.locator(`html[data-appearance=${theme}]`).waitFor()
   await frame.locator('.control-panel').screenshot({path:path.join(out,`${surface}-${width}-${theme}-header.png`)})
  }
  if(!live&&width===1440){
   await frame.evaluate(()=>{
    window.__quickCounts={streams:0,begins:0};window.__quickPartial=false
    const Original=EventSource;window.EventSource=class extends Original{
     constructor(url,...args){super(url,...args);if(String(url).includes('/api/quick/stream'))window.__quickCounts.streams++}
     addEventListener(name,listener,...args){return super.addEventListener(name,name==='summary'?event=>{const data=JSON.parse(event.data);if(window.__quickPartial)data.error='fixture WAN probe failed';listener({data:JSON.stringify(data)})}:listener,...args)}
    }
    const original=fetch;window.fetch=(url,options)=>{if(String(url).includes('/api/history/begin'))window.__quickCounts.begins++;return original(url,options)}
    let tick=0;performance.now=()=>tick+=1000
    window.__quickOriginalButton=document.querySelector('.quick-experience .start-btn')
   })
   for(let run=1;run<=2;run++){
    if(run===2)await frame.evaluate(()=>window.__quickPartial=true)
    await start.click();await frame.waitForFunction(()=>document.querySelector('.quick-experience').dataset.phase==='running')
    assert.equal(await start.isDisabled(),true)
    await frame.page().emulateMedia({colorScheme:run===1?'light':'dark'})
    await frame.waitForFunction(()=>document.querySelector('.quick-experience').dataset.phase==='complete',{},{timeout:60000})
    assert.equal(await start.isEnabled(),true);assert.equal(await frame.locator(absent).count(),0)
    assert.equal(await frame.evaluate(()=>window.__quickOriginalButton===document.querySelector('.quick-experience .start-btn')),true)
    assert.deepEqual(await frame.evaluate(()=>window.__quickCounts),{streams:run,begins:run})
    const top=await quick.locator('.device-browser-container').evaluate(e=>e.getBoundingClientRect().top+scrollY)
    assert.ok(Math.abs(top-layout.resultsTop)<=1,'completion must not insert content before results')
    if(run===2)assert.match(await quick.locator('.quick-error').innerText(),/fixture WAN probe failed/)
    const captions=await quick.locator('.metric-card').evaluateAll(es=>es.map(e=>({value:e.querySelector('.metric-value').textContent.trim(),subtitle:e.querySelector('.metric-subtitle')?.textContent.trim()})))
    for(const item of captions)if(item.subtitle){assert.notEqual(item.value,item.subtitle);assert.doesNotMatch(item.subtitle,/等待.*结果|starting/)}
    await quick.screenshot({path:path.join(out,`${surface}-${run===1?'complete':'partial'}.png`)})
   }
   await frame.locator('.json-card summary').click();assert.equal(await frame.locator('.json-card details').getAttribute('open'),'')
   await frame.locator('.json-card summary').click()
   for(const kind of ['nat','ipv6']){const button=quick.locator(`.panel-head.${kind}-panel`);await button.click();assert.equal(await quick.locator(`.${kind}-panel > .panel-body`).isVisible(),true);await button.click()}
   report.checks.push(`${surface}: one fixed button starts exactly one run, reruns after completion, preserves results position/theme and real errors; logs/JSON and NAT/IPv6 expand work`)
  }
  report.checks.push(`${surface} ${width}px: result-first layout, one entry, light/dark, no removed UI or overflow`)
 }
 try{
  for(const width of [1440,390,320]){
   await page.setViewportSize({width,height:1000});await page.goto(home)
   await page.locator('[data-layout-card="plugin.fastnet"]').scrollIntoViewIfNeeded();await page.locator('[data-fastnet-test]').click()
   await page.frameLocator('iframe[title*=FastNet]').locator('.quick-experience').waitFor();await page.locator('.fn-web-loading').waitFor({state:'hidden'})
   const frame=page.frames().find(f=>new URL(f.url()).pathname.startsWith('/apps/fastnet/'))
   await verify(frame,'iframe',width);await page.locator('[data-network-dialog-close]').click()
   let standalone
   if(live){standalone=await context.newPage();await standalone.setViewportSize({width,height:1000});await standalone.goto('http://192.168.30.7:3200/webui/')}
   else{const popup=page.waitForEvent('popup');await page.evaluate(()=>window.open('/apps/fastnet/webui/'));standalone=await popup;await standalone.setViewportSize({width,height:1000})}
   standalone.on('pageerror',e=>report.errors.push(e.message));await verify(standalone.mainFrame(),'standalone',width);await standalone.close()
  }
  assert.deepEqual(report.errors,[]);report.status='passed'
 }catch(e){report.status='failed';report.failure=e.message.replace(/([?&]token=)[^&\s]+/g,'$1<redacted>');await page.screenshot({path:path.join(out,'failure.png'),fullPage:true});process.exitCode=1}
 finally{await browser.close();fs.writeFileSync(path.join(out,'browser-report.json'),JSON.stringify(report,null,2)+'\n')}
 console.log(JSON.stringify(report))
})().catch(e=>{console.error(e.message);process.exitCode=1})
