const{chromium}=require('playwright'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path')
const out=process.env.OUTPUT_DIR,entry=process.env.CARD_ENTRY,base=process.env.PREVIEW_URL||'http://127.0.0.1:5180'
if(!out||!entry)throw Error('OUTPUT_DIR and CARD_ENTRY required')
fs.mkdirSync(out,{recursive:true})
;(async()=>{
 const b=await chromium.launch(),report={milestone:'FN04',status:'in_progress',checks:[],errors:[]}
 try{
  for(const mode of ['','missing','present','discovery-error']){
   const c=await b.newContext(),p=await c.newPage(),requests=[];p.on('pageerror',e=>report.errors.push(e.message));p.on('request',r=>requests.push(new URL(r.url()).pathname));await p.goto(base+'/?layout=baseline&tools=0&entry='+encodeURIComponent(entry)+(mode?'&external='+mode:''));await p.locator('[data-home-settings]').waitFor();await p.locator('[data-ordered-card=traffic]').waitFor()
   assert.equal(await p.locator('[data-config-domain=layout]:visible').count(),0)
   if(mode==='present'){
    await p.locator('[data-ordered-card="plugin.fastnet"]').scrollIntoViewIfNeeded();await p.locator('[data-external-fixture]').waitFor();assert.equal(requests.filter(u=>u===entry).length,1);report.checks.push('formal HomeWidget loads registered external module at saved layout position')
   }else if(mode){
    const missing=p.locator('[data-ordered-card="plugin.fastnet"] [data-missing-plugin]');await missing.waitFor();assert.equal(requests.filter(u=>u.startsWith('/luci-static/fastnet/')).length,0)
    if(mode==='missing'){

     await p.locator('[data-home-settings]').click();await p.locator('[data-display-settings="plugin.fastnet"]').click();await p.locator('[data-presentation-form=summary]').click();await p.locator('[data-adopt-display]').click();await p.locator('[data-apply]').click();await p.locator('[data-cancel]').waitFor({state:'hidden'})
     const saved=await p.evaluate(()=>window.__quickstartCardFixture.writes.filter(w=>w.domain==='layout'));assert.equal(saved.length,1);assert.equal(saved[0].value.schemaVersion,4);assert.equal(saved[0].value.widgetPreferences['plugin.fastnet'].mode,'summary');assert.equal(saved[0].value.rows.flatMap(r=>r.columns.flatMap(c=>c.cards)).filter(x=>x==='plugin.fastnet').length,1);assert.equal(JSON.stringify(saved).includes('luci-static'),false);report.checks.push('missing plugin summary edit saves v4 without losing identity or storing entry URL')
    }else{assert.match(await missing.innerText(),/无法确认插件状态/);report.checks.push('discovery failure preserves saved placeholder and existing traffic card')}
   }else{
    await p.locator('[data-home-settings]').click();await p.locator('[data-open-library]').click();const choice=p.locator('[data-choice="plugin.fastnet"]');await choice.waitFor();assert.equal(await choice.isDisabled(),true);assert.match(await choice.innerText(),/先安装或升级插件/);assert.equal(requests.filter(u=>u.startsWith('/luci-static/fastnet/')).length,0);report.checks.push('uninstalled FastNet stays visible in lightweight catalog; no executable resource requests')
   }
   await p.screenshot({path:path.join(out,(mode||'catalog')+'.png')});await c.close()
  }
  assert.deepEqual(report.errors,[]);report.status='passed'
 }catch(e){report.status='failed';report.failure=e.stack;throw e}
 finally{await b.close();fs.writeFileSync(path.join(out,'browser-report.json'),JSON.stringify(report,null,2)+'\n')}
 console.log(JSON.stringify(report))
})().catch(e=>{console.error(e);process.exitCode=1})
