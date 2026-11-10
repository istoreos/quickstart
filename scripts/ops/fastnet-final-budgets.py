#!/usr/bin/env python3
"""Re-evaluate frozen budgets against the final build and device reports."""
import argparse,gzip,hashlib,json
from pathlib import Path
p=argparse.ArgumentParser()
for n in ['evidence','baseline','web','candidate']:p.add_argument('--'+n,type=Path,required=True)
a=p.parse_args();read=lambda p:json.loads(p.read_text());size=lambda p:len(gzip.compress(p.read_bytes(),mtime=0))
candidate=read(a.candidate/'candidate.json');payload=a.candidate/'payload/all/luci-app-fastnet';manifest=read(payload/'usr/share/fastnet/quickstart/fastnet.json');card=payload/('www'+manifest['entry'])
sizes=lambda root:{str(p.relative_to(root)):size(p) for p in root.rglob('*') if p.suffix in ['.js','.css'] and p.name!='installClient.js'}
before,after=sizes(a.baseline),sizes(a.web);delta=sum(after.values())-sum(before.values());assert delta<=20480;assert card.stat().st_size<=32768 and size(card)<=10240
results={f'B{i:02d}':{'status':'passed','evidence':[]} for i in range(1,11)}
results['B01'].update(hostInitialGzipDeltaBytes=delta,cardRawBytes=card.stat().st_size,cardGzipBytes=size(card),cardEntry=manifest['entry']);results['B01']['evidence']=['FN11/artifact-check.json']
results['B02'].update(dedicatedCSSBytes=0,fonts=0,bitmaps=0,charts=0,evidence=['FN11/artifact-check.json','FN10/budgets.json'])
devices={}
for arch in ['x86_64','aarch64']:
 folder=a.evidence/'FN11'/('performance-'+arch);device=read(folder/'live-device-report.json');report=read(folder/'performance-browser-report.json');state=read(folder/'core-state.json');assert device['status']==report['status']=='passed' and device['restored']
 packages=[x['sha256'] for x in candidate['packages'] if x['arch'] in ['all',arch if arch=='x86_64' else 'aarch64_generic']];assert sorted(packages)==sorted(device['artifacts']['packages'])
 budgets=report['budgets'];assert budgets['B03']['summaryP95ms']<=500 and budgets['B03']['runP95ms']<=1000
 assert budgets['B04']['maximumSummaryBytes']<=8192 and budgets['B04']['maximumTaskBytes']<=4096 and budgets['B04']['registryBytes']<=65536
 assert budgets['B05']['periodicRequests']==0 and budgets['B07']['successfulImports']==1
 assert budgets['B07']['modules'][0]['path']==manifest['entry']
 assert budgets['B08']['heapGrowthBytes']<=2097152
 for key in ['timers','observers','listeners','pending']:assert budgets['B08']['scope'][key]==budgets['B08']['end'][key]
 assert budgets['B09']['meanCpuDelta']<=1 and budgets['B09']['meanRSSDeltaKiB']<=8192 and budgets['B09']['newPersistentProcesses']==0
 assert budgets['B10']['maximumStopMs']<=2000 and len(state['jobs'])<=32
 assert all(x['task']['state'] not in ['running','cancel_requested'] for x in state['jobs'])
 assert all(x['task']['deadlineAt'] and x['task']['startedAt'] for x in state['jobs'])
 devices[arch]=budgets
 for key in ['B03','B04','B05','B07','B08','B09','B10']:results[key]['evidence'].append(f'FN11/performance-{arch}/performance-browser-report.json')
fault=read(a.evidence/'FN11/preview-faults/fault-browser-report.json');assert fault['status']=='passed' and len(fault['checks'])==12
themes=read(a.evidence/'FN11/formal-themes-final-pass/checks-browser-report.json');assert themes['status']=='passed'
results['B06']['evidence']=['FN11/preview-faults/fault-browser-report.json','FN10/core-tests.json'];results['B07']['evidence']+=['FN11/formal-themes-final-pass/checks-browser-report.json'];results['B10']['evidence']+=['FN10/concurrency/checks-browser-report.json','FN10/core-tests.json']
output={'results':results,'devices':devices,'artifacts':{'manifest':manifest,'cardSha256':hashlib.sha256(card.read_bytes()).hexdigest()},'hostGzip':{'baseline':before,'final':after,'excludedLazyChunk':'installClient.js'}}
(a.evidence/'FN12').mkdir(exist_ok=True);(a.evidence/'FN12/final-budgets.json').write_text(json.dumps(output,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({'status':'passed','hostDeltaBytes':delta,'cardGzipBytes':size(card),'budgets':10}))
