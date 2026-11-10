#!/usr/bin/env python3
"""Audit evidence references and frozen requirement coverage, without running devices."""
import argparse, hashlib, json, re
from pathlib import Path
p=argparse.ArgumentParser()
p.add_argument('--evidence',type=Path,required=True)
p.add_argument('--output',type=Path,required=True)
a=p.parse_args();root=Path(__file__).resolve().parents[2]
plan=(root/'docs/archive/product/fastnet-card-implementation-milestones.md').read_text()
required=set(re.findall(r'\*\*(FN\d{2}-T\d{2})\*\*',plan));cases={};reports={}
for stage in range(1,13):
 name=f'FN{stage:02d}';file=a.evidence/name/'report.json';report=json.loads(file.read_text());assert report['status']=='passed',(name,report['status']);assert not report.get('unresolved'),name
 reports[name]={'path':str(file),'sha256':hashlib.sha256(file.read_bytes()).hexdigest()}
 for case in report['cases']:
  assert case['id'] not in cases and case['status']=='passed',case
  assert case.get('actual') and case.get('evidence'),case['id']
  for ref in case['evidence']:assert (file.parent/ref).exists(),(case['id'],ref)
  cases[case['id']]=case
assert set(cases)==required,{'missing':sorted(required-set(cases)),'unknown':sorted(set(cases)-required)}
matrix=plan.split('## 需求覆盖矩阵')[1].split('## 实施前')[0]
requirements={}
for line in matrix.splitlines():
 if not re.match(r'^\| R\d{2} \|',line):continue
 cells=[x.strip() for x in line.split('|')];ids=re.findall(r'FN\d{2}-T\d{2}',line)
 requirements[cells[1]]={'status':'passed','description':cells[2],'cases':ids}
assert len(requirements)==30
themes=json.loads((a.evidence/'FN12/theme-coverage.json').read_text());assert set(themes)=={f'TH{i:02d}' for i in range(1,13)}
budgets=json.loads((a.evidence/'FN12/final-budgets.json').read_text());assert set(budgets['results'])=={f'B{i:02d}' for i in range(1,11)}
for group in [themes,budgets['results']]:
 for key,value in group.items():
  assert value['status']=='passed' and value.get('evidence'),key
  for ref in value['evidence']:assert (a.evidence/ref).exists(),(key,ref)
result={'status':'passed','milestones':12,'cases':len(cases),'requirements':requirements,'themes':themes,'budgets':budgets['results'],'reports':reports}
a.output.parent.mkdir(parents=True,exist_ok=True);a.output.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({'status':'passed','milestones':12,'cases':len(cases),'requirements':30,'themes':12,'budgets':10}))
