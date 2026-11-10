// Production external card in the ui-design host. No device measurements.
const {chromium}=require('playwright'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path')
const out=process.env.OUTPUT_DIR,base=process.env.PREVIEW_URL||'http://127.0.0.1:5180'
if(!out)throw Error('OUTPUT_DIR required')
fs.mkdirSync(out,{recursive:true})
;(async()=>{
 const browser=await chromium.launch(),report={status:'running',checks:[],errors:[]}
 try{
  const cases=[]
  for(const theme of ['light','dark'])for(const [width,height,form]of [[576,405,'standard'],[320,330,'standard'],[280,260,'standard'],[280,148,'summary'],[240,148,'summary']])cases.push({theme,width,height,form,scene:'ready'})
  for(const scene of ['empty','running','partial','permission'])cases.push({theme:'light',width:280,height:260,form:'standard',scene})
  cases.push({theme:'light',width:280,height:260,form:'standard',scene:'running',touch:true})
  cases.push({theme:'dark',width:240,height:148,form:'summary',scene:'ready',touch:true})
  cases.push({theme:'system',width:390,height:330,form:'standard',scene:'ready'})
  for(const item of cases){
   const context=await browser.newContext({viewport:{width:item.touch?390:1920,height:1000},colorScheme:'light',hasTouch:!!item.touch})
   await context.addInitScript(t=>localStorage.setItem('quickstart.theme.preference',t),item.theme)
   const page=await context.newPage();page.on('pageerror',e=>report.errors.push(e.message))
   await page.goto(`${base}/?external=present&tools=0&form=${item.form}&scene=${item.scene}`)
   const card=page.locator('[data-fastnet-card]');await card.waitFor();await card.scrollIntoViewIfNeeded()
   // Set the slot, so ResizeObserver follows the same path as real card resizing.
   await card.evaluate((e,{width,height})=>{const slot=e.closest('[data-layout-card]');slot.style.width=width+'px';slot.style.height=height+'px';e.parentElement.style.setProperty('height',height+'px','important');e.parentElement.style.minHeight='0';e.parentElement.style.flex='none'},item)
   if(item.scene==='ready'){
    await page.evaluate(async()=>{
     const api=window.__quickstartCardFixture.fullAPI
     const run=await api('/api/history/begin',{kind:'lan'})
     await api('/api/history/results',{id:run.id,results:{lanDownload:{status:'ok',value:57100000},lanUpload:{status:'ok',value:58900000}}})
     await api('/api/history/finish',{id:run.id})
    })
    // Refresh through the existing visibility handler; artificial small slots
    // must not depend on the unrelated fixture row covering a popup menu.
    await page.evaluate(()=>document.dispatchEvent(new Event('visibilitychange')))
    await card.locator('.fn-lan-metrics').getByText(/57.1/).waitFor()
   }
   await page.waitForTimeout(100)
   const geometry=await card.evaluate(e=>{
    const rect=s=>{const r=e.querySelector(s).getBoundingClientRect();return {top:r.top,bottom:r.bottom,left:r.left,right:r.right}}
    return {height:e.clientHeight,scrollHeight:e.scrollHeight,width:e.clientWidth,scrollWidth:e.scrollWidth,wan:rect('.fn-speed-stage'),lan:rect('.fn-lan-summary'),statuses:rect('.qs-card-statuses'),metrics:rect('.qs-card-metrics'),wanTime:rect('.qs-card-time')}
   })
   assert.ok(Math.abs(geometry.height-(item.height-2))<=1,JSON.stringify({item,geometry}))
   assert.ok(geometry.scrollHeight<=geometry.height+2,JSON.stringify({item,geometry}))
   assert.ok(geometry.scrollWidth<=geometry.width+1,JSON.stringify({item,geometry}))
   assert.ok(geometry.metrics.top>=geometry.wan.top-1&&geometry.wanTime.bottom<=geometry.wan.bottom+1,JSON.stringify({item,geometry}))
   assert.ok(geometry.wan.bottom<=geometry.lan.top+1,JSON.stringify(geometry))
   assert.ok(geometry.lan.bottom<=geometry.statuses.top+1,JSON.stringify(geometry))
   assert.equal(await page.evaluate(()=>window.__quickstartCardFixture.stats.runs),0)
   assert.equal(await page.locator('iframe').count(),0)
   if(item.theme==='system'){
    const before=await card.evaluate(e=>getComputedStyle(e).color)
    await page.emulateMedia({colorScheme:'dark'});await page.waitForTimeout(100)
    assert.notEqual(await card.evaluate(e=>getComputedStyle(e).color),before)
    assert.equal(await page.evaluate(()=>window.__quickstartCardFixture.stats.runs),0)
   }
   await card.screenshot({path:path.join(out,`${item.theme}-${item.width}-${item.height}-${item.form}-${item.scene}.png`)})
   report.checks.push({...item,geometry});await context.close()
  }
  // The existing Test action must still open the default complete FastNet page.
  const context=await browser.newContext(),page=await context.newPage()
  await page.goto(base+'/?external=present&tools=0');const card=page.locator('[data-fastnet-card]');await card.waitFor();await card.scrollIntoViewIfNeeded()
  await card.locator('[data-fastnet-test]').click()
  const frame=page.frameLocator('iframe[title*=FastNet]');await frame.getByText('快速体验',{exact:true}).first().waitFor({state:'attached'})
  await page.locator('.fn-web-loading').waitFor({state:'hidden'})
  assert.equal(await page.evaluate(()=>window.__quickstartCardFixture.stats.runs),0)
  await page.locator('[data-network-dialog-close]').click();assert.equal(await page.locator('iframe').count(),0)
  report.checks.push({id:'test-opens-existing-default-page-without-starting-measurement'})
  await context.close();assert.deepEqual(report.errors,[]);report.status='passed'
 }catch(e){report.status='failed';report.failure=e.stack;process.exitCode=1}
 finally{await browser.close();fs.writeFileSync(path.join(out,'browser-report.json'),JSON.stringify(report,null,2)+'\n')}
 console.log(JSON.stringify({status:report.status,checks:report.checks.length,errors:report.errors,failure:report.failure}))
})().catch(e=>{console.error(e);process.exitCode=1})
