#!/usr/bin/env python3
"""Run a Windows browser script with a short-lived LuCI session via stdin."""
import argparse
import json
import secrets
import shlex
import subprocess
import sys
from remote import powershell
from settings import load_settings


def remote(host, command):
    return subprocess.check_output(['ssh', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=8',
                                    '-o', 'StrictHostKeyChecking=yes', host, command], text=True)


def ubus(host, method, data):
    result = remote(host, 'ubus -S call session ' + method + ' ' + shlex.quote(json.dumps(data)))
    return json.loads(result or '{}')


if __name__ == '__main__':
    p = argparse.ArgumentParser()
    p.add_argument('script')
    settings=load_settings()
    p.add_argument('--device', default=settings['homeDevice']['ssh'])
    p.add_argument('--base', default=settings['homeDevice']['base'])
    p.add_argument('--server', default=settings['videoServer'])
    p.add_argument('--mode', default='probe')
    args = p.parse_args()
    if not all(c.isalnum() or c in '._-' for c in args.script):
        p.error('script must be a basename')
    sid = None
    try:
        sid = ubus(args.device, 'create', {'timeout': 1800})['ubus_rpc_session']
        ubus(args.device, 'set', {'ubus_rpc_session': sid, 'values': {'username': 'root', 'token': secrets.token_hex(16)}})
        for scope, objects in [('ubus', [['*', '*']]), ('access-group', [['*', 'read'], ['*', 'write']]), ('uci', [['*', 'read'], ['*', 'write']])]:
            ubus(args.device, 'grant', {'ubus_rpc_session': sid, 'scope': scope, 'objects': objects})
        payload = json.dumps({'sid': sid, 'base': args.base, 'mode': args.mode, 'installationLab':args.device==settings['installDevice']['ssh']}) + '\n'
        script = 'Set-Location ($env:TUTORIAL_SERVER_ROOT+"/tool"); & node ' + args.script + '; exit $LASTEXITCODE'
        code = powershell(args.server, script, payload).returncode
    finally:
        if sid:
            ubus(args.device, 'destroy', {'ubus_rpc_session': sid})
    sys.exit(code)
