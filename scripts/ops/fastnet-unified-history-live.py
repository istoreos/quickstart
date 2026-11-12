#!/usr/bin/env python3
"""Ephemeral root session for the fixed .7 integration check; always destroyed."""
import argparse,json,os,secrets,shlex,subprocess
from pathlib import Path
p=argparse.ArgumentParser();p.add_argument('--output',type=Path,required=True);p.add_argument('--finish-only',action='store_true');a=p.parse_args();a.output.mkdir(parents=True,exist_ok=True)
def remote(command):return subprocess.check_output(['ssh','-o','BatchMode=yes','-o','ConnectTimeout=5','root@192.168.30.7',command])
def ubus(action,value):return json.loads(remote('ubus -S call session '+action+' '+shlex.quote(json.dumps(value))) or b'{}')
sid=None
try:
 sid=ubus('create',{'timeout':600})['ubus_rpc_session'];ubus('set',{'ubus_rpc_session':sid,'values':{'username':'root','token':secrets.token_hex(16)}})
 for scope,objects in [('ubus',[['*','*']]),('access-group',[['*','read'],['*','write']]),('uci',[['*','read'],['*','write']])]:ubus('grant',{'ubus_rpc_session':sid,'scope':scope,'objects':objects})
 env={**os.environ,'DEVICE_SESSION':sid,'NODE_PATH':'/tmp/quickstart-playwright/node_modules','OUTPUT_DIR':str(a.output.resolve()),'FASTNET_LIVE_FINISH_ONLY':'1' if a.finish_only else '0'}
 result=subprocess.run(['node',str(Path(__file__).with_suffix('.cjs'))],env=env,stdout=subprocess.PIPE,stderr=subprocess.STDOUT,timeout=400)
 (a.output/'browser.log').write_bytes(result.stdout);print(result.stdout.decode());raise SystemExit(result.returncode)
finally:
 if sid:ubus('destroy',{'ubus_rpc_session':sid})
