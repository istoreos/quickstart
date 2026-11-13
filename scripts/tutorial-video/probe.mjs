import {chromium} from 'playwright';
import fs from 'node:fs/promises';
import path from 'node:path';
let raw='';for await(const chunk of process.stdin) raw+=chunk;
const config=JSON.parse(raw),out=process.env.TUTORIAL_OUTPUT+(config.installationLab?'/installation-lab':'');
await fs.mkdir(path.join(out,'preflight'),{recursive:true});
const b=await chromium.launch(),c=await b.newContext({viewport:{width:1920,height:1080},locale:'zh-CN',timezoneId:'Asia/Shanghai',colorScheme:'light'});
try {
 await c.addCookies(['sysauth_http','sysauth_https'].map(name=>({name,value:config.sid,url:config.base})));
 const p=await c.newPage();await p.goto(config.base+'/cgi-bin/luci/admin/quickstart');
 await p.locator('[data-home-settings]').waitFor();await p.waitForTimeout(1500);
 await p.screenshot({path:path.join(out,'preflight/home.png')});
 const layout=await p.evaluate(()=>fetch('/cgi-bin/luci/istore/home/v1/layout/').then(r=>r.json()));
 await fs.writeFile(path.join(out,'preflight/layout.json'),JSON.stringify(layout,null,2));
 const summary=config.installationLab?{}:await p.evaluate(()=>fetch('/cgi-bin/luci/admin/services/fastnet/card/summary').then(r=>r.json()));
 await fs.writeFile(path.join(out,'preflight/summary.json'),JSON.stringify(summary,null,2));
 if(!config.installationLab){
 const target=await p.evaluate(()=>fetch('/cgi-bin/luci/admin/services/fastnet/embed?format=json&standalone=1').then(r=>r.json()));
 await p.goto(target.url);await p.getByRole('button',{name:'开始全链路体检',exact:true}).waitFor();await p.waitForTimeout(1000);
 await p.screenshot({path:path.join(out,'preflight/fastnet.png')});
 await fs.writeFile(path.join(out,'preflight/fastnet-text.txt'),await p.locator('body').innerText());
 }
 await p.goto(config.base+'/cgi-bin/luci/admin/store');await p.waitForTimeout(1800);
 await p.screenshot({path:path.join(out,'preflight/store.png')});
 await fs.writeFile(path.join(out,'preflight/store-text.txt'),await p.locator('body').innerText());
 console.log(JSON.stringify({status:'ready',service:summary.service,task:summary.task?.state,layoutState:layout.result?.state,historyCount:summary.historyCount}));
} finally {await c.close();await b.close()}
