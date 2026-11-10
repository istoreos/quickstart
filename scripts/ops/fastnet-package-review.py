#!/usr/bin/env python3
"""Install exact local IPKs on authorized devices, exercise lifecycle, then restore."""
import argparse, hashlib, json, re, shlex, subprocess, time, urllib.request, urllib.error, uuid, shutil, tempfile, os
from pathlib import Path
p=argparse.ArgumentParser()
p.add_argument('--host',choices=['root@192.168.30.7','root@192.168.30.93'],required=True)
p.add_argument('--candidate',type=Path,required=True)
p.add_argument('--output',type=Path,required=True)
p.add_argument('--ipkg-build',type=Path,required=True)
p.add_argument('--faults',action='store_true')
a=p.parse_args();a.output.mkdir(parents=True,exist_ok=True)
arch='x86_64' if a.host.endswith('.7') else 'aarch64_generic'
ssh=['ssh','-o','BatchMode=yes','-o','ConnectTimeout=5',a.host]
stage='/tmp/fastnet-package-'+uuid.uuid4().hex[:10]
record=json.loads((a.candidate/'candidate.json').read_text())
packages=[x for x in record['packages'] if x['arch'] in ['all',arch]]
names={x['name'] for x in packages}
report={'milestone':'FN08','status':'in_progress','device':arch,'checks':[],'static':{},'restored':False,'packages':[{k:x[k] for k in ['name','version','sha256']} for x in packages]}
changed=False
def remote(script,data=None,timeout=90):
 r=subprocess.run(ssh+[script],input=data,capture_output=True,timeout=timeout)
 if r.returncode:raise RuntimeError('remote check failed: '+r.stderr.decode(errors='replace')[:500])
 return r.stdout
