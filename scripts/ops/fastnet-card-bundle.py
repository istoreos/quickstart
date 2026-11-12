#!/usr/bin/env python3
"""Collect exact tested artifacts; does not build, install, or publish anything."""
import argparse, hashlib, json, shutil, tarfile
from pathlib import Path
p=argparse.ArgumentParser()
for name in ['candidate','backend','web','evidence','output']:p.add_argument('--'+name,type=Path,required=True)
a=p.parse_args();a.output.mkdir(parents=True,exist_ok=False)
sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
candidate=json.loads((a.candidate/'candidate.json').read_text())
for item in candidate['packages']:
 f=Path(item['path']);assert sha(f)==item['sha256'];dest=a.output/'ipk'/f.name;dest.parent.mkdir(exist_ok=True);shutil.copy2(f,dest)
runtime=Path(candidate['runtime']['path']);assert sha(runtime)==candidate['runtime']['sha256'];shutil.copy2(runtime,a.output/runtime.name)
for arch in ['x86_64','aarch64']:
 f=a.backend/('quickstart.'+arch);dest=a.output/'quickstart'/f.name;dest.parent.mkdir(exist_ok=True);shutil.copy2(f,dest)
shutil.copytree(a.web/'luci-static/quickstart',a.output/'quickstart/www/luci-static/quickstart')
# No test benchmark or preview resources may enter the production bundle.
assert not (a.output/'quickstart/www/luci-static/quickstart/bench').exists()
for name in ['source-inputs.json','coverage-audit.json','final-budgets.json','theme-coverage.json']:
 f=a.evidence/name;assert f.exists();shutil.copy2(f,a.output/name)
root=Path(__file__).resolve().parents[2]
for f in [root/'docs/fastnet-card-release.md',root/'docs/fastnet-card-contract.md',root/'docs/archive/product/fastnet-card-implementation-milestones.md']:
 shutil.copy2(f,a.output/f.name)
record={'scope':candidate['scope'],'source':json.loads((a.evidence/'source-inputs.json').read_text()),'candidateInputs':candidate,'files':{str(f.relative_to(a.output)):sha(f) for f in sorted(a.output.rglob('*')) if f.is_file()}}
(a.output/'artifacts.json').write_text(json.dumps(record,ensure_ascii=False,indent=2)+'\n')
(a.output/'SHA256SUMS').write_text(''.join(sha(f)+'  '+str(f.relative_to(a.output))+'\n' for f in sorted(a.output.rglob('*')) if f.is_file()))
archive=Path(str(a.output)+'.tar.gz')
with tarfile.open(archive,'w:gz') as tar:tar.add(a.output,arcname=a.output.name)
print(json.dumps({'directory':str(a.output),'archive':str(archive),'sha256':sha(archive)}))
