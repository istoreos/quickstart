import fs from 'node:fs/promises';
import path from 'node:path';
import assert from 'node:assert/strict';
import {chromium} from 'playwright';
import {frameSeconds,validatePlan,cameraFilter} from './timeline.mjs';
import {wordEnd,retimeClips,normalize} from './voice-timeline.mjs';

const root=process.env.TUTORIAL_OUTPUT,parent=process.env.TUTORIAL_PARENT_OUTPUT;
assert.ok(root&&parent&&root!==parent,'A separate revision output is required');
const read=p=>fs.readFile(p,'utf8').then(t=>JSON.parse(t.replace(/^\uFEFF/,'')));
const spec=await read(process.argv[2]||'home-fastnet-card-voice-b-v4.json');
const parentPlan=await read(path.join(root,'parent-edit-plan.json'));
const source=spec.visualSourceFromParent?await read(path.join(parent,spec.visualSourceFromParent)):parentPlan;
const receipt=await read(path.join(root,'parent-manifest.json'));
const audio=await read(path.join(root,'audio/manifest.json'));
const alignment=await read(path.join(root,'alignment/manifest.json'));
const voice=new Map(audio.map(a=>[a.id,a]));
for(const a of audio)assert.equal(alignment[a.id].audioSha256,a.audioSha256,'Stale alignment '+a.id);
const plan={schemaVersion:1,tutorial:spec.id,revision:spec.revision,voiceStyle:'b',voice:spec.voice,
 durationRange:spec.durationRange,fps:30,width:1920,height:1080,subtitles:false,
 software:receipt.software,parentVideoSha256:receipt.videoSha256,previewFile:spec.previewFile||'fastnet-style-b-preview.mp4',specFile:process.argv[2]||'home-fastnet-card-voice-b-v4.json',
 transitions:[],transitionSeconds:.1,narrationTracks:[],scenes:[],pronunciations:spec.pronunciations};
const template=id=>{const s=source.scenes.find(s=>s.id===id);assert.ok(s,id);return s};
let cursor=0;
const track=(id,offset)=>{const a=voice.get(id);assert.ok(a,id);const t={...a,offset};plan.narrationTracks.push(t);return t};
const end=(id,variants)=>wordEnd(alignment[id],variants);
const last=id=>alignment[id].words.filter(w=>normalize(w.word)).at(-1).end;
const round=t=>Math.round(t*30)/30;
function add(id,duration,anchors=[],forceDuration=false){
 if(spec.reuseParentScenes&&!forceDuration&&!['opening-home','opening-enter','opening-test'].includes(id)){
  const approved=parentPlan.scenes.find(s=>s.id===id);assert.ok(approved);duration=approved.duration;
 }
 duration=frameSeconds(duration);const original=template(id);
 const mapped=anchors.map(a=>{
  const oldCue=original.cues.find(c=>c.target===a.target);assert.ok(oldCue,'Original click missing: '+a.target);
  const word= a.wordEnd;return {...a,old:oldCue.at,new:round(word+.15),at:round(word+.15),errorMs:Math.round((round(word+.15)-word)*1000),sourceOriginalAt:oldCue.at};
 });
 const s={...original,start:cursor,duration,audio:[],reuseRendered:false,sourceHasSubtitles:false,
  clips:retimeClips(original,duration,mapped),cues:mapped};
 if(s.camera){s.camera={...s.camera,hold:Math.min(s.camera.hold||0,Math.max(0,duration-.4)),seconds:Math.max(.15,Math.min(s.camera.seconds,duration-(s.camera.hold||0)-.1))};s.cameraFilter=cameraFilter(s.camera)}
 plan.scenes.push(s);cursor=round(cursor+duration-.1);return s;
}

// A single natural utterance spans the three opening shots. Clicks follow the
// actual spoken brand/command, without chopping the voice at every shot change.
const intro=track('opening-intro',.1);
const homeEnd=frameSeconds(end('opening-intro',['成绩了'])+.1+.3);
add('opening-home',homeEnd);
const enterStart=cursor,openWord=intro.offset+end('opening-intro',['FastNet']);
const enterEnd=frameSeconds(openWord+.15+.65);
add('opening-enter',enterEnd-enterStart,[{target:'open-fastnet',wordEnd:openWord-enterStart,audioId:intro.id,anchor:'FastNet'}]);
const testStart=cursor,beginWord=intro.offset+last('opening-intro');
const testEnd=frameSeconds(Math.max(beginWord+.15+.7,intro.offset+intro.duration+.25));
add('opening-test',testEnd-testStart,[{target:'begin-test',wordEnd:beginWord-testStart,audioId:intro.id,anchor:'外网下载'}]);
const resultStart=cursor,result=track('opening-result',resultStart+.12);
add('opening-result',Math.max(result.duration+.42,3.2));
const returnStart=cursor,returnVoice=track('opening-home-return',returnStart+.12);
const returnSplit=frameSeconds(end('opening-home-return',['留在首页'])+.12+.25);
add('opening-return',returnSplit);
add('opening-next',returnStart+.12+returnVoice.duration+.3-cursor);
plan.opening={duration:cursor,voiceParagraphs:3};

