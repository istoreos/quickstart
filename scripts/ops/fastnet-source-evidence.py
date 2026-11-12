#!/usr/bin/env python3
"""Fingerprint compiled Go inputs and owned frontends without including evidence recursively."""
import argparse, hashlib, json, os, subprocess
from pathlib import Path
p=argparse.ArgumentParser()
p.add_argument('--fastnet',type=Path,required=True)
p.add_argument('--packages',type=Path,required=True)
p.add_argument('--output',type=Path,required=True)
a=p.parse_args();quickstart=Path(__file__).resolve().parents[2]
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
def revision(root):return subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip()
def golist(root,cwd,target,arches):
 files={};modules={}
 for arch in arches:
  raw=subprocess.check_output(['go','list','-deps','-json',target],cwd=cwd,text=True,env={**os.environ,'CGO_ENABLED':'0','GOOS':'linux','GOARCH':arch})
  decoder=json.JSONDecoder();raw=raw.lstrip()
  while raw:
   item,end=decoder.raw_decode(raw);raw=raw[end:].lstrip();module=item.get('Module',{})
   if module.get('Path'):modules[module['Path']]={k:module.get(k) for k in ['Version','Sum','Replace'] if module.get(k)}
   directory=Path(item['Dir'])
   if not directory.is_relative_to(root):continue
   for key in ['GoFiles','CgoFiles','SFiles','EmbedFiles']:
    for filename in item.get(key,[]):
     path=directory/filename;files[str(path.relative_to(root))]=sha(path)
 for name in ['go.mod','go.sum']:
  path=cwd/name
  if path.exists():files[str(path.relative_to(root))]=sha(path)
 return {'compiledFiles':files,'modules':modules,'architectures':arches}
q=golist(quickstart,quickstart/'backend','./cmd/backend',['amd64','arm64'])
f=golist(a.fastnet,a.fastnet,'./cmd/fastnet',['amd64','arm64','arm'])
def owned(root,paths):
 files={}
 for path in paths:
  for file in path.rglob('*') if path.is_dir() else [path]:
   if file.is_file() and '__pycache__' not in str(file) and 'node_modules' not in str(file):files[str(file.relative_to(root))]=sha(file)
 return files
q['frontend']=owned(quickstart,[quickstart/'web/src',quickstart/'web/package.json',quickstart/'web/package-lock.json',quickstart/'web/vite.config.ts',quickstart/'web/cardRuntimePlugin.ts'])
f['card']=owned(a.fastnet,[a.fastnet/'quickstart-card',a.fastnet/'scripts/build-quickstart-card.sh',a.fastnet/'scripts/card-provenance.py'])
pkg=owned(a.packages,[a.packages/'apps/fastnet'])
report={'quickstart':{'baseCommit':revision(quickstart),**q},'fastnet':{'baseCommit':revision(a.fastnet),**f},'packages':{'baseCommit':revision(a.packages),'files':pkg}}
for v in report.values():v['sourceSha256']=hashlib.sha256(json.dumps(v,sort_keys=True).encode()).hexdigest()
a.output.parent.mkdir(parents=True,exist_ok=True);a.output.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({k:v['sourceSha256'] for k,v in report.items()}))
