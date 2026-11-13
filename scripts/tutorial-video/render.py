#!/usr/bin/env python3
"""Deterministic FFmpeg rendering from the verified edit plan, on Windows."""
import argparse,json,os,subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--output',type=Path,default=Path(os.environ.get('TUTORIAL_OUTPUT','tutorial-output')));p.add_argument('--scene');p.add_argument('--assemble-only',action='store_true');a=p.parse_args()
ffmpeg=os.environ.get('TUTORIAL_FFMPEG','ffmpeg')
plan=json.loads((a.output/'edit-plan.json').read_text(encoding='utf-8'));directory=a.output/'render'

def run(args,log):
    log.with_suffix('.args.json').write_text(json.dumps(args,ensure_ascii=False,indent=2),encoding='utf-8')
    if os.environ.get('TUTORIAL_RENDER_DRY_RUN')=='1':return
    with log.open('w',encoding='utf-8') as f:
        r=subprocess.run([ffmpeg,'-hide_banner','-nostdin','-filter_complex_threads','1','-y',*args],stdout=f,stderr=subprocess.STDOUT,timeout=300)
    if r.returncode:raise RuntimeError(log.read_text(encoding='utf-8',errors='replace')[-3500:])

def ass_time(t):
    cs=round(t*100);return f'{cs//360000:01}:{cs//6000%60:02}:{cs//100%60:02}.{cs%100:02}'

def subtitles(scene):
    lines=['[Script Info]','ScriptType: v4.00+','PlayResX: 1920','PlayResY: 1080','WrapStyle: 0','[V4+ Styles]',
      'Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding',
      'Style: Default,Microsoft YaHei,36,&H00FFFFFF,&H00FFFFFF,&H0012253E,&H6012253E,0,0,0,0,100,100,0,0,3,8,0,1,70,580,48,1',
      '[Events]','Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text']
    import re
    for audio in scene['audio']:
        text=audio['text'];chunks=[];buf=''
        for ch in text:
            buf+=ch
            if len(buf)>=27 or ch in '。！？':chunks.append(buf);buf=''
        if buf:chunks.append(buf)
        at=audio['offset']
        for chunk in chunks:
            end=at+audio['duration']*len(chunk)/len(text)
            safe=chunk.replace('{','').replace('}','')
            lines.append(f'Dialogue: 0,{ass_time(at)},{ass_time(end)},Default,,0,0,0,,{safe}')
            at=end
    target=directory/(scene['id']+'.ass');target.write_text('\n'.join(lines),encoding='utf-8-sig');return target

for scene in ([] if a.assemble_only else plan['scenes']):
    if a.scene and scene['id']!=a.scene:continue
    if scene.get('reuseRendered'):
        if plan.get('subtitles') is False and not scene.get('sourceHasSubtitles') is False:
            raise RuntimeError('Cannot reuse a chapter with burned subtitles in a subtitle-free revision')
        for suffix in (['.video.mp4'] if plan.get('narrationTracks') else ['.video.mp4','.audio.wav']):
            if not (directory/(scene['id']+suffix)).is_file():raise RuntimeError('Missing verified chapter '+scene['id'])
        print('REUSE '+scene['id'],flush=True)
        continue
    print('RENDER '+scene['id'],flush=True)
    args=[];filters=[];index=0
    for clip in scene['clips']:
        args+=['-threads','1','-ss',str(max(0,clip['start'])),'-t',str(clip['length']),'-i',clip['file']]
        rate=clip.get('playbackRate',1)
        f=f'[{index}:v]setpts=(PTS-STARTPTS)/{rate},fps=30'
        if clip['padStart'] or clip['padEnd']:
            f+=f",tpad=start_mode=clone:start_duration={clip['padStart']}:stop_mode=clone:stop_duration={clip['padEnd']}"
        if clip.get('outputDuration'):
            f+=f",tpad=stop_mode=clone:stop_duration={clip['outputDuration']},trim=duration={clip['outputDuration']},setpts=PTS-STARTPTS"
        filters.append(f+'[p'+str(index)+']');index+=1
    if index>1:filters.append(''.join(f'[p{i}]' for i in range(index))+f'concat=n={index}:v=1:a=0[base]')
    else:filters.append('[p0]null[base]')
    args+=['-i',scene['overlay']];overlay=index;index+=1
    vf='[base]'
    if scene.get('cameraFilter'):
        vf+=scene['cameraFilter']+','
    if scene['crop']:
        c=scene['crop'];vf+=f"crop={c['w']}:{c['h']}:{c['x']}:{c['y']},"
        if scene.get('cardFrame'):
            height=round(c['h']*1820/c['w']/2)*2
            vf+=f"scale=1820:{height}:flags=lanczos,pad=1920:1080:50:300:color=0xf1f5fb,"
        else:
            vf+='scale=1920:1080:flags=lanczos,'
    if scene['zoom']:
        vf+="zoompan=z='1+0.025*min(on/36,1)':x='(iw-iw/zoom)/2':y='(ih-ih/zoom)/2':d=1:s=1920x1080:fps=30,"
    vf+='setsar=1[v0]';filters.append(vf)
    caption_filter=''
    if plan.get('subtitles',True):caption_filter=",ass=filename='"+subtitles(scene).name+"'"
    filters.append(f"[v0][{overlay}:v]overlay=0:0{caption_filter},format=yuv420p[v]")
    video_output=directory/(scene['id']+'.video.mp4')
    args+=['-filter_complex',';'.join(filters),'-map','[v]','-an','-t',str(scene['duration']),'-r','30','-c:v','libx264','-preset','veryfast','-crf','18','-movflags','+faststart',str(video_output)]
    # ASS paths are relative, avoiding Windows drive-letter filter escaping.
    old=os.getcwd();os.chdir(directory)
    try:run(args,directory/(scene['id']+'.log'))
    finally:os.chdir(old)
    if plan.get('narrationTracks'):
        continue
    # Keep video, speech mixing and encoding independently diagnosable. The
    # installed native AAC configuration stalled at 192k mono; 128k/fast was
    # verified on the same input before proceeding with the final render.
    audio_args=[];audio_filters=[];audio_ids=[]
    for index,audio in enumerate(scene['audio']):
        audio_args+=['-i',audio['audio']];delay=round(audio['offset']*1000)
        audio_filters.append(f"[{index}:a]adelay={delay}:all=1,apad=whole_dur={scene['duration']}[a{index}]");audio_ids.append(f'[a{index}]')
    audio_filters.append(''.join(audio_ids)+f"amix=inputs={len(audio_ids)}:normalize=0,atrim=duration={scene['duration']},aresample=48000[a]")
    audio_output=directory/(scene['id']+'.audio.wav')
    run(audio_args+['-filter_complex',';'.join(audio_filters),'-map','[a]','-c:a','pcm_s16le',str(audio_output)],directory/(scene['id']+'.audio.log'))
    run(['-i',str(video_output),'-i',str(audio_output),'-map','0:v','-map','1:a','-t',str(scene['duration']),'-c:v','copy','-c:a','aac','-b:a','128k','-aac_coder','fast','-movflags','+faststart',str(directory/(scene['id']+'.mp4'))],directory/(scene['id']+'.mux.log'))

