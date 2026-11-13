#!/usr/bin/env python3
"""Explicit commands for the small browser-video production pipeline."""
import argparse,subprocess,sys
from pathlib import Path
from remote import powershell
from settings import load_settings

here=Path(__file__).resolve().parent
p=argparse.ArgumentParser()
p.add_argument('command',choices=['sync','prepare','bootstrap','preflight','tts','align','record','render','review','collect'])
p.add_argument('--mode',choices=['measure','home','beta','install','installed','opening'])
p.add_argument('--spec',default='home-fastnet-card.json')
p.add_argument('--delivery',default='delivery')
p.add_argument('--real-test',action='store_true')
p.add_argument('--scene')
p.add_argument('--assemble-only',action='store_true')
p.add_argument('--round',type=int)
p.add_argument('--visual-pass',action='store_true')
p.add_argument('--visual-fail')
a=p.parse_args();settings=load_settings();server=settings['videoServer']
if Path(a.spec).name!=a.spec or Path(a.delivery).name!=a.delivery:p.error('spec and delivery must be basenames')
spec_path=here.parents[1]/'docs/product/tutorials'/a.spec
def remote(script):
    prefix='$ErrorActionPreference="Stop"; '+('' if a.command=='sync' else 'Set-Location ($env:TUTORIAL_SERVER_ROOT+"/tool"); ')
    code=powershell(server,prefix+script).returncode
    if code:raise SystemExit(code)
if a.command=='sync':
    remote('New-Item -ItemType Directory -Force ($env:TUTORIAL_SERVER_ROOT+"/tool"),$env:TUTORIAL_OUTPUT | Out-Null')
    files=[str(x) for x in here.iterdir() if x.suffix in ['.mjs','.py','.ps1'] or x.name in ['package.json','package-lock.json']]
    files.append(str(spec_path))
    subprocess.run(['scp',*files,server+':'+settings['serverRoot']+'/tool/'],check=True)
elif a.command=='bootstrap':remote('& ./bootstrap.ps1; exit $LASTEXITCODE')
elif a.command=='prepare':
    import json
    initializer=json.loads(spec_path.read_text(encoding='utf-8')).get('prepareScript','prepare-opening.ps1')
    if Path(initializer).name!=initializer or not (here/initializer).is_file():p.error('invalid initializer')
    remote('& ./'+initializer)
elif a.command=='preflight':remote('& ./preflight.ps1; exit $LASTEXITCODE')
elif a.command=='tts':remote('python tts.py '+a.spec+'; exit $LASTEXITCODE')
elif a.command=='align':remote('python speech-align.py; exit $LASTEXITCODE')
elif a.command=='render':
    if a.scene and a.assemble_only:p.error('--scene and --assemble-only are exclusive')
    import json
    compiler=json.loads(spec_path.read_text(encoding='utf-8')).get('compiler','compile.mjs')
    if Path(compiler).name!=compiler or not (here/compiler).is_file():p.error('invalid compiler')
    compile_command='' if a.assemble_only else 'node '+compiler+' '+a.spec+'; if($LASTEXITCODE -ne 0){exit $LASTEXITCODE}; '
    flags=(' --scene '+a.scene) if a.scene else (' --assemble-only' if a.assemble_only else '')
    remote(compile_command+'python render.py'+flags+'; exit $LASTEXITCODE')
elif a.command=='record':
    if not a.mode:p.error('--mode required')
    if a.mode=='measure' and not a.real_test:p.error('real WAN measurement requires --real-test')
    install=a.mode in ['install','installed'];device=settings['installDevice' if install else 'homeDevice']
    if a.mode=='install':subprocess.run([sys.executable,str(here/'backup-device.py'),'--device',device['ssh']],check=True)
    subprocess.run([sys.executable,str(here/'device.py'),'capture.mjs','--mode',a.mode,'--device',device['ssh'],'--base',device['base']],check=True)
elif a.command=='review':
    if a.round is None:p.error('--round required')
    import json
    reviewer=json.loads(spec_path.read_text(encoding='utf-8')).get('reviewer','review.py')
    if Path(reviewer).name!=reviewer or not (here/reviewer).is_file():p.error('invalid reviewer')
    command='python '+reviewer+' --round '+str(a.round)
    if reviewer=='review-voice.py':command+=' --spec '+a.spec
    if a.visual_pass:command+=' --visual-pass'
    if a.visual_fail:command+=" --visual-fail '"+a.visual_fail.replace("'","''")+"'"
    remote(command+'; exit $LASTEXITCODE')
elif a.command=='collect':
    import json
    spec=json.loads(spec_path.read_text(encoding='utf-8'))
    output=here/'.local'/a.delivery;output.mkdir(parents=True,exist_ok=True)
    files=['fastnet-home-card-zh-CN.mp4','manifest.json','review-history.json']
    if spec.get('subtitles',True):files+=['fastnet-home-card-zh-CN.srt']
    if settings.get('parentOutput'):files+=[spec.get('previewFile','fastnet-opening-v2-preview.mp4'),'edit-plan.json']
    if spec.get('endingFile'):files+=[spec['endingFile']]
    for name in files:
        subprocess.run(['scp',server+':'+settings['output']+'/'+name,str(output/name)],check=True)
    if spec.get('subtitles') is False:
        subprocess.run(['scp',server+':'+settings['output']+'/render/assembled.audio.wav',str(output/'fastnet-style-b-narration.wav')],check=True)
    print(output)