for(const id of ['install','run','parallel-note','results','beta']){
 const start=cursor,a=track(id,start+.12);let anchors=[];
 if(id==='install')anchors=[{target:'update-fastnet',wordEnd:.12+end(id,['安装或者更新','安装或更新']),audioId:id,anchor:'安装或者更新'}];
 if(id==='run')anchors=[{target:'begin-test',wordEnd:.12+end(id,['全链路体检','全链路体键']),audioId:id,anchor:'开始全链路体检'}];
 add(id,a.duration+.5,anchors);
}

const addStart=cursor,first=track('add-first',addStart+.12);
const place=track('add-place',first.offset+first.duration+.12);
const finish=track('add-finish',place.offset+place.duration+.12);
const addAnchors=[
 {target:'customize',audioId:first.id,wordEnd:first.offset-addStart+end(first.id,['自定义首页']),anchor:'自定义首页'},
 {target:'add-module',audioId:first.id,wordEnd:first.offset-addStart+end(first.id,['添加模块']),anchor:'添加模块'},
 {target:'choose-fastnet',audioId:place.id,wordEnd:place.offset-addStart+end(place.id,['外网测速']),anchor:'外网测速'},
 {target:'place-fastnet',audioId:place.id,wordEnd:place.offset-addStart+end(place.id,['确认放置']),anchor:'确认放置'},
 {target:'finish-layout',audioId:place.id,wordEnd:place.offset-addStart+end(place.id,['点完成','完成']),anchor:'完成'},
 {target:'confirm-finish',audioId:finish.id,wordEnd:finish.offset-addStart+end(finish.id,['确认就行','确定就行']),anchor:'确认就行'}
];
add('add',finish.offset+finish.duration+.65-addStart,addAnchors);
const outro=track('outro',cursor+.12);add('outro',outro.duration+.55);
if(voice.has('closing')){
 const start=cursor,a=track('closing',cursor+.12);
 const scene=add('outro',a.duration+.6,[],true);scene.id='closing';scene.narration=a.text;scene.chapter='酷友社小宝 · 感谢关注与支持';scene.overlay=path.join(root,'render/closing.png');
 await fs.mkdir(path.dirname(scene.overlay),{recursive:true});
 const b=await chromium.launch(),p=await b.newPage({viewport:{width:1920,height:1080}});
 try{
  await p.setContent('<html><meta charset="utf-8"><style>body{margin:0;color:#173558;font-family:"Microsoft YaHei",sans-serif}.top{height:68px;background:#12233f;color:white;font-size:28px;padding:16px 40px;box-sizing:border-box;border-bottom:4px solid #458afa}.pixel{display:inline-block;width:22px;height:22px;border:5px solid #69baff;box-shadow:5px 5px #2967d0;margin-right:20px;vertical-align:middle}.name{position:absolute;left:90px;top:155px;font-size:56px;font-weight:bold}.links{position:absolute;left:90px;top:237px;font-size:28px}.thanks{position:absolute;left:90px;top:825px;font-size:30px}</style><div class="top"><i class="pixel"></i>感谢关注与支持</div><div class="name">酷友社小宝</div><div class="links">开源地址 · 教程链接，都在视频下方</div><div class="thanks">关注酷友社小宝，一起体验更多实用功能。</div></html>');
  await p.screenshot({path:scene.overlay,omitBackground:true});
 }finally{await b.close()}
 plan.closing={start,duration:scene.duration,text:a.text};plan.endingFile=spec.endingFile;const previous=plan.scenes.at(-2);plan.endingPreview={start:previous.start,duration:cursor+.1-previous.start};
}
if(spec.reuseParentScenes){
 assert.equal(receipt.subtitles,false,'Only verified subtitle-free chapters may be reused');
 for(const s of plan.scenes){
  if(['opening-home','opening-enter','opening-test','closing'].includes(s.id))continue;
  const old=parentPlan.scenes.find(p=>p.id===s.id);assert.ok(old&&Math.abs(old.duration-s.duration)<.001,'Reused duration changed: '+s.id);
  s.reuseRendered=true;s.sourceHasSubtitles=false;
 }
}
plan.duration=round(cursor+.1);
plan.transitions=plan.scenes.slice(0,-1).map((s,i)=>({type:i<6?'fade':'pixelize',duration:.1}));
validatePlan(plan);
assert.ok(plan.scenes[3].duration-plan.scenes[3].camera.seconds>=2,'Result hold too short');
assert.ok(plan.narrationTracks.every(t=>!t.spokenText.includes('<|prosody:')),'Do not use rejected slow tags');
assert.ok(plan.narrationTracks[0].spokenText.includes(spec.pronunciations.iStoreOS));
await fs.writeFile(path.join(root,'edit-plan.json'),JSON.stringify(plan,null,2));
await fs.writeFile(path.join(root,'narration.txt'),plan.narrationTracks.map(t=>t.text).join('\n\n')+'\n');
console.log(JSON.stringify({status:'compiled',revision:plan.revision,duration:plan.duration,opening:plan.opening.duration,voiceStyle:'b',subtitles:false,spokenCues:plan.scenes.flatMap(s=>s.cues).length}));