if a.scene:
    print('SCENE_READY '+a.scene,flush=True)
    raise SystemExit(0)

print('ASSEMBLE',flush=True)
args=[];filters=[]
for scene in plan['scenes']:args+=['-i',str(directory/(scene['id']+'.video.mp4'))]
for i in range(len(plan['scenes'])):filters.append(f'[{i}:v]settb=AVTB,format=yuv420p[v{i}]')
prev='v0';offset=plan['scenes'][0]['duration']
for i in range(1,len(plan['scenes'])):
    boundary=plan.get('transitions',[{'duration':plan['transitionSeconds'],'type':'pixelize'}]*(len(plan['scenes'])-1))[i-1]
    transition=boundary['duration'];offset-=transition
    filters.append(f"[{prev}][v{i}]xfade=transition={boundary['type']}:duration={transition}:offset={offset}[x{i}]")
    prev=f'x{i}';offset+=plan['scenes'][i]['duration']
args+=['-filter_complex',';'.join(filters),'-map',f'[{prev}]','-an','-r','30','-c:v','libx264','-preset','veryfast','-crf','18','-pix_fmt','yuv420p','-movflags','+faststart',str(directory/'assembled.video.mp4')]
run(args,directory/'assemble.log')
audio_args=[];audio_filters=[];previous='0:a'
if plan.get('narrationTracks'):
    tracks=plan['narrationTracks']
    for i,track in enumerate(tracks):
        audio_args+=['-i',track['audio']]
        audio_filters.append(f"[{i}:a]adelay={round(track['offset']*1000)}:all=1,apad=whole_dur={plan['duration']}[n{i}]")
    audio_filters.append(''.join(f'[n{i}]' for i in range(len(tracks)))+f"amix=inputs={len(tracks)}:normalize=0,atrim=duration={plan['duration']}[speech]")
    previous='speech'
else:
    for scene in plan['scenes']:audio_args+=['-i',str(directory/(scene['id']+'.audio.wav'))]
    for i in range(1,len(plan['scenes'])):
        transition=plan['transitions'][i-1]['duration'] if plan.get('transitions') else plan['transitionSeconds']
        audio_filters.append(f"[{previous}][{i}:a]acrossfade=d={transition}:c1=tri:c2=tri[ax{i}]");previous=f'ax{i}'
audio_filters.append(f"[{previous}]loudnorm=I=-16:TP=-1.5:LRA={11 if plan.get('narrationTracks') else 7},aresample=48000[finalaudio]")
run(audio_args+['-filter_complex',';'.join(audio_filters),'-map','[finalaudio]','-c:a','pcm_s16le',str(directory/'assembled.audio.wav')],directory/'assemble.audio.log')
run(['-i',str(directory/'assembled.video.mp4'),'-i',str(directory/'assembled.audio.wav'),'-map','0:v','-map','1:a','-c:v','copy','-c:a','aac','-b:a','128k','-aac_coder','fast','-movflags','+faststart',str(a.output/'fastnet-home-card-zh-CN.mp4')],directory/'assemble.mux.log')
if plan.get('opening'):
    preview_name=plan.get('previewFile','fastnet-voice-v3-preview.mp4' if plan.get('revision')=='voice-v3' else 'fastnet-opening-v2-preview.mp4')
    run(['-i',str(a.output/'fastnet-home-card-zh-CN.mp4'),'-t',str(plan['opening']['duration']),'-c:v','libx264','-preset','veryfast','-crf','18','-c:a','aac','-b:a','128k','-aac_coder','fast','-movflags','+faststart',str(a.output/preview_name)],directory/'preview.log')
if plan.get('endingFile'):
    run(['-ss',str(plan['endingPreview']['start']),'-i',str(a.output/'fastnet-home-card-zh-CN.mp4'),'-t',str(plan['endingPreview']['duration']),'-c:v','libx264','-preset','veryfast','-crf','18','-c:a','aac','-b:a','128k','-aac_coder','fast','-movflags','+faststart',str(a.output/plan['endingFile'])],directory/'ending-preview.log')
print('MP4_READY',flush=True)
