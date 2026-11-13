"""Voice-only auditions, preserving the approved A-style API parameters."""
import hashlib,json,os,urllib.request,wave
from pathlib import Path

root=Path(os.environ['TUTORIAL_OUTPUT']);directory=root/'samples';directory.mkdir(parents=True,exist_ok=True)
spec=json.loads(Path('home-fastnet-card-voice-v3.json').read_text(encoding='utf-8'))
preset=dict(spec['voice']);endpoint=os.environ[preset.pop('endpointEnv')]
text='iStoreOS 首页，现在也能看测速成绩了。用的就是刚更新的 FastNet。点一下，测测路由器的外网下载。'
variants=[('02-natural-sentences',text),('03-unhurried-native','<|prosody:speed_slow|>'+text)]
report=[]
for name,input_text in variants:
    body={**preset,'input':input_text};target=directory/(name+'.wav')
    request_file=directory/(name+'.request.json')
    encoded=json.dumps(body,ensure_ascii=False).encode('utf-8')
    if target.exists() and (not request_file.exists() or request_file.read_bytes()!=encoded):
        raise RuntimeError('Audition ID already exists with a different request; choose a new ID')
    if not target.exists():
        req=urllib.request.Request(endpoint,data=encoded,headers={'Content-Type':'application/json; charset=utf-8'})
        with urllib.request.urlopen(req,timeout=300) as response:audio=response.read()
        if audio[:4]!=b'RIFF':raise RuntimeError('Expected WAV')
        target.write_bytes(audio)
    request_file.write_bytes(encoded)
    with wave.open(str(target)) as audio:duration=audio.getnframes()/audio.getframerate()
    report.append({'id':name,'duration':duration,'sha256':hashlib.sha256(target.read_bytes()).hexdigest(),
                   'speed':body['speed'],'postTempo':1.0,'inputControl':'native Higgs slow prosody tag' if name.startswith('03') else 'sentence punctuation only'})
    print(json.dumps(report[-1]),flush=True)
(directory/'auditions.json').write_text(json.dumps(report,ensure_ascii=False,indent=2),encoding='utf-8')
