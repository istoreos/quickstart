"""Cache word timing from the existing local Speech Studio Whisper service.

This produces private editing metadata, not subtitles. No model installation or
engine selection changes are performed.
"""
import hashlib,json,os,urllib.request,uuid
from pathlib import Path

root=Path(os.environ['TUTORIAL_OUTPUT'])
directory=root/'alignment';directory.mkdir(parents=True,exist_ok=True)
audio=json.loads((root/'audio/manifest.json').read_text(encoding='utf-8'))
opener=urllib.request.build_opener(urllib.request.ProxyHandler({}))
endpoint=os.environ['TUTORIAL_TTS_ENDPOINT'].removesuffix('/speech')+'/transcriptions'
manifest={}
for item in audio:
    target=directory/(item['id']+'.json')
    cached=json.loads(target.read_text(encoding='utf-8')) if target.exists() else None
    if not cached or cached.get('audioSha256')!=item['audioSha256']:
        boundary='tutorial-'+uuid.uuid4().hex;parts=[]
        for key,value in [('model','whisper-1'),('response_format','verbose_json'),('language','zh'),('timestamp_granularities[]','word')]:
            parts.append((f'--{boundary}\r\nContent-Disposition: form-data; name="{key}"\r\n\r\n{value}\r\n').encode())
        parts.append((f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="speech.wav"\r\nContent-Type: audio/wav\r\n\r\n').encode()+Path(item['audio']).read_bytes()+b'\r\n')
        parts.append((f'--{boundary}--\r\n').encode())
        req=urllib.request.Request(endpoint,data=b''.join(parts),headers={'Content-Type':'multipart/form-data; boundary='+boundary})
        with opener.open(req,timeout=180) as response:cached=json.loads(response.read())
        assert cached.get('words'),'Speech Studio must return real word timestamps'
        cached['audioSha256']=item['audioSha256']
        target.write_text(json.dumps(cached,ensure_ascii=False,indent=2),encoding='utf-8')
    manifest[item['id']]=cached
    print(json.dumps({'id':item['id'],'duration':cached['duration'],'wordCount':len(cached['words'])}),flush=True)
(directory/'manifest.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2),encoding='utf-8')
