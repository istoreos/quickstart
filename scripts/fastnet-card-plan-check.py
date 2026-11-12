#!/usr/bin/env python3
"""Check the frozen acceptance graph, not milestone implementation success."""
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
plan = (ROOT / 'docs/archive/product/fastnet-card-implementation-milestones.md').read_text()
stages = re.findall(r'^## (FN\d{2}) ', plan, re.M)
targets = re.findall(r'\*\*(FN\d{2}-T\d{2})\*\*', plan)
matrix = plan.split('## 需求覆盖矩阵')[1].split('## 实施前')[0]
refs = set(re.findall(r'FN\d{2}-T\d{2}', matrix))
assert len(stages) == len(set(stages)) == 12
assert len(targets) == len(set(targets)) == 76
assert refs == set(targets), {'missing': set(targets)-refs, 'unknown': refs-set(targets)}
assert len(re.findall(r'^\| R\d{2} \|', matrix, re.M)) == 30
assert len(re.findall(r'^\| TH\d{2} \|', plan, re.M)) == 12
assert len(re.findall(r'^\| B\d{2} \|', plan, re.M)) == 10
fixture = json.loads((ROOT / 'docs/contracts/fastnet-card.json').read_text())
assert fixture['catalog']['id'] == fixture['manifest']['id'] == 'plugin.fastnet'
assert fixture['sdk']['url'].endswith('/card-runtime-v1.js')
print(json.dumps({'status':'passed','milestones':12,'targets':76,'requirements':30,'themes':12,'budgets':10,'allTargetsMapped':True}))
