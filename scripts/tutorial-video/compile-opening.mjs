import fs from 'node:fs/promises';
import path from 'node:path';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {chromium} from 'playwright';
import {frameSeconds,sceneAudio,subtitleChunks,validatePlan,cameraFilter} from './timeline.mjs';

const root=process.env.TUTORIAL_OUTPUT,parent=process.env.TUTORIAL_PARENT_OUTPUT;
assert.ok(root&&parent&&root!==parent,'revision requires an isolated output');
const read=(file)=>fs.readFile(file,'utf8').then(JSON.parse);
const spec=await read('home-fastnet-card-opening-v2.json');
const old=await read(path.join(root,'parent-edit-plan.json')),receipt=await read(path.join(root,'parent-manifest.json'));
const voice=await read(path.join(root,'audio/manifest.json'));
const capture=await read(path.join(root,'capture/capture-opening.json'));
assert.equal(capture.status,'passed');assert.deepEqual(capture.errors,[]);assert.equal(capture.restored,true);
const source=capture.stages.opening;
const event=(type,target)=>{const e=source.events.find(e=>e.type===type&&(!target||e.target===target));assert.ok(e,type+'/'+target);return e};
const raw=path.join(root,'capture',source.file),ffmpeg=process.env.TUTORIAL_FFMPEG;
const pixels=execFileSync(ffmpeg,['-v','error','-i',raw,'-vf','scale=1:1:flags=area','-f','rawvideo','-pix_fmt','rgb24','pipe:1'],{maxBuffer:16*1024*1024});
let marker=-1;for(let i=0;i<pixels.length;i+=3)if(pixels[i]>230&&pixels[i+1]<30&&pixels[i+2]>230){marker=i/3;break}
assert.ok(marker>=0,'opening calibration marker missing');
const offset=marker/30-event('sync-on').t;
const clip=(start,length,padStart=0)=>({file:raw,start:start+offset,length,padStart,padEnd:0});
const run=await read(path.join(parent,'capture/capture-measure.json'));
const re=(type,target)=>{const e=run.stages.run.events.find(e=>e.type===type&&(!target||e.target===target));assert.ok(e,type);return e.t};
const calibration=await read(path.join(parent,'render/calibration.json'));
const runClip=(start,length,padStart=0)=>({file:path.join(parent,'capture',run.stages.run.file),start:start+calibration.run.offset,length,padStart,padEnd:0});
const full={x:0,y:0,w:1920,h:1080};
// Keep navigation and the full card visible on entry; zoom to labels and actual results later.
const wide={x:200,y:72,w:1720,h:967.5};
const cardBox=event('home-ready').box;
const cardFocus={x:0,y:Math.max(0,cardBox.y-85),w:1260,h:708.75};
const resultView={x:512,y:248,w:1152,h:648};
const folder=path.join(root,'render');await fs.mkdir(folder,{recursive:true});
const plan={...old,tutorial:spec.id,revision:spec.revision,scenes:[],software:receipt.software,parentVideoSha256:receipt.videoSha256,
 checks:[...old.checks,'opening browser assertions passed','opening layout restored','measurement unchanged'],
 opening:{calibration:{offset,markerFrame:marker},durationRange:spec.openingDurationRange,sourceMeasurement:receipt.measurement}};
