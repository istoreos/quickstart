// Read-only UI check against ui-design.sh or the retained .7 test homepage.
const {chromium}=require('playwright'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path')
const out=process.env.OUTPUT_DIR,base=process.env.DEVICE_URL||'http://127.0.0.1:5180',sid=process.env.DEVICE_SESSION
fs.mkdirSync(out,{recursive:true})
;(async()=>{
 const browser=await chromium.launch(),report={status:'in_progress',checks:[],errors:[],layouts:[]}
 const context=await browser.newContext({viewport:{width:1440,height:1000},ignoreHTTPSErrors:true,colorScheme:'light'})
 if(sid)await context.addCookies(['sysauth_http','sysauth_https'].map(name=>({name,value:sid,url:base})))
 await context.addInitScript(()=>localStorage.setItem('quickstart.theme.preference','system'))
 const page=await context.newPage();page.on('pageerror',e=>report.errors.push(e.message))
 try{
  await page.goto(base+(sid?'/cgi-bin/luci/admin/quickstart':'/?external=present&tools=0'))
  const card=page.locator('[data-fastnet-card]'),dialog=page.locator('section.network-dialog')
  for(const width of [1440,390,320]){
   await page.setViewportSize({width,height:1000})
   await page.locator('[data-layout-card="plugin.fastnet"]').evaluate(e=>e.scrollIntoView({block:'center'}))
   await card.locator('[data-fastnet-test]').click()
   const frame=page.frameLocator('iframe[title*="FastNet"]')
   await frame.getByText('快速体验',{exact:true}).first().waitFor({state:'attached',timeout:30000})
   await page.locator('.fn-web-loading').waitFor({state:'hidden',timeout:20000})
   assert.equal(await dialog.locator('header h2').innerText(),'外网测速')
   assert.equal(await dialog.locator('header small').count(),0)
   assert.equal(await page.locator('.fn-web-toolbar').count(),0)
   const open=dialog.locator('[data-network-dialog-open]')
   assert.equal(await open.getAttribute('target'),'_blank');assert.equal(await open.getAttribute('rel'),'noopener')
   assert.equal(await open.getAttribute('href'),'/cgi-bin/luci/admin/services/fastnet/open')
   const layout=await dialog.evaluate(e=>{const h=e.querySelector('header'),f=e.querySelector('iframe');return {width:innerWidth,headerHeight:h.getBoundingClientRect().height,iframeOffset:f.getBoundingClientRect().top-e.getBoundingClientRect().top,overflow:e.scrollWidth>e.clientWidth+1}})
   assert.ok(layout.headerHeight<=62);assert.ok(layout.iframeOffset<=63);assert.equal(layout.overflow,false);report.layouts.push(layout)
   for(const theme of ['light','dark']){
    await page.emulateMedia({colorScheme:theme});await frame.locator('html[data-quickstart-theme='+theme+']').waitFor()
    await dialog.screenshot({path:path.join(out,`dialog-${width}-${theme}.png`)})
   }
   // Exercise the real child-to-parent focus boundary messages.
   await frame.locator('body').evaluate(()=>parent.postMessage({type:'fastnet:focus',backward:true},location.origin))
   await page.waitForFunction(()=>document.activeElement?.hasAttribute('data-network-dialog-close'))
   await frame.locator('body').evaluate(()=>parent.postMessage({type:'fastnet:focus',backward:false},location.origin))
   await page.waitForFunction(()=>document.activeElement?.hasAttribute('data-network-dialog-open'))
   if(width===1440){
    const popupPromise=page.waitForEvent('popup');await open.click();const popup=await popupPromise;await popup.waitForLoadState('domcontentloaded').catch(()=>{});await popup.close()
    report.checks.push('new-window action opens a separate page')
   }
   await frame.locator('button:visible').first().focus();await page.keyboard.press('Escape');await dialog.waitFor({state:'hidden'})
   assert.equal(await card.locator('[data-fastnet-test]').evaluate(e=>e===document.activeElement),true)
  }
  report.checks.push('single compact title row at 1440/390/320px in light and dark themes','iframe focus boundaries, Escape and opener focus restoration')
  await card.locator('[data-widget-more]').click();await page.locator('[data-widget-menu-item="details"]').click()
  assert.equal(await dialog.locator('header small').innerText(),'QUICKSTART');assert.equal(await dialog.locator('[data-network-dialog-open]').count(),0)
  await dialog.locator('[data-network-dialog-close]').click();report.checks.push('ordinary result details retain their existing header')
  assert.deepEqual(report.errors,[]);report.status='passed'
 }catch(e){report.status='failed';report.failure=e.message.replace(/([?&]token=)[^&\s]+/g,'$1<redacted>');await page.screenshot({path:path.join(out,'failure.png')});process.exitCode=1}
 finally{await browser.close();fs.writeFileSync(path.join(out,'browser-report.json'),JSON.stringify(report,null,2)+'\n')}
 console.log(JSON.stringify(report))
})().catch(e=>{console.error(e.message);process.exitCode=1})
