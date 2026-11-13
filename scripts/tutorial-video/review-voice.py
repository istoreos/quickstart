"""Bounded checks for original B-style narration and subtitle-free rendering."""
import argparse,hashlib,json,os,re,subprocess,wave
from pathlib import Path

p=argparse.ArgumentParser();p.add_argument('--round',type=int,required=True);p.add_argument('--visual-pass',action='store_true');p.add_argument('--visual-fail');p.add_argument('--spec',default='home-fastnet-card-voice-b-v4.json');a=p.parse_args()
root=Path(os.environ['TUTORIAL_OUTPUT']);history_file=root/'review-history.json'
read=lambda path:json.loads(Path(path).read_text(encoding='utf-8-sig'))
digest=lambda path:hashlib.sha256(Path(path).read_bytes()).hexdigest()
history=read(history_file) if history_file.exists() else []
if a.visual_pass or a.visual_fail:
    assert history and history[-1]['round']==a.round
    if a.visual_pass:assert history[-1]['status']=='passed'
    history[-1]['visual']={'passed':bool(a.visual_pass),'method':'keyframe and contact-sheet inspection','findings':a.visual_fail or 'Brand/context, click targets, real results, return framing and absence of burned captions inspected.'}
    history_file.write_text(json.dumps(history,ensure_ascii=False,indent=2),encoding='utf-8');print(json.dumps({'round':a.round,'visual':history[-1]['visual']['passed']}));raise SystemExit(0)
assert 1<=a.round<=3 and a.round==len(history)+1,'Review limit/order violation'
plan=read(root/'edit-plan.json');movie=root/'fastnet-home-card-zh-CN.mp4'
entry={'round':a.round,'status':'running','videoSha256':digest(movie),'checks':[]};history.append(entry)
history_file.write_text(json.dumps(history,indent=2),encoding='utf-8')
def check(name,condition,detail=None):
    entry['checks'].append({'id':name,'passed':bool(condition),'detail':detail})
    if not condition:raise AssertionError(name+': '+str(detail))
