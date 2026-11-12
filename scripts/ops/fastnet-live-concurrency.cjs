const{chromium}=require('playwright'),assert=require('node:assert/strict'),fs=require('node:fs'),path=require('node:path'),{execFile}=require('node:child_process')
const out=process.env.OUTPUT_DIR,base=process.env.DEVICE_URL,sid=process.env.DEVICE_SESSION,host=process.env.DEVICE_HOST
;(async()=>{
 const b=await chromium.launch(),report={status:'in_progress',checks:[]},contexts=[]
 const open=async()=>{const c=await b.newContext({ignoreHTTPSErrors:true,viewport:{width:1440,height:900}});contexts.push(c);await c.addCookies(['sysauth_http','sysauth_https'].map(name=>({name,value:sid,url:base})));const p=await c.newPage();await p.goto(base+'/cgi-bin/luci/admin/quickstart');await p.locator('[data-home-settings]').waitFor();return p}
 const core=()=>new Promise((resolve,reject)=>execFile('ssh',['-o','BatchMode=yes',host,'cat /etc/fastnet/card-state.json'],(e,stdout)=>e?reject(e):resolve(JSON.parse(stdout))))
 try{
  const first=await open();await first.locator('[data-home-settings]').click();await first.locator('[data-open-library]').click();await first.locator('[data-choice="plugin.fastnet"]').click();await first.locator('[data-confirm-card-install]').click();await first.locator('[data-confirm-choice]').click();await first.locator('[data-apply]').click();if(await first.locator('[data-confirm-finish]').count())await first.locator('[data-confirm-finish]').click();await first.locator('[data-cancel]').waitFor({state:'hidden'});await first.locator('[data-ordered-card="plugin.fastnet"]').scrollIntoViewIfNeeded();await first.locator('[data-fastnet-test]:not(:disabled)').waitFor()
  const second=await open();await second.locator('[data-ordered-card="plugin.fastnet"]').scrollIntoViewIfNeeded();await second.locator('[data-fastnet-test]:not(:disabled)').waitFor()
  await Promise.all([first.locator('[data-fastnet-test]').click(),second.locator('[data-fastnet-test]').click()]);for(const p of [first,second])await p.waitForFunction(()=>!!document.querySelector('[data-fastnet-card]')?.dataset.taskId)
  const id=await first.locator('[data-fastnet-card]').getAttribute('data-task-id');assert.equal(await second.locator('[data-fastnet-card]').getAttribute('data-task-id'),id);const during=await core();assert.equal(during.jobs.length,1);assert.equal(during.jobs[0].task.id,id)
  await first.locator('[data-fastnet-card]').getByRole('button',{name:/^(停止|Stop)$/}).click();for(const p of [first,second])await p.waitForFunction(()=>!document.querySelector('[data-fastnet-card]')?.dataset.taskId)
  const after=await core();assert.equal(after.jobs.length,1);assert.equal(after.jobs[0].task.state,'cancelled');report.taskId=id;report.checks.push('two isolated browser contexts submit independently, join one real job and observe the same cancellation');report.status='passed'
 }catch(e){report.status='failed';report.failure=e.message.split(sid).join('<redacted>');throw e}
 finally{await b.close();fs.writeFileSync(path.join(out,'checks-browser-report.json'),JSON.stringify(report,null,2)+'\n')}
 console.log(JSON.stringify(report))
})().catch(e=>{console.error(e.message);process.exitCode=1})