for(const scene of spec.scenes.slice(0,6)){
 const a=sceneAudio(scene,voice)[0];a.offset=.18;
 let duration=frameSeconds(Math.max(scene.minSeconds,a.duration+.6));
 let clips=[],camera={from:full,to:full,seconds:1,hold:0},cues=[];
 if(scene.id==='opening-home'){
  clips=[clip(event('home-ready').t+.2,duration)];camera={from:full,to:wide,hold:.8,seconds:1.9};
 }else if(scene.id==='opening-enter'){
  const click=event('click','open-fastnet').t,desired=a.offset+a.duration+.15,start=click-desired;
  assert.ok(start>=event('home-ready').t,'insufficient pre-click footage');
  duration=frameSeconds(Math.max(duration,event('fastnet-ready').t-start+.35));clips=[clip(start,duration)];
  camera={from:wide,to:wide,seconds:1,hold:0};cues=[{target:'open-fastnet',at:desired,errorMs:150,sourceTime:click}];
 }else if(scene.id==='opening-test'){
  const click=re('click','begin-test'),start=Math.max(re('sync-off')+.18,click-1.65),desired=a.offset+a.duration+.15;
  const pad=Math.max(0,desired-(click-start));
  duration=frameSeconds(Math.max(duration,desired+1.05));clips=[runClip(start,click+1.05-start,pad)];
  camera={from:full,to:{x:400,y:100,w:1520,h:855},hold:0,seconds:.7};cues=[{target:'begin-test',at:desired,errorMs:150,sourceTime:click}];
 }else if(scene.id==='opening-result'){
  duration=frameSeconds(Math.max(duration,3.15));clips=[runClip(re('test-complete')-.2,duration)];
  camera={from:{x:400,y:100,w:1520,h:855},to:resultView,hold:.1,seconds:.55};
 }else if(scene.id==='opening-return'){
  clips=[clip(event('home-returned').t+.3,duration)];camera={from:cardFocus,to:full,hold:1.25,seconds:1.35};
 }else{
  clips=[clip(event('home-returned').t+.5,duration)];camera={from:full,to:full,hold:0,seconds:1};
 }
 const used=clips.reduce((sum,c)=>sum+c.length+c.padStart+c.padEnd,0);if(used<duration)clips.at(-1).padEnd+=duration-used;
 plan.scenes.push({...scene,start:0,duration,audio:[a],clips,crop:null,zoom:false,camera,cameraFilter:cameraFilter(camera),cues,overlay:path.join(folder,scene.id+'.png')});
}
for(const scene of spec.scenes.slice(6)){
 if(scene.id==='parallel-note'){
  const a=sceneAudio(scene,voice)[0];a.offset=.25;const duration=frameSeconds(Math.max(scene.minSeconds,a.duration+.9));
  plan.scenes.push({...scene,start:0,duration,audio:[a],clips:[runClip(re('test-complete')+.6,duration)],crop:{x:512,y:260,w:1152,h:648},zoom:true,cues:[],overlay:path.join(folder,scene.id+'.png')});
 }else{const original=old.scenes.find(s=>s.id===scene.id);assert.ok(original);plan.scenes.push({...original,start:0,reuseRendered:true})}
}
let at=0;for(const s of plan.scenes){s.start=at;at+=s.duration}
plan.transitions=plan.scenes.slice(0,-1).map((scene,i)=>({type:i<6?'fade':'pixelize',duration:i<6?.1:.3}));
plan.duration=at-plan.transitions.reduce((sum,t)=>sum+t.duration,0);validatePlan(plan);
let overlap=0;for(let i=0;i<plan.scenes.length;i++){plan.scenes[i].start-=overlap;overlap+=plan.transitions[i]?.duration||0}
plan.duration=at-overlap;plan.opening.duration=plan.scenes[6].start;
assert.ok(plan.opening.duration>=spec.openingDurationRange[0]&&plan.opening.duration<=spec.openingDurationRange[1],`opening is ${plan.opening.duration}s`);
assert.ok(plan.duration>=90&&plan.duration<=120,`full film is ${plan.duration}s`);
const b=await chromium.launch(),p=await b.newPage({viewport:{width:1920,height:1080}});
const labels={
 'opening-home':['iStoreOS 首页','测速成绩，打开首页就能看','效果预览 · 首页卡片需 Quickstart BETA'],
 'opening-enter':['FastNet 新版','从首页卡片，进入 FastNet','真实浏览器操作'],
 'opening-test':['FastNet 实测','点一下，开始全链路体检','同一场实测录像 · 等待已剪辑'],
 'opening-result':['本次外网下载','921.4 Mbps','本次实测结果 · 速度随线路与时段变化'],
 'opening-return':['Quickstart BETA','把最近成绩，留在首页','同一次实测 · 首页按界面规则取整显示 921 Mbps'],
 'opening-next':['接下来一起做','安装  →  测速  →  添加卡片','安装与更新过程已剪短等待']};
labels['parallel-note']=['多来源 · 并发下载','减少单个节点的影响','测速方式更新 · 实际速度受线路与时段影响'];
try{for(const scene of plan.scenes.filter(s=>!s.reuseRendered)){
 const [tag,title,note]=labels[scene.id];
 // Small editorial labels preserve software branding and results; no replacement UI.
 await p.setContent(`<html><meta charset="utf-8"><style>*{box-sizing:border-box}body{margin:0;font-family:"Microsoft YaHei",sans-serif;color:#fff}.tag{position:absolute;left:270px;top:18px;background:#12375ef2;padding:10px 20px;border-left:5px solid #52a9ff;font-size:25px;box-shadow:5px 5px 0 #12375e33}.note{position:absolute;right:35px;bottom:30px;background:#12375eef;padding:9px 14px;font-size:19px;border-left:4px solid #70bcff}.bridge{position:absolute;left:290px;top:710px;background:#12375ef5;padding:23px 32px;border:3px solid #70bcff;font-size:36px;box-shadow:8px 8px 0 #24529655}</style><div class="tag">${tag}</div><div class="note">${note}</div>${scene.id==='opening-next'?`<div class="bridge">${title}</div>`:''}</html>`);
 await p.screenshot({path:scene.overlay,omitBackground:true});
}}finally{await b.close()}
await fs.writeFile(path.join(root,'edit-plan.json'),JSON.stringify(plan,null,2));
let srt='',idx=1;const stamp=s=>{const ms=Math.round(s*1000);return `${String(Math.floor(ms/3600000)).padStart(2,'0')}:${String(Math.floor(ms/60000)%60).padStart(2,'0')}:${String(Math.floor(ms/1000)%60).padStart(2,'0')},${String(ms%1000).padStart(3,'0')}`};
for(const scene of plan.scenes)for(const a of scene.audio){let cursor=0;for(const text of subtitleChunks(a.text)){const d=a.duration*text.length/a.text.length;srt+=`${idx++}\n${stamp(scene.start+a.offset+cursor)} --> ${stamp(scene.start+a.offset+cursor+d)}\n${text}\n\n`;cursor+=d}}
await fs.writeFile(path.join(root,'fastnet-home-card-zh-CN.srt'),srt);
console.log(JSON.stringify({status:'compiled',duration:plan.duration,opening:plan.opening.duration,shots:6,scenes:plan.scenes.length}));