ffmpeg=os.environ['TUTORIAL_FFMPEG'];ffprobe=os.environ['TUTORIAL_FFPROBE']
def media(path):return json.loads(subprocess.check_output([ffprobe,'-v','error','-show_format','-show_streams','-of','json',str(path)],text=True))
try:
    meta=media(movie);video=next(s for s in meta['streams'] if s['codec_type']=='video');audio=next(s for s in meta['streams'] if s['codec_type']=='audio');duration=float(meta['format']['duration'])
    check('1080p_h264',video['width']==1920 and video['height']==1080 and video['codec_name']=='h264')
    check('fixed_30fps',video['r_frame_rate']==video['avg_frame_rate']=='30/1')
    check('aac48k',audio['codec_name']=='aac' and audio['sample_rate']=='48000')
    check('duration_matches_voice_plan',abs(duration-plan['duration'])<.12,{'actual':duration,'planned':plan['duration']})
    check('natural_duration_range',plan['durationRange'][0]<=duration<=plan['durationRange'][1],duration)
    check('no_subtitle_stream',len(meta['streams'])==2 and all(s['codec_type']!='subtitle' for s in meta['streams']))
    check('no_caption_files',not list(root.glob('*.srt')) and not list((root/'render').glob('*.ass')))
    commands=[read(x) for x in (root/'render').glob('*.args.json')]
    filters=[args[i+1] for args in commands for i,x in enumerate(args) if x=='-filter_complex']
    check('no_burned_caption_filter',filters and not any('ass=filename' in f or 'subtitles=' in f for f in filters))
    check('no_audio_tempo_or_pitch_change',not any('atempo=' in f or 'asetrate=' in f or 'rubberband=' in f for f in filters))
    decoded=subprocess.run([ffmpeg,'-v','error','-i',str(movie),'-f','null','NUL'],capture_output=True,text=True)
    check('whole_movie_decodes',decoded.returncode==0 and not decoded.stderr.strip(),decoded.stderr[:300])
    parent=Path(os.environ['TUTORIAL_PARENT_OUTPUT'])
    check('parent_video_preserved',digest(parent/'fastnet-home-card-zh-CN.mp4')==plan['parentVideoSha256'])
    spec=read(a.spec);expected={k:v for k,v in spec['voice'].items() if k!='endpointEnv'}
    check('approved_b_values',expected['seed']==76421 and expected['temperature']==.88 and expected['top_p']==.95 and expected['top_k']==80 and expected['speed']==1)
    python_version=subprocess.check_output([os.environ['TUTORIAL_SPEECH_PYTHON'],'--version'],text=True).strip()
    check('original_python31318',python_version=='Python 3.13.18',python_version)
    actual_voice=Path(os.environ['TUTORIAL_VOICE_REFERENCE'])
    check('reference_wav_preserved',digest(actual_voice)=='f10af4483eaf69f1fad558ba9854039466a135e3173e7beb58deac1eaca7068e')
    align=read(root/'alignment/manifest.json');last_end=0
    for track in plan['narrationTracks']:
        name=track['id'];path=Path(track['audio']);request=read(path.with_suffix('.request.json'));body=dict(request);spoken=body.pop('input')
        check('exact_b_request_'+name,body==expected)
        check('voice_audio_hash_'+name,digest(path)==track['audioSha256'])
        check('asr_hash_'+name,align[name]['audioSha256']==track['audioSha256'])
        driver=read(path.with_suffix('.driver.json'))
        check('original_driver_'+name,driver['profile']=='gange-natural-style-b' and driver['voice_state']=='existing' and driver['model']=='higgs')
        brand_edit=path.with_suffix('.brand-edit.json')
        if brand_edit.exists():
            edit=read(brand_edit);native=path.with_suffix('.native.wav')
            check('native_brand_driver_hash',digest(native)==driver['sha256']==edit['nativeSha256'])
            check('compact_brand_output_hash',digest(path)==edit['resultSha256'])
            with wave.open(str(native)) as f:rate=f.getframerate();pcm=f.readframes(f.getnframes())
            with wave.open(str(path)) as f:actual_pcm=f.readframes(f.getnframes())
            pieces=[];at=0
            for cut in edit['cuts']:
                begin=round(cut['startSeconds']*rate)*2;end=round(cut['endSeconds']*rate)*2
                pieces.append(pcm[at:begin]);at=end
                check('brand_cut_is_quiet',cut['peak']<=32767*10**(edit['thresholdDb']/20))
            pieces.append(pcm[at:]);check('spoken_pcm_samples_unchanged',b''.join(pieces)==actual_pcm)
            check('brand_internal_pause_short',edit['longestRemainingQuietSeconds']<=spec['pronunciationControl']['maxInternalPauseSeconds']+.005,edit['longestRemainingQuietSeconds'])
        else:check('native_audio_matches_driver_'+name,digest(path)==driver['sha256'])
        check('no_slow_tag_'+name,'<|prosody:' not in spoken)
        with wave.open(str(path)) as wav:
            check('wav24k_mono16_'+name,wav.getframerate()==24000 and wav.getnchannels()==1 and wav.getsampwidth()==2)
        check('narration_continuous_'+name,track['offset']>=last_end-.001 and track['offset']+track['duration']<=duration+.05)
        last_end=track['offset']+track['duration']
    check('configured_brand_input',spec['pronunciations']['iStoreOS'] in plan['narrationTracks'][0]['spokenText'])
    if spec.get('reuseParentScenes'):
        parent_audio={x['id']:x for x in read(parent/'audio/manifest.json')}
        for track in plan['narrationTracks']:
            if track['id'] in ['opening-intro','closing']:continue
            check('approved_voice_preserved_'+track['id'],track['audioSha256']==parent_audio[track['id']]['audioSha256'])
        closing=next(t for t in plan['narrationTracks'] if t['id']=='closing')
        requested=next(s for s in spec['scenes'] if s['id']=='closing')['narration']
        check('closing_requested_text',closing['text']==requested)
        check('closing_asr_identity','小宝' in align['closing']['text'] and '感谢' in align['closing']['text'])
    for scene in plan['scenes']:
        if scene.get('reuseRendered'):
            check('approved_video_preserved_'+scene['id'],scene.get('sourceHasSubtitles') is False and digest(root/'render'/(scene['id']+'.video.mp4'))==digest(parent/'render'/(scene['id']+'.video.mp4')))
        else:check('raw_render_'+scene['id'],all(Path(c['file']).is_file() for c in scene['clips']))
        check('retimed_frames_'+scene['id'],abs(sum(c['outputDuration'] for c in scene['clips'])-scene['duration'])<.04)
        for cue in scene['cues']:
            track=next(t for t in plan['narrationTracks'] if t['id']==cue['audioId'])
            relative_end=scene['start']+cue['wordEnd']-track['offset']
            check('word_anchor_'+cue['target'],any(abs(w['end']-relative_end)<.001 for w in align[track['id']]['words']))
            error=(cue['at']-cue['wordEnd'])*1000
            check('click_timing_'+cue['target'],0<=error<=300,round(error))
    for mode in ['measure','home','install','installed','beta','opening']:
        captured=read(root/'capture'/('capture-'+mode+'.json'))
        check('source_browser_'+mode,captured['status']=='passed' and not captured['errors'])
        if mode in ['home','opening']:check('source_layout_restored_'+mode,captured['restored'])
    check('real_result_retained',abs(read(root/'capture/measurement.json')['after']['latestResults']['download']['value']/1e6-921.366451)<.000001)
    result=next(s for s in plan['scenes'] if s['id']=='opening-result')
    check('result_hold_at_least_2s',result['duration']-result['camera']['seconds']>=2)
    preview=root/plan['previewFile'];check('preview_duration',abs(float(media(preview)['format']['duration'])-plan['opening']['duration'])<.15)
    if plan.get('endingFile'):check('ending_preview_duration',abs(float(media(root/plan['endingFile'])['format']['duration'])-plan['endingPreview']['duration'])<.15)
    loud=subprocess.run([ffmpeg,'-hide_banner','-i',str(movie),'-af','loudnorm=I=-16:TP=-1.5:LRA=11:print_format=json','-f','null','NUL'],capture_output=True,text=True)
    levels=json.loads(re.findall(r'\{\s*"input_i".*?\}',loud.stderr,re.S)[-1]);check('loudness_and_peak',-18<=float(levels['input_i'])<=-14 and float(levels['input_tp'])<=-1,levels)
    thumbs=root/f'review-round-{a.round}';thumbs.mkdir(exist_ok=True)
    for scene in plan['scenes']:
        shots=[('middle',scene['start']+scene['duration']*.55),('end',scene['start']+scene['duration']-.3)]
        if scene['id'].startswith('opening'):shots.append(('start',scene['start']+.15))
        for cue in scene['cues']:shots.append((cue['target'],scene['start']+cue['at']))
        for label,t in shots:subprocess.run([ffmpeg,'-v','error','-y','-ss',str(t),'-i',str(movie),'-frames:v','1',str(thumbs/(scene['id']+'-'+label+'.png'))],check=True)
    manifest={'tutorial':plan['tutorial'],'revision':plan['revision'],'software':plan['software'],'voiceStyle':'b','voicePreset':expected,
              'speechPython':python_version,'subtitles':False,'audioTempo':1.0,'duration':duration,'openingDuration':plan['opening']['duration'],
              'videoSha256':digest(movie),'previewSha256':digest(preview),'parentVideoSha256':plan['parentVideoSha256'],'checksPassed':len(entry['checks']),
              'audio':[{'id':t['id'],'duration':t['duration'],'sha256':t['audioSha256'],'requestSha256':t['requestSha256']} for t in plan['narrationTracks']],
              'pronunciations':plan['pronunciations'],'listeningReview':'User requested original B preset; new full-video listening feedback pending. ASR and parameter checks do not certify naturalness.'}
    if plan.get('closing'):manifest.update(closingText=plan['closing']['text'],endingPreviewSha256=digest(root/plan['endingFile']))
    (root/'manifest.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2),encoding='utf-8');entry['status']='passed'
except Exception as error:
    entry['status']='failed';entry['failure']=str(error)
finally:history_file.write_text(json.dumps(history,ensure_ascii=False,indent=2),encoding='utf-8')
print(json.dumps({'round':a.round,'status':entry['status'],'checks':len(entry['checks']),'failure':entry.get('failure')}));raise SystemExit(0 if entry['status']=='passed' else 1)
