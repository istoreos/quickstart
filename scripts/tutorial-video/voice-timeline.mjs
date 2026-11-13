import assert from 'node:assert/strict';

export const normalize=s=>s.toLowerCase().replace(/[^a-z0-9\u3400-\u9fff]/g,'');
export function wordEnd(alignment,variants){
 let text='',ends=[];
 for(const word of alignment.words){const n=normalize(word.word);text+=n;ends.push(...Array(n.length).fill(word.end))}
 for(const variant of variants){const v=normalize(variant),at=text.indexOf(v);if(at>=0)return ends[at+v.length-1]}
 throw Error('ASR cue not found: '+variants.join(' / '));
}

// Retime only video. Preserve a short window at native speed around each click;
// compress/extend the intervening footage, while narration remains untouched.
export function retimeClips(scene,duration,anchors=[]){
 const core=[{old:0,new:0},...anchors.map(a=>({old:a.old,new:a.new})).sort((a,b)=>a.old-b.old),{old:scene.duration,new:duration}];
 for(let i=1;i<core.length;i++)assert.ok(core[i].old>core[i-1].old&&core[i].new>core[i-1].new,'Retiming cues must be ordered');
 const knots=[core[0]];
 for(let i=1;i<core.length-1;i++){
  const a=core[i],before=core[i-1],after=core[i+1];
  const h=Math.min(.22,(a.old-before.old)/3,(after.old-a.old)/3,(a.new-before.new)/3,(after.new-a.new)/3);
  knots.push({old:a.old-h,new:a.new-h},a,{old:a.old+h,new:a.new+h});
 }
 knots.push(core.at(-1));
 const map=t=>{let i=1;while(i<knots.length-1&&t>knots[i].old)i++;const a=knots[i-1],b=knots[i];return a.new+(t-a.old)*(b.new-a.new)/(b.old-a.old)};
 let at=0;const spans=[];
 const span=(clip,type,length,start)=>{if(length>0)spans.push({file:clip.file,type,start,from:at,to:at+length});at+=length};
 for(const clip of scene.clips){span(clip,'freeze',clip.padStart||0,clip.start);span(clip,'real',clip.length,clip.start);span(clip,'freeze',clip.padEnd||0,clip.start+Math.max(0,clip.length-1/30))}
 assert.ok(at>=scene.duration-.04,'Source clips must cover the original scene');
 const clips=[];
 for(const s of spans){
  const end=Math.min(s.to,scene.duration);if(s.from>=end)continue;
  const cuts=[s.from,...knots.filter(k=>k.old>s.from&&k.old<end).map(k=>k.old),end];
  for(let i=1;i<cuts.length;i++){
   const a=cuts[i-1],b=cuts[i],length=(Math.round(map(b)*30)-Math.round(map(a)*30))/30;if(length<.02)continue;
   const start=s.start+(s.type==='real'?a-s.from:0),sourceLength=b-a;
   if(s.type==='freeze'||sourceLength<1/30){clips.push({file:s.file,start,length:1/30,padStart:0,padEnd:Math.max(0,length-1/30),outputDuration:length,playbackRate:1})}
   else clips.push({file:s.file,start,length:sourceLength,padStart:0,padEnd:0,outputDuration:length,playbackRate:sourceLength/length});
  }
 }
 assert.ok(Math.abs(clips.reduce((sum,c)=>sum+c.outputDuration,0)-duration)<.04,'Retimed frames must match scene duration');
 return clips;
}
