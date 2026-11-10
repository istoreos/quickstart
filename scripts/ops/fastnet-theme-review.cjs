// Actual production selector, service and portal exercised through both preview roots.
const {chromium}=require('playwright'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path')
const out=process.env.OUTPUT_DIR
if(!out)throw Error('OUTPUT_DIR required')
fs.mkdirSync(out,{recursive:true})
const base=process.env.PREVIEW_URL||'http://127.0.0.1:5180'
;(async()=>{
 const browser=await chromium.launch({headless:true}),report={milestone:'FN02',status:'in_progress',checks:[],errors:[]}
 const choose=async(page,p)=>{const s=page.locator('[data-theme-selector]').first();await s.locator('summary').click();await s.getByRole('radio',{name:{system:/跟随系统/,light:/^浅色/,dark:/^深色/}[p]}).click()}
 const state=async(page,p,e)=>{await page.waitForFunction(([p,e])=>document.body.dataset.quickstartTheme===e&&document.body.getAttribute('theme')===(p==='system'?'normal':p),[p,e]);await page.waitForFunction(([p,e])=>{const s=document.querySelector('[data-theme-selector]');return s?.dataset.themePreference===p&&s?.dataset.effectiveTheme===e},[p,e])}
 try{
  for(const root of ['ordered','baseline'])for(const os of ['light','dark'])for(const p of ['system','light','dark']){
   const c=await browser.newContext({colorScheme:os});await c.addInitScript(p=>localStorage.setItem('quickstart.theme.preference',p),p)
   const page=await c.newPage();page.on('pageerror',e=>report.errors.push(e.message));await page.goto(`${base}/?layout=${root}&tools=0`);await page.locator('[data-theme-selector]').first().waitFor();await state(page,p,p==='system'?os:p)
   await page.emulateMedia({colorScheme:os==='dark'?'light':'dark'});await state(page,p,p==='system'?(os==='dark'?'light':'dark'):p)
   report.checks.push({id:`${root}-${os}-${p}`,status:'passed'});await c.close()
  }
  const c=await browser.newContext({colorScheme:'light'}),page=await c.newPage(),other=await c.newPage()
  page.on('pageerror',e=>report.errors.push(e.message));await page.goto(base+'/?tools=0');await page.locator('[data-theme-selector]').waitFor();await choose(page,'dark');await page.reload();await state(page,'dark','dark')
  await other.goto(base+'/?tools=0');await other.locator('[data-theme-selector]').waitFor();await choose(other,'light');await state(page,'light','light')
  await page.evaluate(()=>document.body.setAttribute('theme','dark'));await state(page,'dark','dark');await state(other,'dark','dark')
  await page.locator('[data-fastnet-card]').getByRole('button',{name:'外网测速更多操作',exact:true}).click();await page.getByRole('menuitem',{name:'查看上次结果',exact:true}).click()
  const dialog=page.getByRole('dialog');await dialog.waitFor();await dialog.focus();await choose(other,'system');await state(page,'system','light')
  const light=await dialog.evaluate(e=>({panel:getComputedStyle(e).backgroundColor,token:getComputedStyle(e).getPropertyValue('--qs-card-panel').trim(),inline:e.style.getPropertyValue('--c-panel')}));assert.equal(light.inline,'');assert.equal(light.token,'#fff')
  await page.emulateMedia({colorScheme:'dark'});await state(page,'system','dark');const dark=await dialog.evaluate(e=>({panel:getComputedStyle(e).backgroundColor,token:getComputedStyle(e).getPropertyValue('--qs-card-panel').trim(),focused:e===document.activeElement}));assert.equal(dark.token,'#2c2c2c');assert.notEqual(dark.panel,light.panel);assert.equal(dark.focused,true)
  await page.screenshot({path:path.join(out,'open-dialog-live-dark.png')});await page.keyboard.press('Escape');await dialog.waitFor({state:'hidden'})
  // An unrelated route/query change must preserve the selected preference.
  await choose(page,'dark');await page.locator('.lab-scenes > summary').click();await page.getByLabel('色彩风格').selectOption('classic');await state(page,'dark','dark')
  report.checks.push({id:'reload-storage-LuCI-open-portal-query',status:'passed',light,dark});await c.close()
  const blocked=await browser.newContext();await blocked.addInitScript(()=>{const get=Storage.prototype.getItem,set=Storage.prototype.setItem;Storage.prototype.getItem=function(k){if(k==='quickstart.theme.preference')throw Error('storage denied');return get.call(this,k)};Storage.prototype.setItem=function(k,v){if(k==='quickstart.theme.preference')throw Error('storage denied');return set.call(this,k,v)}})
  const bp=await blocked.newPage();bp.on('pageerror',e=>report.errors.push(e.message));await bp.goto(base+'/?tools=0');await bp.locator('[data-theme-selector]').waitFor();await choose(bp,'dark');await state(bp,'dark','dark');await choose(bp,'light');await state(bp,'light','light');await blocked.close();report.checks.push({id:'storage-unavailable',status:'passed'})
  assert.deepEqual(report.errors,[]);report.status='passed'
 }catch(e){report.status='failed';report.failure=e.stack;throw e}
 finally{await browser.close();fs.writeFileSync(path.join(out,'browser-report.json'),JSON.stringify(report,null,2)+'\n')}
 console.log(JSON.stringify(report))
})().catch(e=>{console.error(e);process.exitCode=1})