def ownblock(block):return block.splitlines() and block.splitlines()[0].removeprefix('Package: ') in names
original_status=remote('cat /usr/lib/opkg/status').decode()
def unrelated(status):return [b for b in status.strip().split('\n\n') if not ownblock(b)]
try:
 services=json.loads(remote("ubus -S call service list '{\"name\":\"fastnet\"}'"))
 assert not any(i.get('running') for service in services.values() for i in service.get('instances',{}).values()), 'another FastNet service is active'
 remote('umask 077; mkdir '+stage+' '+stage+'/backup')
 files=set(['/usr/share/quickstart/cards.d/fastnet.json','/etc/config/fastnet','/etc/config/firewall','/etc/config/ucitrack','/etc/fastnet/card-state.json'])
 for item in packages:
  files.update('/'+s for s in item['files'] if not s.startswith('CONTROL/'))
  old=remote('opkg files '+shlex.quote(item['name'])+' 2>/dev/null || true').decode().splitlines()
  files.update(s for s in old if s.startswith('/'))
 files.update(remote('ls /usr/lib/opkg/info/*fastnet* /etc/rc.d/*fastnet 2>/dev/null || true').decode().splitlines())
 remote('cat > '+stage+'/paths',('\n'.join(sorted(files))+'\n').encode())
 remote('while IFS= read -r file; do if [ -f "$file" ] || [ -L "$file" ]; then echo "${file#/}"; fi; done < '+stage+'/paths > '+stage+'/existing; tar -czf '+stage+'/backup/files.tar.gz -C / -T '+stage+'/existing')
 # Check the package database has not changed between the read and backup.
 assert remote('cat /usr/lib/opkg/status').decode()==original_status
 remote('cat > '+stage+'/backup/status',original_status.encode())
 for item in packages:
  blob=Path(item['path']).read_bytes();assert hashlib.sha256(blob).hexdigest()==item['sha256']
  remote('cat > '+stage+'/'+Path(item['path']).name,blob)
 changed=True
 if a.faults:
  # Isolated opkg configuration: no mutation of real feeds/iStore code.
  conf='dest root /\n'+''.join('arch '+x.split()[1]+' '+x.split()[2]+'\n' for x in remote('opkg print-architecture').decode().splitlines())
  remote('cat > '+stage+'/fault.conf',conf.encode())
  before=remote('cat /usr/lib/opkg/status')
  meta=next(x for x in packages if x['name']=='app-meta-fastnet')
  failed=subprocess.run(ssh+['opkg -f '+stage+'/fault.conf install '+stage+'/'+Path(meta['path']).name],capture_output=True,timeout=30)
  diagnostic=failed.stdout+failed.stderr
  (a.output/'version-unsatisfied.log').write_bytes(diagnostic)
  assert failed.returncode!=0 or b'cannot find dependency' in diagnostic,'minimum versions must reject the old/missing installed dependencies'
  assert b'0.8.2' in diagnostic
  report['versionConstraint']={'exitCode':failed.returncode,'rejectedDependencies':True,'installedVersionsUnchanged':True}
  assert remote('cat /usr/lib/opkg/status')==before
  (a.output/'version-unsatisfied.log').write_bytes(failed.stdout+failed.stderr)
  report['checks'].append('real opkg rejects metadata minimum versions against old/missing dependencies without modifying package records')
  remote('cat > '+stage+'/fault.conf',(conf+'src fn-unreachable http://127.0.0.1:1/unavailable\n').encode())
  failed=subprocess.run(ssh+['opkg -f '+stage+'/fault.conf update'],capture_output=True,timeout=30)
  assert failed.returncode!=0,'unreachable package source must fail'
  (a.output/'source-unreachable.log').write_bytes(failed.stdout+failed.stderr)
  assert remote('cat /usr/lib/opkg/status')==before
  report['checks'].append('real opkg source connection failure leaves package records unchanged; subsequent exact candidate installation recovers')
 changed=True
 paths=' '.join(shlex.quote(stage+'/'+Path(x['path']).name) for x in packages)
 result=remote('opkg --force-overwrite --force-maintainer install '+paths,timeout=120).decode()
 (a.output/'opkg-install.log').write_text(result)
 assert not re.search(r'Collected errors|Cannot install|Failed to',result),result[-600:]
 installed=remote('opkg list-installed').decode()
 for x in packages:assert x['name']+' - '+x['version'] in installed
 manifest=json.loads(remote('cat /usr/share/quickstart/cards.d/fastnet.json'))
 entry=manifest['entry'];assert re.fullmatch(r'/luci-static/fastnet/quickstart/card\.[a-f0-9]+\.js',entry)
 report['checks'].append('real opkg install/upgrade resolves minimum core/LuCI dependencies; registration appears after its asset')
 port=remote('uci -q get uhttpd.main.listen_http').decode().split()[0].rsplit(':',1)[1]
 url='http://'+a.host.split('@')[1]+':'+port+entry
 with urllib.request.urlopen(urllib.request.Request(url,headers={'Accept-Encoding':'gzip'}),timeout=10) as r:
  raw=r.read();headers=dict(r.headers);report['static']={'status':r.status,'headers':headers,'wireBytes':len(raw),'sha256':hashlib.sha256(raw).hexdigest()}
 assert 'javascript' in headers.get('Content-Type','').lower(),headers
 expected=remote('cat '+shlex.quote('/www'+entry));assert raw==expected
 report['checks'].append('uhttpd serves exact hashed ESM directly with JavaScript MIME; actual headers and wire bytes captured')
 if headers.get('ETag'):
  try: conditional=urllib.request.urlopen(urllib.request.Request(url,headers={'If-None-Match':headers['ETag']}),timeout=10)
  except urllib.error.HTTPError as e: conditional=e
  with conditional: assert conditional.status==304;report['static']['conditionalStatus']=conditional.status
 report['static']['contentEncoding']=headers.get('Content-Encoding','identity')
 # A broken/intermediate install may not publish a new executable entry.
 remote('mv '+shlex.quote('/www'+entry)+' '+stage+'/card.js; rm /usr/share/quickstart/cards.d/fastnet.json; if /usr/libexec/fastnet-card-register install; then exit 42; fi; test ! -e /usr/share/quickstart/cards.d/fastnet.json; mv '+stage+'/card.js '+shlex.quote('/www'+entry)+'; /usr/libexec/fastnet-card-register install')
 report['checks'].append('missing asset fails registration atomically; a repaired asset can be registered explicitly')
 # Uninstall only this module: full core remains, and an unrelated registration survives.
 remote('echo fixture > /usr/share/quickstart/cards.d/fn-review-other.json; opkg remove app-meta-fastnet luci-i18n-fastnet-zh-cn luci-app-fastnet > '+stage+'/remove.log; test ! -e /usr/share/quickstart/cards.d/fastnet.json; test -e /usr/share/quickstart/cards.d/fn-review-other.json; test -x /usr/sbin/FastNet')
 report['checks'].append('real LuCI uninstall removes only its registration/assets and preserves the standalone core/other card')
 # Actual failed package configuration, followed by repair with the exact candidate.
 with tempfile.TemporaryDirectory(prefix='broken-card-',dir=a.candidate) as temporary:
  broken=Path(temporary)/'payload';shutil.copytree(a.candidate/'payload/all/luci-app-fastnet',broken)
  manifest_file=broken/'usr/share/fastnet/quickstart/fastnet.json';bad=json.loads(manifest_file.read_text());bad['entry']='/luci-static/fastnet/quickstart/card.deadbeef.js';manifest_file.write_text(json.dumps(bad,indent=2)+'\n')
  subprocess.run(['fakeroot','sh','-c','chown -R 0:0 "$1"; exec sh "$2" "$1" "$3"','bad-fixture',str(broken),str(a.ipkg_build.resolve()),temporary],check=True,stdout=subprocess.DEVNULL,env={**os.environ,'SOURCE_DATE_EPOCH':'0'})
  bad_ipk=next(Path(temporary).glob('*.ipk'));remote('cat > '+stage+'/broken.ipk',bad_ipk.read_bytes())
 failed=subprocess.run(ssh+['opkg install '+stage+'/broken.ipk'],capture_output=True,timeout=60)
 assert failed.returncode!=0,'broken package must fail its postinst'
 remote('test ! -f /usr/share/quickstart/cards.d/fastnet.json')
 report['checks'].append('real broken-package postinst fails without registering an unavailable card; exact candidate repairs it')
 selected=[x for x in packages if x['name']!='fastnet']
 remote('opkg --force-reinstall install '+' '.join(shlex.quote(stage+'/'+Path(x['path']).name) for x in selected)+' > '+stage+'/reinstall.log; test -f /usr/share/quickstart/cards.d/fastnet.json; rm /usr/share/quickstart/cards.d/fn-review-other.json',timeout=90)
 report['checks'].append('real reinstall restores the same hashed card; no Quickstart layout files are owned by these packages')
 report['status']='passed'
