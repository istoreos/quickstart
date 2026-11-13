#!/usr/bin/env python3
"""Segmented local speech generation; executed only on the video server."""
import argparse
import hashlib
import json
import os
import subprocess
import urllib.request
import wave
from pathlib import Path

p=argparse.ArgumentParser()
p.add_argument('tutorial',type=Path)
p.add_argument('--output',type=Path,default=Path(os.environ.get('TUTORIAL_OUTPUT','tutorial-output')))
a=p.parse_args()
spec=json.loads(a.tutorial.read_text(encoding='utf-8'))
voice=dict(spec['voice']);endpoint=os.environ.get(voice.pop('endpointEnv'),'http://127.0.0.1:8797/v1/audio/speech')
reference=Path(os.environ['TUTORIAL_VOICE_REFERENCE'])
voice_files={suffix:hashlib.sha256(reference.with_suffix(suffix).read_bytes()).hexdigest()
             for suffix in ['.wav','.txt','.json','.words.json'] if reference.with_suffix(suffix).is_file()}
fingerprint=voice_files['.wav']
directory=a.output/'audio';directory.mkdir(parents=True,exist_ok=True)
manifest=[]
segments=[]
for scene in spec['scenes']:
    if scene.get('segments'):
        segments.extend({'id':scene['id']+'-'+str(i+1).zfill(2),'narration':text} for i,text in enumerate(scene['segments']))
    else:
        segments.append(scene)
for scene in segments:
    spoken=scene['narration']
    for name,pronunciation in spec.get('pronunciations',{}).items():spoken=spoken.replace(name,pronunciation)
    request={**voice,'input':spoken}
    digest=hashlib.sha256(json.dumps([request,voice_files],ensure_ascii=False,sort_keys=True).encode()).hexdigest()
    target=directory/(scene['id']+'.wav');meta=target.with_suffix('.json')
    if not(target.exists() and meta.exists() and json.loads(meta.read_text())['key']==digest):
        body=json.dumps(request,ensure_ascii=False).encode('utf-8')
        print('TTS '+scene['id'],flush=True)
        if os.environ.get('TUTORIAL_SPEECH_DRIVER'):
            text_file=target.with_suffix('.input.txt');text_file.write_text(spoken,encoding='utf-8')
            command=[os.environ['TUTORIAL_SPEECH_PYTHON'],os.environ['TUTORIAL_SPEECH_DRIVER'],'--style',os.environ['TUTORIAL_SPEECH_STYLE'],
                     '--input-file',str(text_file),'--output',str(target),'--base-url',endpoint.removesuffix('/v1/audio/speech')]
            env={**os.environ,'PYTHONUTF8':'1','PYTHONIOENCODING':'utf-8'}
            dry=subprocess.run(command+['--dry-run'],capture_output=True,text=True,encoding='utf-8',env=env,check=True,timeout=30)
            actual=json.loads(dry.stdout)
            if actual['request']!=request:raise RuntimeError('Original driver preset differs from approved request: '+scene['id'])
            generated=subprocess.run(command+(['--overwrite'] if target.exists() else []),capture_output=True,text=True,encoding='utf-8',env=env,check=True,timeout=360)
            receipt=json.loads(generated.stdout)
            if receipt['voice_state']!='existing':raise RuntimeError('Expected existing voice')
            target.with_suffix('.driver.json').write_text(json.dumps(receipt,ensure_ascii=False,indent=2),encoding='utf-8')
            data=target.read_bytes()
        else:
            req=urllib.request.Request(endpoint,data=body,headers={'Content-Type':'application/json; charset=utf-8'})
            with urllib.request.urlopen(req,timeout=300) as r:data=r.read()
        if data[:4]!=b'RIFF':raise RuntimeError('TTS did not return WAV for '+scene['id'])
        temporary=target.with_suffix('.partial');temporary.write_bytes(data);temporary.replace(target)
        meta.write_text(json.dumps({'key':digest,'voiceSha256':fingerprint,'voiceFilesSha256':voice_files}),encoding='utf-8')
    target.with_suffix('.request.json').write_text(json.dumps(request,ensure_ascii=False,indent=2),encoding='utf-8')
    with wave.open(str(target)) as audio:
        duration=audio.getnframes()/audio.getframerate()
    manifest.append({'id':scene['id'],'audio':str(target),'audioSha256':hashlib.sha256(target.read_bytes()).hexdigest(),'duration':duration,
                     'requestSha256':hashlib.sha256(json.dumps(request,ensure_ascii=False,sort_keys=True).encode()).hexdigest(),
                     'voiceFilesSha256':voice_files,'text':scene['narration'],'spokenText':spoken})
    print(scene['id']+' '+str(round(duration,3))+'s',flush=True)
(directory/'manifest.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2),encoding='utf-8')
print('TTS_COMPLETE',flush=True)
