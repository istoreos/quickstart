#!/usr/bin/env python3
"""Non-destructive HTTP boundary and business-read audit of an SSH-authorized device.

Temporary rpcd sessions are destroyed in finally. No credentials or API response
bodies are persisted. This verifies sessions, not a successful password login.
"""
import argparse
import datetime
import json
from pathlib import Path
import re
import secrets
import shlex
import ssl
import subprocess
import time
import urllib.error
import urllib.request

ROOT = Path(__file__).resolve().parents[2]
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--host', required=True)
parser.add_argument('--output', required=True, type=Path)
parser.add_argument('--ui', action='store_true')
args = parser.parse_args()
if not re.fullmatch(r'[A-Za-z0-9.-]+', args.host):
    parser.error('invalid host')
args.output.mkdir(parents=True, exist_ok=True)
context = ssl._create_unverified_context()  # Appliance's existing self-signed TLS.
base = 'https://' + args.host
prefix = '/cgi-bin/luci/istore/'
http_socket = '/var/run/quickstart/http.sock'
report = {'host': args.host, 'at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
          'checks': [], 'businessReads': [], 'passwordLoginVerified': False,
          'tlsCertificateTrusted': False, 'temporarySessionsDestroyed': False}
sessions = []
completed = False


def remote(command):
    return subprocess.check_output(['ssh', '-o', 'BatchMode=yes', '-o', 'ConnectTimeout=5',
                                    'root@' + args.host, command], text=True, timeout=30)


def ubus(method, data):
    raw = remote('ubus -S call session ' + method + ' ' + shlex.quote(json.dumps(data)))
    return json.loads(raw) if raw.strip() else {}


def session(username):
    sid = ubus('create', {'timeout': 600})['ubus_rpc_session']
    sessions.append(sid)
    token = secrets.token_hex(16)
    ubus('set', {'ubus_rpc_session': sid, 'values': {'username': username, 'token': token}})
    if username == 'root':
        for scope, objects in [('ubus', [['*', '*']]), ('access-group', [['*', 'read']]), ('uci', [['*', 'read']])]:
            ubus('grant', {'ubus_rpc_session': sid, 'scope': scope, 'objects': objects})
    return sid, token


def request(path, method='GET', sid=None, headers=None, body=None):
    hdr = dict(headers or {})
    if sid:
        hdr['Cookie'] = 'sysauth_https=' + sid
    req = urllib.request.Request(base + path, method=method, data=body, headers=hdr)
    try:
        response = urllib.request.urlopen(req, context=context, timeout=20)
    except urllib.error.HTTPError as err:
        response = err
    with response:
        raw = response.read(2 * 1024 * 1024 + 1)
        try:
            value = json.loads(raw)
        except (ValueError, UnicodeError):
            value = None
        return response.status, value, {k.lower(): v for k, v in response.headers.items()}


def check(name, path=prefix + 'system/version/', **kwargs):
    expected = kwargs.pop('expected', 'denied')
    status, value, headers = request(path, **kwargs)
    denied = status in (401, 403) or isinstance(value, dict) and value.get('success') == -1001
    passed = denied if expected == 'denied' else status == expected
    report['checks'].append({'name': name, 'http': status,
                             'success': value.get('success') if isinstance(value, dict) else None,
                             'passed': passed})