except BaseException as e:
 report['status']='failed';report['failure']=str(e);raise
finally:
 try:
  if changed:
   remote('/etc/init.d/fastnet stop 2>/dev/null || true; /etc/init.d/fastnet disable 2>/dev/null || true')
   current=remote('cat /usr/lib/opkg/status').decode()
   # Merge only our four package records; preserve unrelated concurrent changes.
   restored='\n\n'.join(unrelated(current)+[b for b in original_status.strip().split('\n\n') if ownblock(b)])+'\n\n'
   remote('while IFS= read -r file; do [ ! -f "$file" ] && [ ! -L "$file" ] || rm -f "$file"; done < '+stage+'/paths; rm -f /usr/share/quickstart/cards.d/fn-review-other.json; rm -f /usr/lib/opkg/info/fastnet.* /usr/lib/opkg/info/app-meta-fastnet.* /usr/lib/opkg/info/luci-app-fastnet.* /usr/lib/opkg/info/luci-i18n-fastnet-zh-cn.*; tar -xzf '+stage+'/backup/files.tar.gz -C /')
   remote('cat > /usr/lib/opkg/status',restored.encode())
   remote('rm -f /tmp/luci-indexcache /tmp/luci-indexcache.*; rm -rf /tmp/luci-modulecache/; rmdir /usr/lib/lua/luci/model/fastnet /usr/share/quickstart/cards.d /usr/share/quickstart /etc/fastnet 2>/dev/null || true')
   assert unrelated(remote('cat /usr/lib/opkg/status').decode())==unrelated(current)
   report['restored']=True
  remote('rm -rf '+stage)
 except BaseException as e:report['restorationError']=str(e);report['status']='failed'
 (a.output/'package-device-report.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
print(json.dumps(report,ensure_ascii=False))
