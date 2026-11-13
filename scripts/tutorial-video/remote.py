#!/usr/bin/env python3
"""Small SSH transport; credentials stay in SSH and JSON stdin, never source."""
import argparse
import base64
import subprocess
import sys
from pathlib import Path
from settings import load_settings,windows_environment


def powershell(host, script, payload=None):
    prefix = '$ProgressPreference="SilentlyContinue"; [Console]::OutputEncoding=[Text.UTF8Encoding]::new(); '
    prefix += '$env:Path=[Environment]::GetEnvironmentVariable("Path","Machine")+";"+[Environment]::GetEnvironmentVariable("Path","User"); '
    prefix += windows_environment(load_settings())
    encoded = base64.b64encode((prefix + script).encode('utf-16le')).decode()
    return subprocess.run(
        ['ssh', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=10', '-o', 'StrictHostKeyChecking=yes', host,
         'powershell.exe -NoLogo -NoProfile -OutputFormat Text -EncodedCommand ' + encoded],
        input=payload, text=True, check=False)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--host', default=load_settings()['videoServer'])
    parser.add_argument('--script', type=Path)
    parser.add_argument('--command')
    args = parser.parse_args()
    script = args.script.read_text(encoding='utf-8') if args.script else args.command
    if not script:
        parser.error('provide --script or --command')
    sys.exit(powershell(args.host, script).returncode)