try:
    root, token = session('root')
    guest, _ = session('security-audit-guest')
    check('anonymous')
    check('forged forwarded headers', headers={'X-Forwarded-Sid': '1' * 32, 'X-Forwarded-Token': 'forged'})
    check('zero session', sid='0' * 32)
    check('unknown session', sid='1' * 32)
    check('non-root live session', sid=guest)
    status, value, _ = request(prefix+'system/getToken/', headers={'Cookie':'sysauth='+guest+'; sysauth_https='+root})
    report['checks'].append({'name':'token bound to authenticated session with mixed cookies',
                            'passed':status==200 and value.get('result', {}).get('token')==token})
    expired = ubus('create', {'timeout': 1})['ubus_rpc_session']
    sessions.append(expired)
    ubus('set', {'ubus_rpc_session': expired, 'values': {'username': 'root', 'token': secrets.token_hex(16)}})
    time.sleep(2)
    check('expired root session', sid=expired)
    for cookie in ('sysauth', 'sysauth_http', 'sysauth_https'):
        status, value, headers = request(prefix + 'system/version/', headers={'Cookie': cookie + '=' + root})
        report['checks'].append({'name': cookie + ' valid root', 'http': status,
                                'passed': status == 200 and isinstance(value, dict) and
                                isinstance(value.get('result'), dict) and headers.get('cache-control') == 'no-store'})
    # Invalid JSON ensures that an accidental missing guard cannot change state.
    write = prefix + 'system/module-settings/'
    for name, headers, expected in [
        ('cross-origin JSON', {'Origin': 'https://attacker.invalid', 'Content-Type': 'application/json'}, 403),
        ('cross-origin text', {'Origin': 'https://attacker.invalid', 'Content-Type': 'text/plain'}, 403),
        ('null origin', {'Origin': 'null', 'Content-Type': 'application/json'}, 403),
        # This firmware's uhttpd omits Sec-Fetch-* from the CGI environment.
        # Real cross-origin POSTs also include Origin; assert that boundary.
        ('cross-site fetch', {'Origin': 'https://attacker.invalid', 'Sec-Fetch-Site': 'cross-site', 'Content-Type': 'application/json'}, 403),
        ('same-site fetch', {'Origin': 'https://other.invalid', 'Sec-Fetch-Site': 'same-site', 'Content-Type': 'application/json'}, 403),
        ('HTML form', {'Content-Type': 'application/x-www-form-urlencoded'}, 415),
        ('plain text', {'Content-Type': 'text/plain'}, 415),
    ]:
        check(name, write, method='POST', sid=root, headers=headers, body=b'{', expected=expected)
    check('same-origin JSON reaches parser', write, method='POST', sid=root,
          headers={'Origin': base, 'Content-Type': 'application/json'}, body=b'{', expected=200)
    # The HTTP API is Unix-only; test its authentication directly through SSH.
    for name, hdr in [('direct forged session', {'X-Forwarded-Sid': '1'*32, 'X-Forwarded-Token': 'forged'}),
                      ('direct wrong token', {'X-Forwarded-Sid': root, 'X-Forwarded-Token': 'wrong'})]:
        command = ['curl', '-fsS', '--max-time', '10', '--unix-socket', http_socket]
        for k, v in hdr.items():
            command += ['-H', k + ': ' + v]
        command += ['http://localhost' + prefix + 'system/version/']
        value = json.loads(remote(shlex.join(command)))
        report['checks'].append({'name': name, 'passed': value.get('success') == -1001})
    # Send only the oversized Content-Length over the HTTP socket: exercise the
    # backend limit independently of LuCI's smaller 100 KiB upload limit.
    command = ['curl', '-sS', '--max-time', '10', '--unix-socket', http_socket,
               '-o', '/dev/null', '-w', '%{http_code}', '-X', 'POST',
               '-H', 'X-Forwarded-Sid: '+root, '-H', 'X-Forwarded-Token: '+token,
               '-H', 'Content-Type: application/json', '-H', 'Content-Length: 1048577',
               'http://localhost'+write]
    result = remote(shlex.join(command)).strip()
    report['checks'].append({'name':'direct oversized Content-Length', 'passed':result=='413'})
    # Literal public-path inventory, plus generated home paths. This is a
    # denial sweep, not evidence that every method's business behavior passed.
    paths = {v for p in (ROOT/'backend/modules').rglob('*.go') if not p.name.endswith('_test.go')
             for v in re.findall(r'"(/cgi-bin/luci/istore/[^"\n]+)"', p.read_text())}
    paths = {p for p in paths if not p.endswith('/home/v1/')}
    paths.update(prefix+'home/v1/'+d+'/' for d in ('layout', 'launcher'))
    failures = []
    for path in sorted(paths):
        status, value, _ = request(path)
        if status not in (401, 403) and not (isinstance(value, dict) and value.get('success') == -1001):
            failures.append({'path': path, 'http': status})
    report['anonymousPathSweep'] = {'count': len(paths), 'failures': failures, 'passed': not failures}
    reads = [
        'system/version/', 'u/system/version/', 'system/time/', 'system/status/', 'system/module-settings/',
        'network/status/', 'network/statistics/', 'network/port/list/', 'network/device/list/',
        'nas/disk/status/', 'u/nas/service/status/', 'raid/list/', 'smart/list/',
        'guide/docker/status/', 'guide/download-service/status/', 'u/guide/ddns/',
        'share/user/list/', 'share/service/list/', 'share/protocol/webdav/', 'share/protocol/samba/',
        'app/install-list/', 'lanctrl/globalConfigs/', 'lanctrl/v2/devices/', 'lanctrl/v2/router-context/',
        'lanctrl/v2/device-traffic/', 'lanctrl/v2/network-rules/', 'lanctrl/v2/migration/plan/',
        'lanctrl/v2/feature-components/', 'lanctrl/v2/device-groups/', 'home/v1/layout/', 'home/v1/launcher/',
    ]
    for path in reads:
        try:
            status, value, _ = request(prefix+path, sid=root)
            report['businessReads'].append({'path': path, 'http': status,
                'success': value.get('success', 0) if isinstance(value, dict) else None,
                'hasResult': isinstance(value, list) or isinstance(value, dict) and value.get('result') is not None,
                'hasError': bool(value.get('error')) if isinstance(value, dict) else value is None})
        except (OSError, ValueError) as err:
            report['businessReads'].append({'path': path, 'http': 0, 'errorType': type(err).__name__})
    # The home store guarantees that saving identical data is a no-op. Verify
    # real authenticated POST and conflict handling without changing settings.
    report['homeNoOpWrites'] = []
    for domain in ('layout', 'launcher'):
        path = prefix+'home/v1/'+domain+'/'
        status, value, _ = request(path, sid=root)
        snapshot = value['result']
        before = remote('sha256sum /etc/quickstart/home-'+domain+'.json; stat -c "%i %Y %a" /etc/quickstart/home-'+domain+'.json')
        body = json.dumps({'expectedVersion': snapshot['version'], 'value': snapshot['value']}).encode()
        status, saved, _ = request(path, method='POST', sid=root,
                                   headers={'Origin':base,'Content-Type':'application/json'}, body=body)
        after = remote('sha256sum /etc/quickstart/home-'+domain+'.json; stat -c "%i %Y %a" /etc/quickstart/home-'+domain+'.json')
        report['homeNoOpWrites'].append({'domain':domain, 'http':status,
            'passed':status==200 and saved['result']['version']==snapshot['version'] and before==after})
        body = json.dumps({'expectedVersion': 'stale-audit-version', 'value': snapshot['value']}).encode()
        check('stale '+domain+' version', path, method='POST', sid=root,
              headers={'Origin':base,'Content-Type':'application/json'}, body=body, expected=409)
    if args.ui:
        import os
        env = dict(os.environ, NODE_PATH='/config/playwright-runner/node_modules', QS_SECURITY_HOST=args.host,
                   QS_SECURITY_SID=root, QS_SECURITY_OUTPUT=str(args.output.resolve()))
        result = subprocess.run(['node', str(Path(__file__).with_name('release-security-ui.cjs'))], env=env, timeout=240)
        report['uiExitCode'] = result.returncode
    ubus('destroy', {'ubus_rpc_session': root})
    sessions.remove(root)
    check('revoked session', sid=root)
    completed = True
finally:
    cleanup_failures = 0
    for sid in sessions:
        try:
            ubus('destroy', {'ubus_rpc_session': sid})
        except subprocess.CalledProcessError as error:
            # The one-second expiry case is already absent in rpcd.
            if error.returncode != 4:
                cleanup_failures += 1
    report['temporarySessionsDestroyed'] = cleanup_failures == 0
    report['securityPassed'] = (completed and cleanup_failures == 0 and all(c['passed'] for c in report['checks'])
        and report.get('anonymousPathSweep', {}).get('passed', False)
        and all(c['passed'] for c in report.get('homeNoOpWrites', [])))
    (args.output/'security.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
print(json.dumps({'securityPassed': report['securityPassed'], 'checks': len(report['checks']),
                  'paths': report.get('anonymousPathSweep', {}).get('count'), 'reads':len(report['businessReads'])}))
raise SystemExit(0 if report['securityPassed'] else 1)
