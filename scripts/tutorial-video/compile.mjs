import fs from 'node:fs/promises';
import path from 'node:path';
import {execFileSync} from 'node:child_process';
import {chromium} from 'playwright';
import assert from 'node:assert/strict';
import {frameSeconds,sceneAudio,subtitleChunks,validatePlan} from './timeline.mjs';

const root=process.env.TUTORIAL_OUTPUT;assert.ok(root,'TUTORIAL_OUTPUT required');
const ffmpeg=process.env.TUTORIAL_FFMPEG||'ffmpeg';
const spec=JSON.parse(await fs.readFile('home-fastnet-card.json','utf8'));
const read=name=>fs.readFile(path.join(root,name),'utf8').then(JSON.parse);
const voice=await read('audio/manifest.json'),capture={};
for(const mode of ['measure','home','beta','install','installed']){
 const r=await read(`capture/capture-${mode}.json`);assert.equal(r.status,'passed',mode);assert.deepEqual(r.errors,[]);Object.assign(capture,r.stages);
 if(mode==='home')assert.equal(r.restored,true);
}
const measurement=await read('capture/measurement.json');assert.ok(measurement.after.latestResults.download.value>0);
const renderDir=path.join(root,'render');await fs.mkdir(renderDir,{recursive:true});
const when=(stage,type,target)=>{const e=capture[stage].events.find(e=>e.type===type&&(!target||e.target===target));assert.ok(e,`${stage}/${type}/${target}`);return e.t};
const corrections={};
for(const [name,s] of Object.entries(capture)){
 const file=path.join(root,'capture',s.file);
 const pixels=execFileSync(ffmpeg,['-v','error','-i',file,'-vf','scale=1:1:flags=area','-f','rawvideo','-pix_fmt','rgb24','pipe:1'],{maxBuffer:32*1024*1024});
 let first=-1;for(let i=0;i<pixels.length;i+=3)if(pixels[i]>230&&pixels[i+1]<30&&pixels[i+2]>230){first=i/3;break}
 assert.ok(first>=0,`missing synchronization marker: ${name}`);
 corrections[name]={offset:first/spec.fps-when(name,'sync-on'),markerFrame:first};
}
await fs.writeFile(path.join(renderDir,'calibration.json'),JSON.stringify(corrections,null,2));
const clip=(stage,start,length,padStart=0)=>({file:path.join(root,'capture',capture[stage].file),start:start+corrections[stage].offset,length,padStart,padEnd:0});
const measuredAudio=scene=>{let offset=.45;return sceneAudio(scene,voice).map(a=>{const item={...a,offset};offset+=a.duration+.45;return item})};
const plan={schemaVersion:1,tutorial:spec.id,fps:30,width:1920,height:1080,transitionSeconds:.3,software:{quickstart:'c3565a795e981c5b5ee7a95d06359a1fadde7071',fastnet:'0.8.9'},checks:['browser assertions passed','layout restored','actual measurement saved'],scenes:[]};
let at=0;
for(const scene of spec.scenes){
 let audio=measuredAudio(scene),duration=frameSeconds(Math.max(scene.minSeconds,audio.at(-1).offset+audio.at(-1).duration+1));
 let clips=[],crop=null,zoom=false,cues=[];
 if(scene.id==='intro'||scene.id==='results'){
  clips=[clip('run',when('run','test-complete')+.5,duration)];crop={x:512,y:260,w:1152,h:648};zoom=scene.id==='intro';
 }else if(scene.id==='parallel'){
  clips=[clip('run',when('run','test-running')+9,duration)];crop={x:512,y:190,w:1152,h:648};zoom=true;
 }else if(scene.id==='install'||scene.id==='run'){
  const stage=scene.id==='install'?'install':'run',target=stage==='install'?'update-fastnet':'begin-test';
  const event=when(stage,'click',target),start=Math.max(when(stage,'sync-off')+.2,event-1.8),desired=audio[0].offset+audio[0].duration+.15;
  const pad=Math.max(0,desired-(event-start));
  const first=clip(stage,start,event+1.3-start,pad);clips.push(first);
  if(stage==='install')clips.push(clip('installed',when('installed','installed-ready')+.2,5));
  else clips.push(clip('run',when('run','test-complete')-1.5,5.5));
  const minimum=clips.reduce((sum,c)=>sum+c.length+c.padStart,0);duration=frameSeconds(Math.max(duration,minimum));
  cues.push({target,at:desired,errorMs:150});
 }else if(scene.id==='beta'){
  clips=[clip('beta',when('beta','sync-off')+.2,duration)];
 }else if(scene.id==='add'){
  const narration=capture.add.events.filter(e=>e.type==='narration');assert.equal(narration.length,7);
  const start=narration[0].t-.4;
  const split=when('add','card-framed')+.3,nextStart=narration.at(-1).t-.2;
  const removedGap=Math.max(0,nextStart-split);
  audio=narration.map(e=>{const a=voice.find(x=>x.id===e.id);const click=capture.add.events.find(x=>x.type==='click'&&x.narration===e.id);const offset=Math.max(e.t-start,click?click.t-start-a.duration-.15:0);return {...a,text:scene.segments[Number(e.id.slice(-2))-1],offset,captureOffset:e.t-start}});
  audio.at(-1).offset-=removedGap;
  duration=frameSeconds(audio.at(-1).offset+audio.at(-1).duration+2);
  clips=removedGap>0?[clip('add',start,split-start),clip('add',nextStart,duration-(split-start))]:[clip('add',start,duration)];
  cues=capture.add.events.filter(e=>e.type==='click'&&e.narration).map(e=>{const a=audio.find(a=>a.id===e.narration);return {target:e.target,at:e.t-start,errorMs:Math.round((e.t-start-a.offset-a.duration)*1000),captureErrorMs:e.cueErrorMs}});
 }else if(scene.id==='outro'){
  clips=[clip('add',when('add','card-framed')+.3,duration)];const box=capture.add.events.find(e=>e.type==='card-framed').box;crop={x:Math.floor(box.x/2)*2,y:Math.floor((box.y-8)/2)*2,w:Math.floor(box.width/2)*2,h:Math.ceil((box.height+16)/2)*2};zoom=false;
 }
 const used=clips.reduce((sum,c)=>sum+c.length+c.padStart+c.padEnd,0);if(used<duration)clips.at(-1).padEnd+=duration-used;
 const item={...scene,start:at,duration,audio,clips,crop,zoom,cues,cardFrame:scene.id==='outro',overlay:path.join(renderDir,scene.id+'.png')};plan.scenes.push(item);at+=duration;
}
validatePlan(plan);
// Actual chapter transitions overlap only quiet scene tails.
plan.duration=at-(plan.scenes.length-1)*plan.transitionSeconds;
assert.ok(plan.duration>=90&&plan.duration<=120);
for(let i=0;i<plan.scenes.length;i++)plan.scenes[i].start-=i*plan.transitionSeconds;
const b=await chromium.launch(),p=await b.newPage({viewport:{width:1920,height:1080}});
const escape=s=>s.replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
const speed=(measurement.after.latestResults.download.value/1e6).toFixed(1);
try{for(const [i,scene]of plan.scenes.entries()){
 let note='',extra='';
 if(scene.id==='intro'){note=`本次实测下载 ${speed} Mbps · 结果随线路与时段变化`;extra=`<div style="position:absolute;left:80px;top:770px;background:#12233ff2;padding:20px 35px;border-left:8px solid #67b7ff"><small style="font-size:23px;color:#bcd8f7">本次路由器外网下载</small><div style="font-size:72px;font-weight:bold">${speed} <span style="font-size:30px">Mbps</span></div></div>`}
 if(scene.id==='parallel')note='真实测速片段 · 多来源并发下载';
 if(scene.id==='run')note='实际等待已剪短 · 本次测速使用 Windows 浏览器连接 x86 测试机';
 if(scene.id==='results')note='本次 NAT 超时、IPv6 未就绪 · 结果如实保留';
 if(scene.id==='install'){note='ARM64 实机更新：0.7.7 → 0.8.9 · 下载等待已剪短';extra='<div class="command"><small>也可在路由器终端执行</small><code>is-opkg install app-meta-fastnet</code></div>'}
 if(scene.id==='beta'){note='Quickstart BETA · 更新前先做好完整系统备份';extra='<div class="guide"><b>01 完整系统备份</b><b>02 --check 检查</b><b>03 按说明更新</b><small>下载链接及操作说明见配套图文</small></div>'}
 if(scene.id==='outro'){note='首页显示最近有效结果 · 留意每项测量时间';extra='<div style="position:absolute;left:90px;top:175px;color:#173558;font-weight:bold;font-size:55px">测一次，把成绩留在首页。</div><div style="position:absolute;left:90px;top:820px;color:#245296;font-size:29px">FastNet · 安装 → 测速 → 添加首页卡片</div>'}
 await p.setContent(`<html><meta charset="utf-8"><style>*{box-sizing:border-box}body{margin:0;font-family:"Microsoft YaHei",sans-serif;color:#fff}.header{position:absolute;inset:0 0 auto;height:68px;background:#12233f;display:flex;align-items:center;gap:18px;padding:0 36px;font-size:27px;box-shadow:0 4px 0 #458afa}.pixel{width:28px;height:28px;border:6px solid #69baff;box-shadow:8px 8px 0 #2967d0;image-rendering:pixelated}.brand{font-weight:700;letter-spacing:1px}.chapter{margin-left:22px;color:#dceaff}.count{margin-left:auto;font-size:20px;color:#9ab7de}.note{position:absolute;top:88px;left:36px;background:#12233fed;padding:10px 18px;border-left:6px solid #67b7ff;font-size:21px;box-shadow:5px 5px 0 #10233d33}.command{position:absolute;left:90px;top:810px;background:#12233ff5;border:3px solid #67b7ff;padding:18px 24px;box-shadow:8px 8px 0 #2967d066}.command small{display:block;font-size:20px;color:#a9c8ec;margin-bottom:9px}.command code{font-size:31px;color:#fff}.guide{position:absolute;left:160px;top:720px;width:1600px;background:#12233ff2;border:3px solid #67b7ff;padding:28px;display:flex;gap:45px;flex-wrap:wrap;font-size:29px;box-shadow:8px 8px 0 #2967d066}.guide b{padding:12px 20px;background:#245296}.guide small{width:100%;font-size:22px;color:#c4daf4}</style><div class="header"><i class="pixel"></i><span class="brand">FastNet</span><span class="chapter">${escape(scene.chapter)}</span><span class="count">${i+1} / ${plan.scenes.length}</span></div>${note?'<div class="note">'+escape(note)+'</div>':''}${extra}</html>`);
 await p.screenshot({path:scene.overlay,omitBackground:true});
}}finally{await b.close()}
await fs.writeFile(path.join(root,'edit-plan.json'),JSON.stringify(plan,null,2));
let srt='',idx=1;
const stamp=s=>{const ms=Math.round(s*1000);return `${String(Math.floor(ms/3600000)).padStart(2,'0')}:${String(Math.floor(ms/60000)%60).padStart(2,'0')}:${String(Math.floor(ms/1000)%60).padStart(2,'0')},${String(ms%1000).padStart(3,'0')}`};
for(const scene of plan.scenes)for(const a of scene.audio){const chunks=subtitleChunks(a.text);let cursor=0;for(const text of chunks){const duration=a.duration*text.length/a.text.length;srt+=`${idx++}\n${stamp(scene.start+a.offset+cursor)} --> ${stamp(scene.start+a.offset+cursor+duration)}\n${text}\n\n`;cursor+=duration}}
await fs.writeFile(path.join(root,'fastnet-home-card-zh-CN.srt'),srt);
console.log(JSON.stringify({status:'compiled',duration:plan.duration,scenes:plan.scenes.length,cues:plan.scenes.flatMap(s=>s.cues).length,calibration:corrections}));
