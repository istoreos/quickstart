#!/usr/bin/env python3
"""Keep the tested Quickstart candidate on the explicitly named lab appliance.

Uses the existing atomic candidate installer. Does not install FastNet, alter
iStore code, reset the homepage, or run a speed test. Private rollback material
stays on the appliance; the local report contains only hashes and status.
"""
import argparse, datetime, hashlib, json, shlex, subprocess, zipfile
from pathlib import Path

p=argparse.ArgumentParser()
p.add_argument('--host',choices=['root@192.168.30.7'],required=True)
p.add_argument('--bundle',type=Path,required=True)
p.add_argument('--output',type=Path,required=True)
a=p.parse_args();a.output.mkdir(parents=True,exist_ok=False)
root=Path(__file__).resolve().parents[2]
sha=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
for line in (a.bundle/'SHA256SUMS').read_text().splitlines():
 digest,name=line.split('  ',1);assert sha(a.bundle/name)==digest,name
binary=a.bundle/'quickstart/quickstart.x86_64'
web=a.bundle/'quickstart/www/luci-static/quickstart'
release='fastnet-ui-'+datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ')+'-'+sha(binary)[:8]
stage='/tmp/'+release;backup='/root/quickstart-backups/'+release
ssh=['ssh','-o','BatchMode=yes','-o','ConnectTimeout=5',a.host]
q=shlex.quote
def remote(command,data=None,timeout=60):
 result=subprocess.run(ssh+[command],input=data,capture_output=True,timeout=timeout)
 if result.returncode:raise RuntimeError(result.stderr.decode(errors='replace')[-1200:])
 return result.stdout
def snapshot():
 paths=['/etc/quickstart/home-layout.json','/etc/quickstart/home-launcher.json','/usr/sbin/FastNet','/etc/config/network','/etc/config/dhcp','/etc/config/firewall']
 return {path:remote('if [ -f '+q(path)+' ]; then sha256sum '+q(path)+'; fi').decode().split()[0] if remote('test -f '+q(path)+' && echo yes || true').strip() else None for path in paths}
report={'status':'in_progress','host':a.host,'release':release,'privateBackup':backup,'persistent':True,'FastNetInstalledByDeployment':False,'iStoreCodeChanged':False}
try:
 report['before']=snapshot()
 remote('umask 077; mkdir '+q(stage))
 remote('cat > '+q(stage+'/quickstart'),binary.read_bytes())
 archive=a.output/'web.zip'
 with zipfile.ZipFile(archive,'w',zipfile.ZIP_DEFLATED) as z:
  for file in sorted(web.rglob('*')):
   if file.is_file():z.write(file,'luci-static/quickstart/'+str(file.relative_to(web)))
 remote('cat > '+q(stage+'/web.zip'),archive.read_bytes())
 installer=root/'scripts/ops/install-candidate-remote.sh'
 remote('cat > '+q(stage+'/install.sh'),installer.read_bytes())
 # Keep the original homepage documents in a private recovery snapshot as well.
 remote('tar -czf '+q(stage+'/home-config.before.tar.gz')+' -C / etc/quickstart')
 args=[release,stage+'/quickstart',stage+'/web.zip','/usr/sbin/quickstart','/www/luci-static/quickstart','/usr/lib/lua/luci/view/quickstart/main.htm','/root/quickstart-backups','/etc/init.d/quickstart',sha(binary),sha(web/'index.js'),sha(web/'i18n/en.json'),release,'http://127.0.0.1:3038/cgi-bin/luci/istore/version/','/tmp/quickstart-candidates','20','/',sha(web/'style.css'),sha(web/'vendor.js')]
 output=remote('sh '+q(stage+'/install.sh')+' '+' '.join(map(q,args)),timeout=90)
 (a.output/'installer.log').write_bytes(output)
 remote('mv '+q(stage+'/home-config.before.tar.gz')+' '+q(backup+'/home-config.before.tar.gz'))
 report['artifacts']={'backend':sha(binary),'web':{str(f.relative_to(web)):sha(f) for f in web.rglob('*') if f.is_file()}}
 report['after']=snapshot();assert report['before']==report['after'],'deployment changed existing homepage, network configuration or manual FastNet core'
 assert remote('sha256sum /usr/sbin/quickstart').decode().split()[0]==sha(binary)
 assert not remote('opkg list-installed app-meta-fastnet').strip(),'FastNet was unexpectedly installed'
 rollback='''#!/bin/sh
set -eu
/etc/init.d/quickstart stop
cp -p "$(dirname "$0")/quickstart" /usr/sbin/quickstart
chmod 755 /usr/sbin/quickstart
rm -rf /www/luci-static/quickstart
cp -a "$(dirname "$0")/web" /www/luci-static/quickstart
cp -p "$(dirname "$0")/main.htm" /usr/lib/lua/luci/view/quickstart/main.htm
# Keep later user layout changes. Old hosts protect layout v4 as read-only.
/etc/init.d/quickstart start
'''
 remote('cat > '+q(backup+'/rollback.sh'),rollback.encode());remote('chmod 700 '+q(backup+'/rollback.sh')+'; sync')
 report['status']='deployed'
finally:
 remote('rm -rf '+q(stage))
 (a.output/'deployment.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
print(json.dumps({'status':report['status'],'release':release,'backup':backup}))
