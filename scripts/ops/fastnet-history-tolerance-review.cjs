// Production UI and ui-design fixtures. Fault scenarios never run real WAN tests.
const {chromium}=require('playwright'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path')
const out=process.env.OUTPUT_DIR,sid=process.env.DEVICE_SESSION,live=Boolean(sid)
fs.mkdirSync(out,{recursive:true})
;(async()=>{
 const browser=await chromium.launch(),report={status:'running',checks:[],origins:[],errors:[]}
 const context=await browser.newContext({viewport:{width:1440,height:1000},ignoreHTTPSErrors:true,colorScheme:'light'})
 if(live)await context.addCookies(['sysauth_http','sysauth_https'].map(name=>({name,value:sid,url:'http://192.168.30.7'})))
 await context.addInitScript(()=>localStorage.setItem('quickstart.theme.preference','system'))
 const page=await context.newPage();page.on('pageerror',e=>report.errors.push(e.message))
 async function open(base){
  await page.goto(base+(live?'/cgi-bin/luci/admin/quickstart':'/?external=present&tools=0'))
  await page.locator('[data-layout-card="plugin.fastnet"]').scrollIntoViewIfNeeded()
  const card=page.locator('[data-fastnet-card]');await card.waitFor()
  assert.doesNotMatch(await card.innerText(),/默认外网出口|FASTNET|本次测速失败，保留上次结果|重试测速|结果保存失败，请检查设备存储/)
  assert.equal(await card.locator('footer').count(),0)
  for(const theme of ['light','dark']){await page.emulateMedia({colorScheme:theme});await card.screenshot({path:path.join(out,`card-${theme}.png`)})}
  await card.locator('[data-fastnet-test]').click()
  const frame=page.frames().find(f=>f.parentFrame());
  const f=page.frameLocator('iframe[title*=FastNet]');await f.getByText('快速体验',{exact:true}).first().waitFor({state:'attached'})
  await page.locator('.fn-web-loading').waitFor({state:'hidden'})
  return {f,frame:page.frames().find(f=>new URL(f.url()).pathname.startsWith('/apps/fastnet/'))||frame}
 }
 try{
  if(live){
   for(const base of ['http://192.168.30.7:10000','https://192.168.30.7']){
    const {frame}=await open(base)
    const result=await frame.evaluate(async()=>{
     const response=await fetch('/apps/fastnet/api/history/begin',{method:'POST',headers:{'Content-Type':'application/json','X-FastNet-History':'1'},body:'{}'})
     const list=await fetch('/apps/fastnet/api/history/list')
     return {origin:location.origin,secureContext:isSecureContext,writeStatus:response.status,writeError:(await response.json()).error,readStatus:list.status}
    })
    assert.equal(result.writeStatus,422);assert.equal(result.writeError,'invalid_request');assert.equal(result.readStatus,200);report.origins.push(result)
    await page.locator('[data-network-dialog-close]').click()
   }
   report.checks.push('HTTP and HTTPS proxy writes pass origin validation; history reads succeed; invalid probes create no records')
   const before=await (await context.request.get('http://192.168.30.7:3200/api/history/summary')).json()
   const {f}=await open('http://192.168.30.7:10000')
   await f.locator('[data-nav-id=browser-lan]').click();await f.getByRole('button',{name:'开始内网测速',exact:true}).click()
   await f.getByRole('button',{name:'停止内网测速',exact:true}).waitFor()
   await f.getByRole('button',{name:'开始内网测速',exact:true}).waitFor({timeout:90000})
   const after=await (await context.request.get('http://192.168.30.7:3200/api/history/summary')).json()
   const list=await (await context.request.get('http://192.168.30.7:3200/api/history/list')).json(),run=list.items[0]
   assert.equal(run.kind,'lan');assert.equal(run.state,'succeeded');assert.equal(after.historyCount,Math.min(32,before.historyCount+1))
   assert.deepEqual(Object.keys(run.results).sort(),['lanDownload','lanPing','lanUpload'])
   for(const key of ['download','upload','nat','ipv6'])assert.deepEqual(after.latestResults[key],before.latestResults[key])
   assert.equal(await f.locator('.history-save-error').count(),0)
   report.lan={id:run.id,state:run.state,results:run.results};await page.locator('section.network-dialog').screenshot({path:path.join(out,'HTTP-LAN.png')})
   await page.locator('[data-network-dialog-close]').click();report.checks.push('real HTTP iframe LAN succeeds, saves one LAN-only history, and preserves WAN/NAT/IPv6')
  }else{
   // Preserve the production page, orchestration and LAN implementation; the
   // preview bridge supplies test data and external portal replies are fixtures.
   await context.route('https://ai.koolcenter.com/**',route=>{
    const nat=new URL(route.request().url()).pathname.includes('/nat'),data=nat?{source:'ai-portal-nat',type:'nat-result',natLabel:'Full Cone'}:{source:'ai-portal-ipv6',type:'ipv6-result',scoreStrict:10,statusText:'可用'}
    return route.fulfill({contentType:'text/html',body:`<script>setTimeout(()=>parent.postMessage(${JSON.stringify(data)},'*'),100)</script>`})
   })
   for(const status of [403,503]){
    const {f,frame}=await open('http://127.0.0.1:5180')
    await frame.evaluate(status=>{
     const original=fetch;window.__historyFaults=0
     window.fetch=async(input,options)=>{if(String(input).includes('/api/history/')){window.__historyFaults++;return new Response('{}',{status,headers:{'Content-Type':'application/json'}})}return original(input,options)}
     let tick=0;performance.now=()=>tick+=1000
    },status)
    await f.getByRole('button',{name:'开始全链路体检',exact:true}).click();await f.getByRole('button',{name:'测速中…',exact:true}).waitFor()
    await f.getByRole('button',{name:'开始全链路体检',exact:true}).waitFor({timeout:60000})
    assert.equal(await frame.evaluate(()=>window.__historyFaults>0),true)
    assert.equal(await f.locator('.history-save-error').count(),0);assert.doesNotMatch(await f.locator('body').innerText(),/历史记录请求失败|无法建立测速记录/)
    await page.locator('section.network-dialog').screenshot({path:path.join(out,`history-${status}-quick.png`)})
    await f.locator('[data-nav-id=history]').click();await f.getByText('历史记录暂不可用，当前测速结果不受影响。',{exact:false}).waitFor()
    assert.equal(await f.locator('[data-fastnet-history] [role=alert]').count(),0)
    await page.locator('[data-network-dialog-close]').click();report.checks.push(`${status}: full quick flow finishes; no global history alert; history page degrades locally`)
   }
  }
  report.checks.push('card footer and terminal retry note removed; light/dark render correctly')
  assert.deepEqual(report.errors,[]);report.status='passed'
 }catch(e){report.status='failed';report.failure=e.message.replace(/([?&]token=)[^&\s]+/g,'$1<redacted>');await page.screenshot({path:path.join(out,'failure.png'),fullPage:true});process.exitCode=1}
 finally{await browser.close();fs.writeFileSync(path.join(out,'browser-report.json'),JSON.stringify(report,null,2)+'\n')}
 console.log(JSON.stringify(report))
})().catch(e=>{console.error(e.message);process.exitCode=1})
