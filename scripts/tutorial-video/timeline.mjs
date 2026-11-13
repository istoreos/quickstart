import assert from 'node:assert/strict';
export const frameSeconds=(seconds,fps=30)=>Math.ceil(seconds*fps)/fps;
export function sceneAudio(scene,audio){
 const texts=scene.segments||[scene.narration];
 return texts.map((text,i)=>{const id=scene.segments?`${scene.id}-${String(i+1).padStart(2,'0')}`:scene.id;const item=audio.find(x=>x.id===id);assert.ok(item&&item.duration>0,`Missing measured audio: ${id}`);return {...item,text}});
}
export function subtitleChunks(text,max=27){
 assert.ok(max>0);const chunks=[];let buf='';
 for(const ch of text){buf+=ch;if(buf.length>=max||/[。！？]/.test(ch)){chunks.push(buf);buf=''}}
 if(buf)chunks.push(buf);return chunks;
}
export function validatePlan(plan){
 assert.ok(plan.scenes.length);let previous=0;
 for(const [i,scene] of plan.scenes.entries()){const overlap=i>0?(plan.transitions?.[i-1]?.duration||0):0;assert.ok(scene.duration>0&&scene.start>=previous-overlap-1e-6);previous=scene.start+scene.duration;
  for(const a of scene.audio)assert.ok(a.offset>=0&&a.offset+a.duration<=scene.duration+.01,'audio exceeds scene');
  for(const cue of scene.cues||[])assert.ok(Math.abs(cue.errorMs)<=300,'cue exceeds 300ms');
 }
 const duration=plan.duration??previous;
 const [min,max]=plan.durationRange||[90,120];
 assert.ok(duration>=min&&duration<=max,`duration ${duration} is outside ${min}–${max}s`);
 let audioEnd=0;for(const track of plan.narrationTracks||[]){assert.ok(track.offset>=audioEnd-.001&&track.offset+track.duration<=duration+.001,'Narration overlaps or exceeds film');audioEnd=track.offset+track.duration}
 return true;
}
export function claimReviewRound(history){assert.ok(history.length<3,'Three review rounds exhausted');return history.length+1}
export function cameraFilter(camera){
 const {from,to,seconds,hold=0}=camera;
 for(const r of [from,to])assert.ok(r.w>0&&r.h>0&&r.x>=0&&r.y>=0&&r.x+r.w<=1920.01&&r.y+r.h<=1080.01&&Math.abs(r.w/r.h-16/9)<.001,'camera rectangle must fit the 16:9 source');
 assert.ok(seconds>0&&hold>=0);
 const u=`min(max((on-${hold*30})/${seconds*30},0),1)`,ease=`((${u})*(${u})*(3-2*(${u})))`;
 const v=(key)=>`${from[key]}+(${to[key]-from[key]})*${ease}`;
 return `scale=3840:2160:flags=lanczos,zoompan=z='1920/(${v('w')})':x='2*(${v('x')})':y='2*(${v('y')})':d=1:s=1920x1080:fps=30`;
}
