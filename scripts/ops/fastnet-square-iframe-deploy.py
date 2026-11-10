#!/usr/bin/env python3
"""Retain an explicitly scoped UI candidate on the existing .7 test appliance.

No opkg install, UCI writes, homepage reset, service enablement or automatic test.
The old version stays in a private rollback archive; package records are retained.
"""
import argparse,datetime,gzip,hashlib,json,re,shlex,shutil,subprocess,tarfile
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--web',type=Path,required=True);p.add_argument('--candidate',type=Path,required=True);p.add_argument('--output',type=Path,required=True);a=p.parse_args()
a.output.mkdir(parents=True,exist_ok=False)
root=Path(__file__).resolve().parents[2];hub=Path('/projects/workspace-linkease-ubuntu/openwrt-apps/istoreos-app-hub/apps/fastnet')
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest();q=shlex.quote
ssh=['ssh','-o','BatchMode=yes','-o','ConnectTimeout=5','root@192.168.30.7']
def remote(command,data=None,timeout=60):
 r=subprocess.run(ssh+[command],input=data,capture_output=True,timeout=timeout)
 if r.returncode:raise RuntimeError(r.stderr.decode(errors='replace')[-1200:])
 return r.stdout
protected=['/etc/config/network','/etc/config/dhcp','/etc/config/firewall','/etc/config/fastnet','/etc/quickstart/home-layout.json','/etc/quickstart/home-launcher.json','/usr/lib/opkg/status','/usr/sbin/quickstart']
def snapshot():return {name:remote('if test -f '+q(name)+'; then sha256sum '+q(name)+'; else echo absent; fi').decode().split()[0] for name in protected}
record=json.loads((a.candidate/'candidate.json').read_text())
for item in record['packages']:assert sha(Path(item['path']))==item['sha256']
manifest=json.loads((hub/'luci-app-fastnet/root/usr/share/fastnet/quickstart/fastnet.json').read_text())
release='fastnet-square-'+datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ')
stage='/tmp/'+release;backup='/root/quickstart-backups/'+release
report={'status':'in_progress','host':'root@192.168.30.7','url':'http://192.168.30.7:10000/cgi-bin/luci/admin/quickstart','release':release,'privateBackup':backup,'scope':'Quickstart web/template; FastNet core embedded frontend; FastNet controller and card assets; local test feed','packageRecordsChanged':False}
changed=False
try:
 report['before']=snapshot()
 summary=json.loads(remote('curl --fail -s --max-time 8 --unix-socket /var/run/fastnet-card.sock http://localhost/api/card/summary'));assert summary['task'] is None,'An active WAN test must finish before deployment'
 service=json.loads(remote('ubus -S call service list \'{"name":"fastnet"}\''));running=any(i.get('running') for s in service.values() for i in s.get('instances',{}).values());assert running,'Expected the existing enabled FastNet service'
 original=remote('cat /usr/lib/lua/luci/view/quickstart/main.htm').decode();template=re.sub(r'local asset_version = "[^"]+"','local asset_version = "'+release+'"',original);assert template!=original
 payload=a.output/'payload';(payload/'usr/sbin').mkdir(parents=True)
 shutil.copy2(a.candidate/'payload/x86_64/fastnet/usr/sbin/FastNet',payload/'usr/sbin/FastNet')
 shutil.copytree(a.web/'luci-static/quickstart',payload/'www/luci-static/quickstart')
 luci=hub/'luci-app-fastnet';controller=Path('usr/lib/lua/luci/controller/fastnet.lua');(payload/controller.parent).mkdir(parents=True,exist_ok=True);shutil.copy2(luci/'luasrc/controller/fastnet.lua',payload/controller)
 shutil.copytree(luci/'root/www/luci-static/fastnet/quickstart',payload/'www/luci-static/fastnet/quickstart')
 plugin=payload/'usr/share/fastnet/fastnet-plugin.json';plugin.parent.mkdir(parents=True,exist_ok=True);shutil.copy2(hub/'fastnet/files/fastnet-plugin.json',plugin)
 for path in ['usr/share/fastnet/quickstart/fastnet.json','usr/share/quickstart/cards.d/fastnet.json']:
  target=payload/path;target.parent.mkdir(parents=True,exist_ok=True);target.write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
 template_path=payload/'usr/lib/lua/luci/view/quickstart/main.htm';template_path.parent.mkdir(parents=True,exist_ok=True);template_path.write_text(template)
 feed=payload/'www/quickstart-fastnet-test';feed.mkdir(parents=True)
 entries=[]
 for item in record['packages']:
  if item['arch'] not in ['all','x86_64']:continue
  f=Path(item['path']);shutil.copy2(f,feed/f.name)
  entries.append('Package: '+item['name']+'\nVersion: '+item['version']+'\nArchitecture: '+item['arch']+'\nDepends: '+item['depends']+'\nFilename: '+f.name+'\nSize: '+str(f.stat().st_size)+'\nSHA256sum: '+sha(f)+'\nDescription: FastNet local test candidate\n')
 index='\n'.join(entries)+'\n';(feed/'Packages').write_text(index);(feed/'Packages.gz').write_bytes(gzip.compress(index.encode(),mtime=0))
 archive=a.output/'payload.tar.gz'
 with tarfile.open(archive,'w:gz') as tar:
  for file in sorted(payload.rglob('*')):
   if file.is_file():
    info=tar.gettarinfo(str(file),str(file.relative_to(payload)));info.uid=info.gid=0;info.uname=info.gname='root';tar.addfile(info,file.open('rb'))
 paths=['usr/sbin/FastNet','usr/share/fastnet/fastnet-plugin.json','www/luci-static/quickstart','www/luci-static/fastnet/quickstart','usr/lib/lua/luci/controller/fastnet.lua','usr/lib/lua/luci/view/quickstart/main.htm','usr/share/fastnet/quickstart/fastnet.json','usr/share/quickstart/cards.d/fastnet.json','www/quickstart-fastnet-test']
 remote('set -eu; umask 077; mkdir '+q(stage)+'; mkdir '+q(backup)+'; tar -czf '+q(backup+'/before.tar.gz')+' -C / '+' '.join(map(q,paths)))
 remote('cat > '+q(stage+'/payload.tar.gz'),archive.read_bytes())
 rollback='#!/bin/sh\nset -eu\n/etc/init.d/fastnet stop\nrm -rf /www/luci-static/quickstart /www/luci-static/fastnet/quickstart /www/quickstart-fastnet-test\ntar -xzf "$(dirname "$0")/before.tar.gz" -C /\nrm -f /tmp/luci-indexcache /tmp/luci-indexcache.*\nrm -rf /tmp/luci-modulecache/\n/etc/init.d/fastnet start\nsync\n'
 remote('cat > '+q(backup+'/rollback.sh'),rollback.encode());remote('chmod 700 '+q(backup+'/rollback.sh'))
 # Check the shared task slot again immediately before the one service restart.
 summary=json.loads(remote('curl --fail -s --max-time 8 --unix-socket /var/run/fastnet-card.sock http://localhost/api/card/summary'));assert summary['task'] is None
 changed=True
 remote('set -eu\n/etc/init.d/fastnet stop\nmkdir '+q(stage+'/unpacked')+'\ntar -xzf '+q(stage+'/payload.tar.gz')+' -C '+q(stage+'/unpacked')+'\nrm -rf /www/luci-static/quickstart\ncp -a '+q(stage+'/unpacked/.')+' /\nchmod 755 /usr/sbin/FastNet\nrm -f /tmp/luci-indexcache /tmp/luci-indexcache.*\nrm -rf /tmp/luci-modulecache/\n/etc/init.d/fastnet start\nsync')
 report['after']=snapshot();assert report['after']==report['before'],'A protected layout, configuration, package record or backend changed'
 expected={str(f.relative_to(payload)):sha(f) for f in payload.rglob('*') if f.is_file()}
 for name,digest in expected.items():assert remote('sha256sum '+q('/'+name)).decode().split()[0]==digest,name
 report['files']=expected;report['cardEntry']=manifest['entry'];report['status']='deployed';print('Scoped test candidate retained; verifying local package index',flush=True)
 update=remote('is-opkg update',timeout=180);(a.output/'feed-update.log').write_bytes(update)
 assert remote('sha256sum /tmp/is-root/tmp/opkg-lists/quickstart_fastnet_test').decode().split()[0]==sha(feed/'Packages')
 report['feedUpdated']=True
 report['health']=json.loads(remote('curl --fail -s --max-time 8 http://127.0.0.1:3200/api/health'));assert report['health']['ok'] is True
 report['status']='deployed_and_healthy'
except Exception as e:
 report['status']='failed';report['failure']=str(e)
 if changed:
  remote('sh '+q(backup+'/rollback.sh'));report['rolledBack']=True
 raise
finally:
 remote('rm -rf '+q(stage))
 (a.output/'deployment.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({'status':report['status'],'release':release,'privateBackup':backup}))
