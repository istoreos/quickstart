#!/usr/bin/env python3
"""Restore paired old binaries, config and data inside an owned router namespace."""
import argparse
import hashlib
import json
import pathlib
import shlex
import subprocess
import tarfile
import time
import uuid
from urllib.parse import urlencode

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--application', required=True)
parser.add_argument('--legacy-application', required=True)
parser.add_argument('--engine', required=True)
parser.add_argument('--legacy-engine', required=True)
parser.add_argument('--go-test', required=True)
parser.add_argument('--work', type=pathlib.Path, required=True)
parser.add_argument('--output', type=pathlib.Path, required=True)
args = parser.parse_args()
target = 'root@192.168.30.7'
identity = uuid.uuid4().hex[:6]
base = '/tmp/quickstart-multi-lan/m10-' + identity
router = 'qsm10-' + identity
namespaces = [router, router + '-w', router + '-c1', router + '-c2']
report = {'host': target, 'base': base, 'status': 'starting', 'checks': [],
          'namespaceIsolation': True, 'rpcMetadata': 'synthetic netifd/rpcd only',
          'legacyQuickstartCommit': '66589b22f140615db97851b6c3b1eec7aad975d5',
          'legacyEngineCommit': '54b5338ad8e03f46ba04387eb232aeb676ff68d2'}
args.work.mkdir(parents=True, exist_ok=True)
args.output.parent.mkdir(parents=True, exist_ok=True)


def remote(command, check=True, timeout=60):
    result = subprocess.run(['ssh', '-o', 'BatchMode=yes', target,
                             shlex.join(command)], capture_output=True,
                            text=True, timeout=timeout)
    if check and result.returncode:
        raise RuntimeError((command, result.stdout[-2000:], result.stderr[-2000:]))
    return result


def inside(command, **options):
    return remote(['ip', 'netns', 'exec', router] + command, **options)


def upload(source, destination):
    subprocess.run(['scp', '-O', '-q', str(source), target + ':' + destination],
                   check=True, timeout=60)


def write(path, content):
    remote(['sh', '-c', 'printf %s ' + shlex.quote(content) + ' > ' + shlex.quote(path)])


def api(endpoint, body=None, network=None, native=False, check=True, query=None):
    port = 8765 if native else 19481
    prefix = '/api/v1/' if native else '/cgi-bin/luci/istore/lanctrl/v2/'
    path = prefix + endpoint + ('' if native else '/')
    params = dict(query or {})
    if network is not None:
        params['network'] = network
    if params:
        path += '?' + urlencode(params)
    command = ['curl', '-sS', '--max-time', '8', '-H',
               'X-Forwarded-Sid: 11111111111111111111111111111111', '-H',
               'X-Forwarded-Token: m10-isolated']
    if body is not None:
        command += ['-H', 'Content-Type: application/json', '--data-binary', json.dumps(body)]
    result = json.loads(inside(command + ['http://127.0.0.1:' + str(port) + path]).stdout)
    if check:
        assert not result.get('error') and not result.get('result', {}).get('error'), result
    return result


def stop_pid_file(name):
    result = remote(['cat', base + '/' + name], check=False)
    if result.returncode:
        return
    pid = result.stdout.strip()
    assert pid.isdigit()
    owned = remote(['ip', 'netns', 'pids', router]).stdout.split()
    if pid in owned:
        remote(['kill', '-TERM', pid], check=False)
    deadline = time.monotonic() + 10
    while pid in remote(['ip', 'netns', 'pids', router]).stdout.split():
        if time.monotonic() > deadline:
            remote(['kill', '-KILL', pid], check=False)
            break
        time.sleep(.2)


