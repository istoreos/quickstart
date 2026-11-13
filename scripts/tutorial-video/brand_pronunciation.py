"""Shorten verified silence inside a brand name, never its spoken phonemes."""
import argparse,array,hashlib,json,math,os,re,shutil,wave
from pathlib import Path

def quiet_runs(samples,rate,start,end,threshold):
    window=max(1,round(rate*.005));runs=[];opened=None
    for i in range(start,end,window):
        stop=min(i+window,end);quiet=max((abs(x) for x in samples[i:stop]),default=0)<=threshold
        if quiet and opened is None:opened=i
        if not quiet and opened is not None:runs.append((opened,i));opened=None
    if opened is not None:runs.append((opened,end))
    return runs

def compact_pcm(samples,rate,gaps,maximum=.12,target=.06,threshold_db=-45):
    assert 0<target<maximum
    threshold=32767*10**(threshold_db/20);cuts=[]
    for start,end in gaps:
        a=max(0,round(start*rate));b=min(len(samples),round(end*rate))
        # Only trim silence positively identified inside the ASR word boundary.
        for left,right in quiet_runs(samples,rate,a,b,threshold):
            if (right-left)/rate<=maximum:continue
            keep=round(target*rate);begin=left+keep//2;finish=right-(keep-keep//2)
            cuts.append((begin,finish))
    output=array.array('h');cursor=0
    for begin,finish in sorted(cuts):
        assert begin>=cursor and all(abs(x)<=threshold for x in samples[begin:finish])
        output.extend(samples[cursor:begin]);cursor=finish
    output.extend(samples[cursor:]);return output,cuts

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--spec',required=True);args=parser.parse_args()
    root=Path(os.environ['TUTORIAL_OUTPUT']);spec=json.loads(Path(args.spec).read_text(encoding='utf-8'))
    control=spec['pronunciationControl'];target=root/'audio/opening-intro.wav';receipt=target.with_suffix('.brand-edit.json')
    sha=lambda p:hashlib.sha256(Path(p).read_bytes()).hexdigest()
    if receipt.exists():
        saved=json.loads(receipt.read_text(encoding='utf-8'))
        if sha(target)==saved['resultSha256']:print(json.dumps(saved));return
        raise RuntimeError('Audio changed after brand edit; use a new revision')
    alignment=json.loads((root/'alignment/opening-intro.json').read_text(encoding='utf-8'))
    assert alignment['audioSha256']==sha(target),'Alignment must match native audio'
    words=[]
    for w in alignment['words']:
        if re.search('[\u3400-\u9fff]',w['word']):break
        if re.search('[a-zA-Z]',w['word']):words.append(w)
    assert len(words)>=3,'Expected separate I / Store / OS pronunciation tokens'
    gaps=[(a['end'],b['start']) for a,b in zip(words,words[1:]) if b['start']>a['end']]
    with wave.open(str(target)) as f:
        params=f.getparams();assert params.nchannels==1 and params.sampwidth==2
        samples=array.array('h',f.readframes(f.getnframes()))
    output,cuts=compact_pcm(samples,params.framerate,gaps,control['maxInternalPauseSeconds'],control['targetInternalPauseSeconds'])
    native=target.with_suffix('.native.wav');shutil.copy2(target,native)
    temporary=target.with_suffix('.partial.wav')
    with wave.open(str(temporary),'wb') as f:f.setparams(params);f.writeframes(output.tobytes())
    temporary.replace(target)
    removed=sum(b-a for a,b in cuts)/params.framerate
    prefix_start=round(words[0]['start']*params.framerate);prefix_end=round(words[-1]['end']*params.framerate)-round(removed*params.framerate)
    threshold=32767*10**(-45/20)
    longest=max(((b-a)/params.framerate for a,b in quiet_runs(output,params.framerate,prefix_start,prefix_end,threshold)),default=0)
    report={'word':control['word'],'spokenText':control['spoken'],'nativeSha256':sha(native),'resultSha256':sha(target),
            'removedSilenceSeconds':removed,'longestRemainingQuietSeconds':longest,'thresholdDb':-45,'audioSpeed':1.0,
            'cuts':[{'startSeconds':a/params.framerate,'endSeconds':b/params.framerate,'peak':max(abs(x) for x in samples[a:b])} for a,b in cuts],
            'preserved':'Every PCM sample outside the listed low-amplitude silence cuts is unchanged.'}
    receipt.write_text(json.dumps(report,ensure_ascii=False,indent=2),encoding='utf-8');print(json.dumps(report,ensure_ascii=False))

if __name__=='__main__':main()
