// Run after ui-design.sh and the optional explicit Go HTTP integration fixture.
const {chromium}=require(process.env.PLAYWRIGHT_MODULE);
const assert=require('node:assert/strict');
const fs=require('node:fs/promises');
const path=require('node:path');
const http=require('node:http');
const output=process.env.MULTI_LAN_BROWSER_OUTPUT;
if(!output)throw new Error('MULTI_LAN_BROWSER_OUTPUT required');
const executablePath=process.env.CHROMIUM_PATH;
(async()=>{
 await fs.mkdir(output,{recursive:true});
 const browser=await chromium.launch({headless:true,executablePath,args:['--no-sandbox']});
 const results=[],errors=[];
 const context=await browser.newContext({viewport:{width:1440,height:1000}});
 const page=await context.newPage();page.on('pageerror',e=>errors.push(e.message));
 const chooseNetwork=async(value)=>{
  if(await page.locator('.drawer-backdrop').isVisible())await page.keyboard.press('Escape');
  const target=page.locator('.network-navigation button[data-network='+JSON.stringify(value)+']');
  if(await target.isVisible())await target.click();else await page.locator('.lan-selector select').selectOption(value);
 };
 const open=async(query='')=>{await page.goto('http://127.0.0.1:5180/devicemanagement?layout=ordered&variant=rows&filter=all'+query);await page.locator('.device-table-wrap').waitFor()};
 async function check(name,fn){await fn();results.push({name,passed:true});console.log(name)}
 try {
 await check('single LAN hides selector and retains three tabs',async()=>{await open();assert.equal(await page.locator('.lan-selector').count(),0);assert.equal(await page.locator('.primary-tabs button').count(),3);await page.screenshot({path:path.join(output,'single-desktop.png'),fullPage:true})});
 await check('custom single LAN works without lan or br-lan',async()=>{await open('&lans=custom');assert.equal(await page.locator('.lan-selector').count(),0);assert.ok((await page.evaluate(()=>window.__quickstartDesign.requests)).some(r=>r.path.includes('network=home')))});
 await check('same MAC rows have independent keys, scoped drawers and row writes',async()=>{
  await open('&lans=2&network=all');const keys=await page.locator('.device-table tbody tr').evaluateAll(rows=>rows.map(row=>row.dataset.deviceId));assert.equal(new Set(keys).size,keys.length);assert.ok(keys.some(id=>id.startsWith('lan1'))&&keys.some(id=>id.startsWith('lan2')));
  await page.locator('.device-table tbody tr').filter({hasText:'lan2'}).first().click();await page.getByRole('button',{name:'设备资料',exact:true}).click();await page.locator('.profile-editor input').first().fill('lan2 独立备注');await page.getByRole('button',{name:'保存设备资料',exact:true}).click();await page.locator('.profile-editor .feedback.success').waitFor();assert.ok((await page.evaluate(()=>window.__quickstartDesign.requests)).some(r=>r.method==='POST'&&r.path.includes('device-profile/?network=lan2')));
  await page.getByRole('button',{name:'关闭',exact:true}).click();await page.screenshot({path:path.join(output,'multi-overview.png'),fullPage:true});
 });
 await check('network change discards drawer and draft, refresh preserves selection',async()=>{
  await page.locator('.device-table tbody tr').first().click();await page.getByRole('button',{name:'设备资料',exact:true}).click();await page.locator('.profile-editor input').first().fill('unsaved');await chooseNetwork('lan2');await page.locator('.device-table tbody tr').first().waitFor();assert.equal(await page.locator('.device-drawer').count(),0);assert.ok((await page.locator('.device-table tbody tr').allTextContents()).every(text=>text.includes('lan2')));await page.reload();await page.locator('.device-table tbody tr').first().waitFor();assert.equal(await page.locator('.lan-selector select').inputValue(),'lan2');
 });
 await check('slow old responses cannot overwrite the newly selected network',async()=>{
  await page.evaluate(()=>{const original=window.fetch;window.fetch=async(input,init)=>{const url=String(input);const response=await original(input,init);if(url.includes('/v2/devices/')&&url.includes('network=lan1'))await new Promise(resolve=>setTimeout(resolve,1500));return response}});
  await chooseNetwork('lan1');await chooseNetwork('lan2');await page.locator('.device-table tbody tr').first().waitFor();await page.waitForTimeout(1700);assert.ok((await page.locator('.device-table tbody tr').allTextContents()).every(text=>text.includes('lan2')));assert.equal(await page.locator('.device-drawer').count(),0);
 });
 await check('legacy access review is visible only for the affected LAN',async()=>{
  await open('&lans=2&network=all&legacy-quota=1');await page.locator('.device-table tbody tr').filter({hasText:'lan1'}).first().click();
  const warning='旧版断网限制仍保留，取消额度或周期重置不会自动恢复上网。确认后，请使用“预览恢复联网”恢复。';
  await page.getByText(warning,{exact:true}).waitFor();await page.getByRole('button',{name:'使用管理',exact:true}).click();await page.getByText(warning,{exact:true}).waitFor();
  await chooseNetwork('lan2');await page.locator('.device-table tbody tr').first().waitFor();await page.locator('.device-table tbody tr').first().click();await page.getByRole('button',{name:'使用管理',exact:true}).click();await page.locator('.usage-policy fieldset').first().waitFor();assert.equal(await page.getByText(warning,{exact:true}).count(),0);
 });
 await check('aggregate network settings require an explicit LAN',async()=>{await chooseNetwork('all');await page.getByRole('tab',{name:'局域网设置',exact:true}).click();await page.getByText('请选择一个局域网，管理分组、计划和网络设置',{exact:true}).waitFor();assert.equal(await page.locator('.lan-settings').count(),0)});
 await check('invalid URL never redirects writes to a different LAN',async()=>{await page.goto('http://127.0.0.1:5180/devicemanagement?layout=ordered&lans=2&network=deleted');await page.getByText('此局域网已不存在或不可访问',{exact:false}).waitFor();assert.equal(await page.locator('.primary-tabs').count(),0)});
 await check('401,403 and service errors do not fall back to default LAN',async()=>{for(const code of ['401','403','500']){await page.goto('http://127.0.0.1:5180/devicemanagement?layout=ordered&discovery='+code);await page.locator('[role="alert"]').waitFor();assert.equal(await page.locator('.primary-tabs').count(),0)}});
 await check('unsupported network endpoint preserves old single LAN API',async()=>{await open('&discovery=404');assert.equal(await page.locator('.lan-selector').count(),0);const requests=await page.evaluate(()=>window.__quickstartDesign.requests);assert.ok(requests.some(r=>r.path==='/lanctrl/v2/devices/'));assert.ok(!requests.some(r=>r.path.includes('network=')))});
 await check('390px light and dark layouts remain usable with keyboard',async()=>{
  await page.setViewportSize({width:390,height:844});await open('&lans=2&network=all');for(const theme of ['light','dark']){await page.evaluate(value=>document.body.setAttribute('theme',value),theme);const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>window.innerWidth);assert.equal(overflow,false);await page.screenshot({path:path.join(output,'multi-mobile-'+theme+'.png'),fullPage:true})}
  const row=page.locator('.device-card').first();await row.focus();await page.keyboard.press('Enter');await page.locator('.device-drawer').waitFor();await page.keyboard.press('Escape');assert.equal(await page.locator('.device-drawer').count(),0);
 });
 assert.deepEqual(errors,[]);
 if(process.env.PRODUCTION_WEB_ROOT){
  const root=process.env.PRODUCTION_WEB_ROOT;
  const server=http.createServer(async(req,res)=>{
   if(req.url.startsWith('/cgi-bin/')){const proxy=http.request({hostname:'127.0.0.1',port:19489,path:req.url,method:req.method,headers:req.headers},up=>{res.writeHead(up.statusCode,up.headers);up.pipe(res)});proxy.on('error',()=>{res.writeHead(503);res.end('{}')});req.pipe(proxy);return}
   let file=path.resolve(root,'.'+new URL(req.url,'http://localhost').pathname);if(!file.startsWith(root+path.sep))file=path.join(root,'index.html');
   try{const content=await fs.readFile(file);res.setHeader('Content-Type',file.endsWith('.js')?'text/javascript':file.endsWith('.css')?'text/css':'text/html');res.end(content)}catch{res.setHeader('Content-Type','text/html');res.end(await fs.readFile(path.join(root,'index.html')))}
  });await new Promise(resolve=>server.listen(19490,'127.0.0.1',resolve));
  try{await check('production bundle integrates real Go scope boundary and persistent profile service',async()=>{
   const prod=await browser.newPage({viewport:{width:1440,height:1000}});await prod.addInitScript(()=>{window.vue_base='/';window.vue_lang='zh-cn';window.quickstart_features=[];window.quickstart_configs={}});
   await prod.goto('http://127.0.0.1:19490/devicemanagement?network=all&filter=all');await prod.locator('.device-table tbody tr').first().waitFor();assert.equal(await prod.locator('.device-table tbody tr').count(),2);
   await prod.locator('.device-table tbody tr').filter({hasText:'lan2'}).click();await prod.getByRole('button',{name:'设备资料',exact:true}).click();await prod.locator('.profile-editor input').first().fill('浏览器联调 lan2');await prod.getByRole('button',{name:'保存设备资料',exact:true}).click();await prod.locator('.profile-editor .feedback.success').waitFor();
   await prod.reload();await prod.locator('.device-table tbody tr').filter({hasText:'浏览器联调 lan2'}).waitFor();const scoped=await prod.evaluate(async()=>{const url='/cgi-bin/luci/istore/lanctrl/v2/device-profile/?deviceId=mac%3A02%3A00%3A00%3A00%3A00%3A55&network=';return await Promise.all(['lan1','lan2'].map(async id=>(await (await fetch(url+id)).json()).result.profile.alias))});assert.notEqual(scoped[0],scoped[1]);assert.equal(scoped[1],'浏览器联调 lan2');await prod.screenshot({path:path.join(output,'production-backend.png'),fullPage:true});await prod.close();
  })}finally{await new Promise(resolve=>server.close(resolve))}
 }
 await fs.writeFile(path.join(output,'result.json'),JSON.stringify({status:'passed',results,errors},null,2));
 }catch(error){await page.screenshot({path:path.join(output,'failure.png'),fullPage:true});await fs.writeFile(path.join(output,'result.json'),JSON.stringify({status:'failed',results,error:String(error),errors},null,2));throw error}
 finally{await browser.close()}
})().catch(error=>{console.error(error);process.exitCode=1});
