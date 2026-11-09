const {chromium}=require('/var/cache/build-cache/users/911/projects/quickstart-f0ca7c908f5f/work/multi-lan-review-tools/node_modules/playwright');
const assert=require('node:assert/strict');
const fs=require('node:fs/promises');
const path=require('node:path');
(async()=>{
const out=process.argv[2];await fs.mkdir(out,{recursive:true});
const browser=await chromium.launch({headless:true,executablePath:'/config/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell',args:['--no-sandbox']});
const results=[];
try {
for(const c of [{name:'desktop',width:1440,height:1000},{name:'wide',width:1920,height:1080},{name:'dark',width:1440,height:1000,theme:'dark'},{name:'english',width:1440,height:1000,lang:'en'},{name:'mobile',width:390,height:844},{name:'mobile-english',width:320,height:844,lang:'en'},{name:'empty',width:1440,height:1000,scenario:'empty'},{name:'single',width:1440,height:1000,single:true}]){
const page=await browser.newPage({viewport:{width:c.width,height:c.height}});const errors=[];page.on('pageerror',e=>errors.push(e.message));
const query=new URLSearchParams({layout:'ordered',variant:'rows',filter:'all',lans:c.single?'custom':'2',network:c.single?'home':'all',theme:c.theme||'light',lang:c.lang||'zh-cn',scenario:c.scenario||'normal'});
await page.goto('http://127.0.0.1:5180/devicemanagement?'+query,{waitUntil:'networkidle'});
await page.locator('.primary-tabs').waitFor();
await page.locator('.page-state--loading').waitFor({state:'hidden'});
const sidebar=page.locator('.network-sidebar');assert.equal(await sidebar.isVisible(),!c.single&&c.width>900);
if(!c.single){assert.equal(await page.locator('.lan-selector').isVisible(),c.width<=900);if(c.width>900){const nav=await sidebar.boundingBox(),panel=await page.locator('.workspace-panel').boundingBox();assert.ok(nav.x+nav.width<panel.x);assert.equal(await page.locator('.network-navigation button[aria-current=true]').getAttribute('data-network'),'all')}}
assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>window.innerWidth),false,c.name+' horizontal overflow');
assert.deepEqual(errors,[]);
await page.screenshot({path:path.join(out,c.name+'.png'),fullPage:false});
results.push({name:c.name,width:c.width,passed:true});await page.close();
}
await fs.writeFile(path.join(out,'result.json'),JSON.stringify({status:'passed',results},null,2));console.log(JSON.stringify(results));
}finally{await browser.close()}
})().catch(e=>{console.error(e);process.exitCode=1});
