#!/usr/bin/env python3
"""Bounded, persistent media validation. Never starts a fourth review round."""
import argparse,hashlib,json,os,re,subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--round',type=int,required=True);p.add_argument('--visual-pass',action='store_true');p.add_argument('--visual-fail');a=p.parse_args()
root=Path(os.environ['TUTORIAL_OUTPUT']);history_path=root/'review-history.json'
history=json.loads(history_path.read_text(encoding='utf-8-sig')) if history_path.exists() else []
if a.visual_pass or a.visual_fail:
    assert history and history[-1]['round']==a.round
    if a.visual_pass:assert history[-1]['status']=='passed','Automatic checks must pass before visual approval'
    history[-1]['visual']={'passed':bool(a.visual_pass),'reviewer':'Codex image inspection','findings':a.visual_fail or 'Chapter frames, action targets, subtitles, results and privacy regions inspected.'}
    history_path.write_text(json.dumps(history,ensure_ascii=False,indent=2),encoding='utf-8')
    print(json.dumps(history[-1],ensure_ascii=False));raise SystemExit(0)
assert 1<=a.round<=3 and a.round==len(history)+1,'Review limit or order violation'
movie=root/'fastnet-home-card-zh-CN.mp4';plan=json.loads((root/'edit-plan.json').read_text(encoding='utf-8'))
entry={'round':a.round,'status':'running','videoSha256':hashlib.sha256(movie.read_bytes()).hexdigest(),'checks':[]}
history.append(entry);history_path.write_text(json.dumps(history,indent=2),encoding='utf-8')
ffmpeg=os.environ.get('TUTORIAL_FFMPEG','ffmpeg');ffprobe=os.environ.get('TUTORIAL_FFPROBE','ffprobe')
def check(name,condition,detail=None):
    entry['checks'].append({'id':name,'passed':bool(condition),'detail':detail})
    if not condition:raise AssertionError(name+': '+str(detail))
