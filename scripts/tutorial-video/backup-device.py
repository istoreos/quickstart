#!/usr/bin/env python3
"""Back up just FastNet package-owned files and package records before filming."""
import argparse,json,shlex,subprocess,uuid
from pathlib import Path
from device import remote
from settings import load_settings
p=argparse.ArgumentParser();p.add_argument('--device',default=load_settings()['installDevice']['ssh']);p.add_argument('--output',type=Path,default=Path(__file__).parent/'.local/installation-lab');a=p.parse_args();a.output.mkdir(parents=True,exist_ok=True)
stage='/root/quickstart-video-backup-'+uuid.uuid4().hex[:12]
names=['fastnet','luci-app-fastnet','luci-i18n-fastnet-zh-cn','app-meta-fastnet']
status=remote(a.device,'cat /usr/lib/opkg/status')
blocks=[b for b in status.strip().split('\n\n') if b.splitlines()[0] in ['Package: '+n for n in names]]
(a.output/'package-records.txt').write_text('\n\n'.join(blocks)+'\n\n')
paths=set(['/etc/config/fastnet','/etc/fastnet/card-state.json','/etc/config/network','/etc/config/dhcp','/etc/config/firewall','/etc/config/ucitrack'])
for name in names:
 paths.update(x for x in remote(a.device,'opkg files '+name).splitlines() if x.startswith('/'))
paths.update(remote(a.device,'ls /usr/lib/opkg/info/*fastnet* /etc/rc.d/*fastnet 2>/dev/null || true').splitlines())
command='set -eu; umask 077; test ! -e '+stage+'; mkdir '+stage+'; '
command+='printf %s '+shlex.quote('\n'.join(sorted(paths))+'\n')+' > '+stage+'/paths; '
command+='while IFS= read -r p; do if [ -f "$p" ] || [ -L "$p" ]; then echo "${p#/}"; fi; done < '+stage+'/paths > '+stage+'/existing; '
command+='tar -czf '+stage+'/files.tar.gz -C / -T '+stage+'/existing; '
command+='sha256sum /etc/config/network /etc/config/dhcp /etc/config/firewall > '+stage+'/network.sha256; '
command+='cp /usr/lib/opkg/status '+stage+'/status; sha256sum '+stage+'/files.tar.gz'
print(remote(a.device,command).strip())
subprocess.run(['scp',a.device+':'+stage+'/files.tar.gz',str(a.output/'files.tar.gz')],check=True)
(a.output/'metadata.json').write_text(json.dumps({'device':a.device,'stage':stage,'paths':sorted(paths)},indent=2))
