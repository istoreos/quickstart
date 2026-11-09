const fs = require('node:fs/promises');
const {chromium} = require('/var/cache/build-cache/users/911/projects/quickstart-f0ca7c908f5f/work/multi-lan-review-tools/node_modules/playwright');
(async()=>{
const browser = await chromium.launch({headless:true,executablePath:'/config/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell',args:['--no-sandbox']});
const context = await browser.newContext({viewport:{width:1440,height:1000},ignoreHTTPSErrors:true});
await context.addCookies([{name:'sysauth_http',value:process.env.QS_DEPLOY_SID,domain:'192.168.30.93',path:'/'}]);
const page = await context.newPage(); const errors=[]; const reads=[];
page.on('pageerror',error=>errors.push(error.message));
page.on('response',response=>{if(response.url().includes('/lanctrl/v2/networks/'))reads.push({url:response.url(),status:response.status()})});
try{
 const url='http://192.168.30.93/cgi-bin/luci/admin/quickstart/devicemanagement?network=all';
 await page.goto(url,{waitUntil:'domcontentloaded',timeout:60000});
 await page.locator('.network-sidebar').waitFor({timeout:40000});
 const options=await page.locator('.lan-selector select option').evaluateAll(options=>options.map(o=>({value:o.value,label:o.textContent})));
 for(const id of ['all','lan','lan1','lan2'])if(!options.some(o=>o.value===id))throw new Error('Missing network option: '+id);
 await page.locator('.page-state--loading').waitFor({state:'hidden',timeout:40000});
 if(await page.locator('.page-state--error').count())throw new Error(await page.locator('.page-state--error').innerText());
 await page.screenshot({path:process.env.QS_DEPLOY_EVIDENCE+'/devices-all.png',fullPage:true});
 for(const network of ['lan','lan1','lan2']){
  await page.locator('.network-navigation button[data-network='+JSON.stringify(network)+']').click();
  await page.waitForURL(url=>url.searchParams.get('network')===network);
  await page.waitForTimeout(1000);
  await page.locator('.page-state--loading').waitFor({state:'hidden',timeout:40000});
  if(await page.locator('.page-state--error').count())throw new Error(await page.locator('.page-state--error').innerText());
  if(await page.locator('.lan-selector select').inputValue()!==network)throw new Error('Selection changed');
 }
 await page.locator('.network-navigation button[data-network=all]').click();
 await page.waitForURL(url=>url.searchParams.get('network')==='all');
 await page.locator('.page-state--loading').waitFor({state:'hidden',timeout:40000});
 await page.screenshot({path:process.env.QS_DEPLOY_EVIDENCE+'/devices-all.png',fullPage:true});
 await page.setViewportSize({width:390,height:844});
 await page.locator('.lan-selector').waitFor();
 await page.waitForTimeout(500);
 if(await page.evaluate(()=>document.documentElement.scrollWidth>window.innerWidth)){const sizes=await page.evaluate(()=>({viewport:innerWidth,scroll:document.documentElement.scrollWidth,items:[...document.querySelectorAll('body *')].filter(e=>e.getBoundingClientRect().right>innerWidth+1&&e.getBoundingClientRect().width).slice(0,40).map(e=>({tag:e.tagName,text:e.textContent,cls:e.className,parent:e.parentElement.className,parentWidth:e.parentElement.getBoundingClientRect().width,flexWrap:getComputedStyle(e.parentElement).flexWrap,x:e.getBoundingClientRect().x,width:e.getBoundingClientRect().width,minWidth:getComputedStyle(e).minWidth,grid:getComputedStyle(e).gridTemplateColumns,container:getComputedStyle(e).containerType}))}));await fs.writeFile(process.env.QS_DEPLOY_EVIDENCE+'/overflow.json',JSON.stringify(sizes,null,2));throw new Error('Mobile horizontal overflow')}
 await page.screenshot({path:process.env.QS_DEPLOY_EVIDENCE+'/devices-mobile.png',fullPage:true});
 if(errors.length)throw new Error(JSON.stringify(errors));
 await fs.writeFile(process.env.QS_DEPLOY_EVIDENCE+'/browser.json',JSON.stringify({status:'passed',url,options,reads,errors,checks:['Real LuCI session and production bundle','All/lan/lan1/lan2 selector visible','Each network selection changes the URL without runtime errors']},null,2));
 console.log('Actual LuCI production page passed; all/lan/lan1/lan2 selectable');
}catch(error){
 await page.screenshot({path:process.env.QS_DEPLOY_EVIDENCE+'/browser-failure.png',fullPage:true});
 await fs.writeFile(process.env.QS_DEPLOY_EVIDENCE+'/browser.json',JSON.stringify({status:'failed',url:page.url(),error:String(error),errors,reads},null,2));
 throw error;
}finally{await browser.close()}
})().catch(error=>{console.error(error.message);process.exitCode=1});