def start_engine(legacy, index=1):
    scopes = json.loads(remote(['cat', base + '/scopes.json']).stdout)
    port = 8765 if index == 1 else 19500 + index
    command = [base + '/rootfs/opt/quickstart-netpolicy', '--iface', scopes[index-1]['device'],
               '--port', str(port), '--data-dir', base + '/engine-' + str(index),
               '--enable-traffic']
    if not legacy:
        command += ['--network-id', scopes[index-1]['networkId'],
                    '--instance-id', 'main' if index == 1 else scopes[index-1]['networkId']]
    shell = shlex.join(['ip', 'netns', 'exec', router] + command)
    shell += ' > ' + shlex.quote(base + '/engine-' + str(index) + '-rollback.log')
    shell += ' 2>&1 & echo $! > ' + shlex.quote(base + '/engine-' + str(index) + '.pid')
    remote(['sh', '-c', shell])
    deadline = time.monotonic() + 30
    while True:
        response = inside(['curl', '-sS', '--max-time', '2',
                           f'http://127.0.0.1:{port}/api/v1/system/health'], check=False)
        if response.returncode == 0 and json.loads(response.stdout)['data']['state'] == 'healthy':
            return
        if time.monotonic() > deadline:
            raise RuntimeError(remote(['cat', base + '/engine-' + str(index) + '-rollback.log']).stdout)
        time.sleep(.25)


def start_application(legacy, reuse=True):
    remote(['rm', '-f', base + '/backend.ready'])
    command = ['ip', 'netns', 'exec', router, 'env',
               'QUICKSTART_MULTI_LAN_CAPACITY_BASE=' + base,
               'QUICKSTART_MULTI_LAN_APPLICATION_LEGACY=' + ('1' if legacy else '0'),
               'QUICKSTART_MULTI_LAN_APPLICATION_REUSE_ROOT=' + ('1' if reuse else '0'),
               base + '/service.test', '-test.run', '^TestMultiLANFullApplication$',
               '-test.timeout', '30m']
    shell = shlex.join(command) + ' > ' + shlex.quote(base + '/application-rollback.log')
    shell += ' 2>&1 & echo $! > ' + shlex.quote(base + '/application-launcher.pid')
    remote(['sh', '-c', shell])
    deadline = time.monotonic() + 70
    while remote(['test', '-f', base + '/backend.ready'], check=False).returncode:
        if time.monotonic() > deadline:
            raise RuntimeError(remote(['cat', base + '/application-rollback.log']).stdout[-6000:])
        time.sleep(.5)


def stop_application():
    stop_pid_file('application-launcher.pid')
    stop_pid_file('backend.pid')


def snapshot(name):
    archive = base + '/' + name + '.tar'
    remote(['tar', '-cf', archive, '-C', base, 'rootfs', 'engine-1', 'engine-2'])
    file = args.work / (identity + '-' + name + '.tar')
    subprocess.run(['scp', '-O', '-q', target + ':' + archive, str(file)], check=True, timeout=60)
    with tarfile.open(file) as stream:
        hashes = {member.name: hashlib.sha256(stream.extractfile(member).read()).hexdigest()
                  for member in stream if member.isfile()}
    return file, hashes


