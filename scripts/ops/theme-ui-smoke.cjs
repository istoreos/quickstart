const assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path')
const {chromium}=require('playwright')

const host=process.env.QS_THEME_HOST,sid=process.env.QS_THEME_SID,out=process.env.QS_THEME_OUT,scheme=process.env.QS_THEME_SCHEME||'https'
if(!host||!sid||!out)throw Error('QS_THEME_HOST, QS_THEME_SID and QS_THEME_OUT are required')
fs.mkdirSync(out,{recursive:true})

async function waitTheme(page,preference,effective){
 await page.waitForFunction(([preference,effective])=>document.body.getAttribute('theme')===(preference==='system'?'normal':preference)&&document.body.dataset.quickstartTheme===effective&&localStorage.getItem('quickstart.theme.preference')===preference,[preference,effective])
 const state=await page.evaluate(()=>({body:document.body.getAttribute('theme'),effective:document.body.dataset.quickstartTheme,html:document.documentElement.dataset.quickstartTheme,colorScheme:document.documentElement.style.colorScheme,panel:getComputedStyle(document.querySelector('[data-ordered-home]')).getPropertyValue('--c-panel').trim(),icon:document.querySelector('[data-theme-selector] svg')?.getAttribute('data-theme-icon')}))
 assert.equal(state.effective,effective);assert.equal(state.html,effective);assert.equal(state.colorScheme,effective)
 assert.equal(state.icon,preference)
 assert.equal(state.panel,effective==='dark'?'#2c2c2c':'#fff')
 return state
}

async function choose(page,index){
 const selector=page.locator('[data-theme-selector]');await selector.locator('summary').click();const options=selector.getByRole('radio');assert.equal(await options.count(),3);await options.nth(index).click()
}

(async()=>{
 const browser=await chromium.launch({headless:true}),report={host,checks:[],errors:[],badResponses:[],states:{}}
 try{
  const context=await browser.newContext({viewport:{width:1440,height:950},colorScheme:'dark',locale:'zh-CN',ignoreHTTPSErrors:true})
  await context.addCookies([{name:'sysauth_http',value:sid,domain:host,path:'/'},{name:'sysauth_https',value:sid,domain:host,path:'/',secure:true}])
  await context.addInitScript(()=>{if(!sessionStorage.getItem('quickstart.theme.smoke-started')){localStorage.removeItem('quickstart.theme.preference');sessionStorage.setItem('quickstart.theme.smoke-started','1')}})
  const page=await context.newPage();page.setDefaultNavigationTimeout(20000)
  page.on('pageerror',error=>report.errors.push(error.message));page.on('console',message=>{if(message.type()==='error')report.errors.push(message.text())});page.on('response',response=>{if(response.status()>=400)report.badResponses.push({path:new URL(response.url()).pathname,status:response.status()})})
  await page.goto(`${scheme}://${host}/cgi-bin/luci/admin/quickstart/`,{waitUntil:'domcontentloaded'});await page.locator('[data-theme-selector]').waitFor();await page.waitForTimeout(1200)
  assert.equal(await page.locator('input[type=password]').count(),0,'authenticated homepage required')
  const firstCanvas=await page.locator('.echart canvas').first().elementHandle()

  await choose(page,1);report.states.light=await waitTheme(page,'light','light');await page.screenshot({path:path.join(out,'light-1440.png'),fullPage:true})
  await choose(page,2);report.states.dark=await waitTheme(page,'dark','dark');await page.screenshot({path:path.join(out,'dark-1440.png'),fullPage:true})
  if(firstCanvas){const current=await page.locator('.echart canvas').first().elementHandle();assert.equal(await firstCanvas.evaluate((old,node)=>old.isSameNode(node),current),false,'ECharts canvas must be recreated after theme change');report.checks.push('ECharts recreated on effective theme change')}

  await choose(page,0);report.states.systemDark=await waitTheme(page,'system','dark');await page.screenshot({path:path.join(out,'system-dark-1440.png'),fullPage:true})
  await page.emulateMedia({colorScheme:'light'});report.states.systemLight=await waitTheme(page,'system','light');report.checks.push('system preference follows live prefers-color-scheme changes')

  await page.evaluate(()=>document.body.setAttribute('theme','dark'));report.states.hostDark=await waitTheme(page,'dark','dark');report.checks.push('external LuCI body theme changes synchronize and persist')
  await page.reload({waitUntil:'domcontentloaded'});await page.locator('[data-theme-selector]').waitFor();report.states.reloaded=await waitTheme(page,'dark','dark');report.checks.push('explicit preference survives reload')

  await page.setViewportSize({width:390,height:900});await page.locator('[data-theme-selector] summary').click();assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);await page.screenshot({path:path.join(out,'dark-390-menu.png'),fullPage:true});report.checks.push('theme menu remains usable without horizontal overflow at 390px')
  assert.deepEqual(report.errors,[]);assert.deepEqual(report.badResponses,[])
  await context.close()
 }finally{await browser.close();fs.writeFileSync(path.join(out,'report.json'),JSON.stringify(report,null,2)+'\n')}
 console.log(`Theme UI smoke ok: ${host}; ${report.checks.length} checks`)
})().catch(error=>{console.error(error.stack||error);process.exitCode=1})
