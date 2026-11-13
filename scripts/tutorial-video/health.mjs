import {chromium} from 'playwright';
import {createRequire} from 'node:module';
const require=createRequire(import.meta.url);
const browser=await chromium.launch();
try{
 const context=await browser.newContext({viewport:{width:1920,height:1080}});
 const page=await context.newPage();await page.setContent('<p>浏览器录制环境就绪</p>');
 console.log(JSON.stringify({status:'ready',playwright:require('playwright/package.json').version,browser:browser.version(),viewport:page.viewportSize()}));
 await context.close();
}finally{await browser.close()}