before = remote(['sha256sum', '/etc/config/network', '/etc/config/dhcp', '/etc/config/firewall']).stdout
try:
    print(json.dumps({'stage': 'setup', 'base': base}), flush=True)
    remote(['mkdir', '-p', base])
    for source, name in [(args.engine, 'quickstart-netpolicy'), (args.go_test, 'service.test'),
                         (args.legacy_application, 'application'),
                         (pathlib.Path(__file__).with_name('multi-lan-capacity-namespace.sh'), 'setup.sh')]:
        upload(source, base + '/' + name)
    remote(['chmod', '700', base + '/quickstart-netpolicy', base + '/service.test'])
    remote(['sh', base + '/setup.sh', base, router, namespaces[1], 'qc' + identity, '2', '0'], timeout=180)
    stop_pid_file('backend.pid')
    for index in (1, 2):
        stop_pid_file('engine-' + str(index) + '.pid')
    scopes = json.loads(remote(['cat', base + '/scopes.json']).stdout)
    scopes[0]['networkId'] = 'lan'
    # The pinned old executable deliberately expects its original br-lan.
    remote(['ip', '-n', router, 'link', 'set', scopes[0]['device'], 'down'])
    remote(['ip', '-n', router, 'link', 'set', scopes[0]['device'], 'name', 'br-lan'])
    remote(['ip', '-n', router, 'link', 'set', 'br-lan', 'up'])
    scopes[0]['device'] = 'br-lan'
    write(base + '/scopes.json', json.dumps(scopes))
    remote(['cp', base + '/networks/capacity1/traffic.json', base + '/traffic.json'])
    start_application(True, reuse=False)
    stop_application()
    root = base + '/rootfs'
    cfg = root + '/etc/config'
    shim_path = root + '/opt/test-bin/ubus'
    shim = remote(['cat', shim_path]).stdout
    status_case = r''' network.interface.*:status)
 name=${2#network.interface.}
 jsonfilter -i /opt/runtime.json -e "@.interface[@.interface=\"$name\"]";;
'''
    write(shim_path, shim.replace(' network.interface:dump)', status_case + ' network.interface:dump)'))
    write(cfg + '/dhcp', "config dnsmasq 'main'\n"
          "config dhcp 'lan'\n option interface 'lan'\n option instance 'main'\n"
          " option start '100'\n option limit '150'\n option leasetime '12h'\n"
          "config dhcp 'office_pool'\n option interface 'capacity2'\n option instance 'main'\n"
          " option start '100'\n option limit '150'\n option leasetime '12h'\n")
    uci = ['uci', '-c', cfg, '-t', base + '/uci-state']
    for scope in scopes:
        remote(uci + ['set', 'network.' + scope['networkId'] + '.ipaddr=' + scope['ipv4Prefixes'][0]])
    remote(uci + ['commit', 'network'])
    remote(uci + ['rename', 'quickstart-netpolicy.lan=main'])
    remote(uci + ['set', 'quickstart-netpolicy.main.port=8765'])
    remote(uci + ['set', 'quickstart-netpolicy.capacity2.enabled=0'])
    remote(uci + ['commit', 'quickstart-netpolicy'])
    write(root + '/etc/quickstart/rate-limit-provider', 'quickstart-native\n')
    # These directories contain only the disposable setup's engine state.
    for index in (1, 2):
        remote(['rm', '-rf', base + '/engine-' + str(index)])
        remote(['mkdir', '-p', base + '/engine-' + str(index)])
    upload(args.legacy_engine, root + '/opt/quickstart-netpolicy')
    remote(['chmod', '700', root + '/opt/quickstart-netpolicy'])
    start_engine(True)
    request = {'expectedRevision': 0, 'idempotencyKey': 'rollback-original',
               'owner': {'product': 'quickstart', 'instance': 'local-router'},
               'policies': [{'device': {'kind': 'mac', 'value': '02:00:00:00:00:01'},
                             'rateLimit': {'enabled': True, 'uploadBitsPerSecond': 3000000,
                                           'downloadBitsPerSecond': 6000000}}]}
    plan = api('policies/plan', request, native=True)['data']
    assert api('policies/apply', {'request': request, 'plan': plan}, native=True)['status'] == 'success'
    start_application(True)
    device = 'mac:02:00:00:00:00:01'
    api('device-profile', {'action': 'patch', 'deviceId': device, 'alias': '回退前默认设备'})
    api('device-groups', {'action': 'upsert_group', 'group': {'id': 'family', 'name': '原有分组', 'members': [device]}})
    def profile(network=None):
        return api('device-profile', network=network, query={'deviceId': device})['result']['profile']
    assert profile()['alias'] == '回退前默认设备'
    def speed():
        return api('device-policy', query={'deviceId': device})['result']['policy']['speed']
    assert speed() == {'enabled': True, 'uploadSpeed': 3, 'downloadSpeed': 6}, speed()
    stop_application()
    stop_pid_file('engine-1.pid')
    archive, original = snapshot('pre-upgrade')
    print(json.dumps({'stage': 'old_executables_backed_up', 'base': base}), flush=True)
    assert original['rootfs/opt/quickstart'] == hashlib.sha256(pathlib.Path(args.legacy_application).read_bytes()).hexdigest()
    upload(args.application, root + '/opt/quickstart')
    upload(args.engine, root + '/opt/quickstart-netpolicy')
    remote(['chmod', '700', root + '/opt/quickstart', root + '/opt/quickstart-netpolicy'])
    remote(uci + ['set', 'quickstart-netpolicy.capacity2.enabled=1'])
    remote(uci + ['commit', 'quickstart-netpolicy'])
    start_engine(False)
    start_engine(False, 2)
    start_application(False)
    assert len(api('networks')['result']['networks']) == 2
    assert profile('lan')['alias'] == '回退前默认设备'
    assert profile()['alias'] == '回退前默认设备'
    assert speed() == {'enabled': True, 'uploadSpeed': 3, 'downloadSpeed': 6}, speed()
    assert profile('capacity2').get('alias', '') == ''
    api('device-profile', {'action': 'patch', 'deviceId': device, 'alias': '升级后第二网设备'}, network='capacity2')
    assert profile('capacity2')['alias'] == '升级后第二网设备'
    assert profile('lan')['alias'] == '回退前默认设备'
    api('device-groups', {'action': 'upsert_group', 'group': {'id': 'office', 'name': '第二网分组', 'members': [device]}}, network='capacity2')
    report['checks'].append('old default profile/group and native revision seeded by actual old executables; new executables manage additional LAN')
    print(json.dumps({'stage': 'upgrade_verified', 'base': base}), flush=True)
    stop_application()
    for index in (1, 2):
        stop_pid_file('engine-' + str(index) + '.pid')
    _, upgraded = snapshot('upgraded')
    assert any(name.startswith('rootfs/etc/quickstart/networks/capacity2/') for name in upgraded)
    assert upgraded != original
    for name in ('rootfs', 'engine-1', 'engine-2'):
        remote(['mv', base + '/' + name, base + '/' + name + '-upgraded'])
    remote(['tar', '-xf', base + '/pre-upgrade.tar', '-C', base])
    _, restored = snapshot('restored')
    assert restored == original, 'binary/config/data restoration was not exact'
    report['checks'].append('both binaries, all configuration and both engine directories restored to exact pre-upgrade file hashes')
    start_engine(True)
    start_application(True)
    assert profile()['alias'] == '回退前默认设备'
    assert speed() == {'enabled': True, 'uploadSpeed': 3, 'downloadSpeed': 6}, speed()
    groups = api('device-groups')['result']['groups']
    assert len(groups) == 1 and groups[0]['name'] == '原有分组', groups
    policies = api('policies', native=True)['data']
    assert policies['revision'] == 1 and len(policies['policies']) == 1, policies
    report['checks'].append('restored old executables serve original profile, group and native policy revision; old/new/default client reads original 3/6 Mbit policy through port 8765 and original local-router owner')
    report.update(status='passed', backupSha256=hashlib.sha256(archive.read_bytes()).hexdigest(), restoredFileSha256=restored)
except BaseException as error:
    report.update(status='failed', error=repr(error))
    raise
finally:
    for namespace in namespaces:
        pids = remote(['ip', 'netns', 'pids', namespace], check=False).stdout.split()
        if pids:
            remote(['kill', '-TERM'] + pids, check=False)
    time.sleep(1)
    for namespace in namespaces:
        pids = remote(['ip', 'netns', 'pids', namespace], check=False).stdout.split()
        if pids:
            remote(['kill', '-KILL'] + pids, check=False)
        remote(['ip', 'netns', 'delete', namespace], check=False)
    report['rootConfigurationUnchanged'] = before == remote(['sha256sum', '/etc/config/network', '/etc/config/dhcp', '/etc/config/firewall']).stdout
    assert report['rootConfigurationUnchanged']
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'status': report['status'], 'output': str(args.output)}))