try:
    meta=json.loads(subprocess.check_output([ffprobe,'-v','error','-show_streams','-show_format','-of','json',str(movie)],text=True))
    video=next(s for s in meta['streams'] if s['codec_type']=='video');audio=next(s for s in meta['streams'] if s['codec_type']=='audio')
    check('1080p_h264',video['width']==1920 and video['height']==1080 and video['codec_name']=='h264')
    check('fixed_30fps',video['r_frame_rate']=='30/1' and video['avg_frame_rate']=='30/1',video['avg_frame_rate'])
    duration=float(meta['format']['duration']);check('90_to_120_seconds',90<=duration<=120,duration)
    check('duration_matches_plan',abs(duration-plan['duration'])<.15,{'actual':duration,'plan':plan['duration']})
    check('aac_48khz',audio['codec_name']=='aac' and audio['sample_rate']=='48000')
    decoded=subprocess.run([ffmpeg,'-v','error','-i',str(movie),'-f','null','NUL'],capture_output=True,text=True)
    check('entire_mp4_decodes',decoded.returncode==0 and not decoded.stderr.strip(),decoded.stderr[:500])
    srt=(root/'fastnet-home-card-zh-CN.srt').read_text(encoding='utf-8')
    text=''.join(line for line in srt.splitlines() if line and not line.isdigit() and '-->' not in line)
    expected=''.join(a['text'] for scene in plan['scenes'] for a in scene['audio'])
    check('subtitle_text_exact',text==expected)
    for scene in plan['scenes']:
        for cue in scene['cues']:check('cue_'+cue['target'],abs(cue['errorMs'])<=300,cue['errorMs'])
        for clip in scene['clips']:check('real_source_'+scene['id'],Path(clip['file']).is_file())
    for name in ['measure','home','beta','install','installed']:
        capture=json.loads((root/f'capture/capture-{name}.json').read_text(encoding='utf-8'))
        check('browser_'+name,capture['status']=='passed' and not capture['errors'])
        if name=='home':check('layout_restored',capture['restored'] is True)
        if name=='install':check('actual_update',capture['install']['before']=='v0.7.7' and capture['install']['observedVersion']=='0.8.9')
    if plan.get('revision')=='opening-v2':
        opening=json.loads((root/'capture/capture-opening.json').read_text(encoding='utf-8'))
        check('opening_browser_and_restore',opening['status']=='passed' and not opening['errors'] and opening['restored'])
        check('opening_20_to_25_seconds',20<=plan['opening']['duration']<=25,plan['opening']['duration'])
        check('opening_story_order',[s['id'] for s in plan['scenes'][:6]]==['opening-home','opening-enter','opening-test','opening-result','opening-return','opening-next'])
        first=plan['scenes'][0];check('homepage_full_before_zoom',first['camera']['from']=={'x':0,'y':0,'w':1920,'h':1080} and first['camera']['hold']>=.7)
        check('result_readable_hold',plan['scenes'][3]['duration']-plan['scenes'][3]['camera']['seconds']>=2)
        parent=Path(os.environ['TUTORIAL_PARENT_OUTPUT'])
        check('original_video_preserved',hashlib.sha256((parent/'fastnet-home-card-zh-CN.mp4').read_bytes()).hexdigest()==plan['parentVideoSha256'])
        parent_plan=json.loads((root/'parent-edit-plan.json').read_text(encoding='utf-8'))
        for scene in (s for s in plan['scenes'] if s.get('reuseRendered')):
            original=next(s for s in parent_plan['scenes'] if s['id']==scene['id'])
            check('body_content_'+scene['id'],scene['audio']==original['audio'] and scene['clips']==original['clips'])
            for suffix in ['.video.mp4','.audio.wav']:
                check('body_hash_'+scene['id']+suffix,hashlib.sha256((root/'render'/(scene['id']+suffix)).read_bytes()).hexdigest()==hashlib.sha256((parent/'render'/(scene['id']+suffix)).read_bytes()).hexdigest())
        preview=root/'fastnet-opening-v2-preview.mp4'
        preview_meta=json.loads(subprocess.check_output([ffprobe,'-v','error','-show_format','-of','json',str(preview)],text=True))
        check('opening_preview_duration',abs(float(preview_meta['format']['duration'])-plan['opening']['duration'])<.15)
    measure=json.loads((root/'capture/measurement.json').read_text(encoding='utf-8'))
    check('real_download_saved',measure['after']['latestResults']['download']['value']>0)
    loud=subprocess.run([ffmpeg,'-hide_banner','-i',str(movie),'-af','loudnorm=I=-16:TP=-1.5:LRA=7:print_format=json','-f','null','NUL'],capture_output=True,text=True)
    matches=re.findall(r'\{\s*"input_i".*?\}',loud.stderr,re.S);assert matches,'No loudness statistics'
    levels=json.loads(matches[-1]);check('loudness_and_peak',-18<=float(levels['input_i'])<=-14 and float(levels['input_tp'])<=-1,levels)
    thumbs=root/f'review-round-{a.round}';thumbs.mkdir(exist_ok=True)
    for scene in plan['scenes']:
        times=[('middle',scene['start']+min(scene['duration']*.55,scene['duration']-1))]
        if scene['cues']:times.append(('click',scene['start']+scene['cues'][0]['at']))
        times.append(('end',scene['start']+scene['duration']-.8))
        if scene['id'].startswith('opening-'):times.append(('start',scene['start']+.12))
        for label,time in times:
            subprocess.run([ffmpeg,'-v','error','-y','-ss',str(time),'-i',str(movie),'-frames:v','1',str(thumbs/f'{scene["id"]}-{label}.png')],check=True)
    # Public receipts omit private paths, cookie-bearing Trace and network data.
    manifest={'tutorial':plan['tutorial'],'software':plan['software'],'duration':duration,'width':1920,'height':1080,'fps':30,'videoSha256':entry['videoSha256'],'subtitleSha256':hashlib.sha256((root/'fastnet-home-card-zh-CN.srt').read_bytes()).hexdigest(),'voice':'gange-natural','audio':[{'id':x['id'],'duration':x['duration'],'sha256':x['audioSha256']} for s in plan['scenes'] for x in s['audio']],'measurement':{'downloadMbps':measure['after']['latestResults']['download']['value']/1e6,'uploadMbps':measure['after']['latestResults']['upload']['value']/1e6},'checksPassed':len(entry['checks'])}
    if plan.get('revision'):
        manifest.update(revision=plan['revision'],parentVideoSha256=plan['parentVideoSha256'],openingDuration=plan['opening']['duration'],homepageDownloadDisplay='921 Mbps (rounded by existing UI)',previewSha256=hashlib.sha256((root/'fastnet-opening-v2-preview.mp4').read_bytes()).hexdigest())
    (root/'manifest.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2),encoding='utf-8')
    entry['status']='passed';entry['media']={'duration':duration,'video':video['codec_name'],'audio':audio['codec_name']}
except Exception as error:
    entry['status']='failed';entry['failure']=str(error)
finally:
    history_path.write_text(json.dumps(history,ensure_ascii=False,indent=2),encoding='utf-8')
print(json.dumps({'round':a.round,'status':entry['status'],'checks':len(entry['checks']),'failure':entry.get('failure')},ensure_ascii=False))
raise SystemExit(0 if entry['status']=='passed' else 1)
