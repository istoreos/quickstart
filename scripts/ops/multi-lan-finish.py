#!/usr/bin/env python3
"""Close M10 only after the fixed candidate's real 24-hour gates finish."""
import argparse
import datetime
import hashlib
import json
import os
import pathlib
import shlex
import signal
import subprocess
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--watch', action='store_true')
parser.add_argument('--optimization', type=int, default=5)

args = parser.parse_args()
prefix = f'optimization-{args.optimization}'
root = pathlib.Path(__file__).resolve().parents[2]
folder = root / 'docs/archive/evidence/multi-lan/M10'


def read(name):
    # Older running probes publish with write_text. Retry a partial read;
    # persistent malformed evidence still fails final acceptance.
    for attempt in range(10):
        try:
            return json.loads((folder / name).read_text())
        except json.JSONDecodeError:
            if attempt == 9:
                raise
            time.sleep(.2)


def write(path, value):
    temporary = path.with_name(path.name + '.pending')
    temporary.write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n')
    temporary.replace(path)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def verify_candidate():
    manifest = read(f'{prefix}-candidate-manifest.json')
    assert sha(root / 'docs/contracts/multi-lan.json') == manifest['contractSha256'], 'acceptance contract changed'
    for repository in manifest['repositories']:
        names = subprocess.check_output(['git', '-C', repository['path'], 'ls-files', '-z',
                                         '--cached', '--others', '--exclude-standard']).decode().split('\0')
        def production(name):
            if repository['kind'] == 'quickstart':
                return (name.startswith('backend/') and not name.endswith('_test.go')
                        or name.startswith('web/src/') or name in ('web/package.json', 'web/package-lock.json'))
            return name.startswith(('bandix/src/', 'bandix-ebpf/src/', 'packaging/', '.cargo/')) or name.endswith(('Cargo.toml', 'Cargo.lock'))
        current = {name for name in names if production(name) and (pathlib.Path(repository['path']) / name).is_file()}
        assert current == set(repository['productionFileSha256']), 'candidate source file list changed'
        for name, expected in repository['productionFileSha256'].items():
            path = pathlib.Path(repository['path']) / name
            assert path.is_file() and sha(path) == expected, 'candidate source changed: ' + str(path)
    for name, expected in manifest['artifactSha256'].items():
        path = pathlib.Path(name)
        assert path.is_file() and sha(path) == expected, 'candidate artifact changed: ' + name
    for arch in ('arm', 'x86'):
        startup = read(f'{prefix}-legacy-startup-{arch}.json')
        lifecycle = read(f'{prefix}-legacy-lifecycle-{arch}.json')
        assert lifecycle['passed'] and lifecycle['legacyManualStartupAndRebinding'], 'manual startup/rebind isolation failed'
        assert startup['passed'] and startup['productionConfigUnchanged'], 'legacy active-period startup isolation failed'
        for variant in ('missing', 'normal'):
            value = read(f'{prefix}-legacy-{variant}-{arch}.json')
            assert value['passed'] and value['productionConfigUnchanged'], 'legacy packet gate failed'
    assert read(f'{prefix}-rollback.json')['status'] == 'passed', 'paired rollback gate failed'
    assert read(f'{prefix}-rollback.json')['rootConfigurationUnchanged']
    assert read(f'{prefix}-rollback-default-client.json')['status'] == 'passed', 'original default client compatibility failed'
    assert read(f'{prefix}-rollback-default-client.json')['rootConfigurationUnchanged']
    assert read('browser/result.json')['status'] == 'passed', 'browser gate failed'
    return manifest


def storage(host, case):
    base = case['base']
    assert base.startswith('/tmp/quickstart-multi-lan/m10-')
    paths = [base + f'/rootfs/etc/quickstart/networks/capacity{i}/traffic-insights.json'
             for i in range(1, case['lanCount'] + 1)]
    command = ['wc', '-c'] + paths
    result = subprocess.run(['ssh', '-o', 'BatchMode=yes', 'root@' + host, shlex.join(command)],
                            capture_output=True, text=True, check=True, timeout=30)
    sizes = {line.split()[1]: int(line.split()[0]) for line in result.stdout.splitlines()
             if len(line.split()) == 2 and line.split()[1] != 'total'}
    assert len(sizes) == case['lanCount'] and all(0 < size <= 2 * 1024 * 1024 for size in sizes.values()), 'history file bound'
    command = ['du', '-sk'] + [base + f'/engine-{i}' for i in range(1, case['lanCount'] + 1)]
    result = subprocess.run(['ssh', '-o', 'BatchMode=yes', 'root@' + host, shlex.join(command)],
                            capture_output=True, text=True, check=True, timeout=30)
    return {'historyBytes': sizes, 'engineDirectoryKiB': result.stdout.splitlines()}


def summarize(value, arch):
    assert value['status'] == 'passed' and value['rootConfigurationUnchanged'], value.get('error')
    assert value['rootConfigurationBefore'] == value['rootConfigurationAfter']
    assert value['warmupElapsedSeconds'] >= 1800 and value['soakElapsedSeconds'] >= 86400
    assert sorted(case['lanCount'] for case in value['cases']) == [1, 2, 4]
    assert all(case['cleaned'] for case in value['cases']), 'owned namespace cleanup incomplete'
    rows = [json.loads(line) for line in (folder / f'{prefix}-capacity-{arch}.samples.jsonl').read_text().splitlines()]
    max_rss, max_bpf, max_cpu, last = {}, 0, 0, {}
    for row in rows:
        for name, process in row['processes'].items():
            max_rss[name] = max(max_rss.get(name, 0), process['rssKiB'])
            max_bpf = max(max_bpf, process['bpfBytes'])
        count = row['lanCount']
        previous = last.get(count)
        if previous and not row['loadProbe'] and not previous['loadProbe']:
            elapsed = row['cpuSampleMonotonicSeconds'] - previous['cpuSampleMonotonicSeconds']
            if elapsed >= 30:
                ticks = sum(process['ticks'] - previous['processes'][name]['ticks']
                            for name, process in row['processes'].items())
                max_cpu = max(max_cpu, ticks / elapsed)
        last[count] = row
    assert max_cpu <= 25 and max_bpf <= 32 * 1024 * 1024
    assert all(rss <= (128 if name == 'backend.pid' else 64) * 1024 for name, rss in max_rss.items())
    largest = next(case for case in value['cases'] if case['lanCount'] == 4)
    return {'host': value['host'], 'soakHours': value['soakElapsedSeconds'] / 3600,
            'maxIdleCpuPercent': max_cpu, 'peakRssKiB': max_rss, 'peakBpfBytes': max_bpf,
            'warmApiP95Ms': {str(case['lanCount']): max(api['p95Ms'] for api in case['warm']['api'])
                             for case in value['cases']},
            'packetGuardChecks': largest['packetGuardChecks'], 'rootConfigurationUnchanged': True,
            'storage': storage(value['host'], largest)}


def stop_other_runners():
    for runner in read(f'{prefix}-runners.json')['runners']:
        path = pathlib.Path('/proc') / str(runner['pid']) / 'cmdline'
        if not path.exists():
            continue
        command = path.read_bytes().split(b'\0')
        if b'scripts/ops/multi-lan-capacity-probe.py' in command and runner['host'].encode() in command:
            os.kill(runner['pid'], signal.SIGTERM)


def finalize(states):
    manifest = verify_candidate()
    summary = {arch: summarize(state, arch) for arch, state in states.items()}
    now = datetime.datetime.now(datetime.timezone.utc).isoformat()
    write(folder / f'{prefix}-summary.json', {'status': 'passed', 'completedAt': now, 'hosts': summary})
    ledger_path = folder.parent / 'execution.json'
    ledger = json.loads(ledger_path.read_text())
    assert all(item['status'] == 'passed' for item in ledger['milestones'][:-1]), 'earlier milestone incomplete'
    item = ledger['milestones'][-1]
    item.update(status='passed', completedAt=now)
    item.pop('failedGate', None)
    ledger['currentMilestone'] = 'complete'
    write(ledger_path, ledger)
    result = read('result.json')
    result.update(status='passed', completedAt=now, runningGates=[], remainingRequired=[], failedChecks=[])
    result['passedChecks'].append(f'optimization {args.optimization}: both-host full-application 1/2/4 LAN capacity, 30-minute warmup, actual >=24-hour soak and bounded history files')
    write(folder / 'result.json', result)
    matrix = read(f'{prefix}-support-matrix.json')
    matrix.update(status='passed_with_documented_scope_limits', completedAt=now)
    matrix.pop('remainingGate', None)
    write(folder / f'{prefix}-support-matrix.json', matrix)
    manifest.update(status='validated_candidate', validatedAt=now)
    write(folder / f'{prefix}-candidate-manifest.json', manifest)
    (folder / 'progress.md').write_text(f'# M10 已通过\n\nM01–M10 在已列支持范围内全部通过。用户授权的第 {args.optimization} 轮统一修复了旧规则启动、持续恢复和改绑的网络隔离，未增加普通用户操作。\n\n双机完整应用均完成 30 分钟预热和至少 24 小时真实稳定性，容量为 1/2/4 LAN、每 LAN 256 台设备；预算未放宽。升级回退、IPv4/IPv6 数据包、批处理和失败回滚证据见本目录。\n\n资源汇总见 {prefix}-summary.json，最终场景矩阵见 {prefix}-support-matrix.json，候选身份见 {prefix}-candidate-manifest.json。测试环境所属命名空间已清理，原网络配置一致。未提交、推送或发布制品。\n')
    milestones = root / 'docs/archive/product/multi-lan-implementation-milestones.md'
    text = milestones.read_text().splitlines()
    text[2] = '状态：M01–M10 已完成；最终结果、支持范围和证据见 [执行记录](../evidence/multi-lan/execution.json) 与 M10 目录。未提交、推送或发布制品。'
    milestones.write_text('\n'.join(text) + '\n')
    return summary


try:
    while True:
        states = {arch: read(f'{prefix}-capacity-{arch}.json') for arch in ('arm', 'x86')}
        for arch, state in states.items():
            age = time.time() - (folder / f'{prefix}-capacity-{arch}.json').stat().st_mtime
            if state['status'] not in ('passed', 'failed') and age > 300:
                stop_other_runners()
                raise RuntimeError('capacity runner stopped publishing evidence: ' + arch)
        failed = {arch: state for arch, state in states.items() if state['status'] == 'failed'}
        if failed:
            stop_other_runners()
            raise RuntimeError({arch: state.get('error') for arch, state in failed.items()})
        if all(state['status'] == 'passed' for state in states.values()):
            print(json.dumps({'status': 'passed', 'hosts': finalize(states)}, ensure_ascii=False), flush=True)
            break
        print(json.dumps({'status': 'waiting', 'hosts': {arch: {'status': state['status'],
                         'warmupSeconds': state.get('warmupElapsedSeconds', 0),
                         'soakSeconds': state.get('soakElapsedSeconds', 0)}
                         for arch, state in states.items()}}, ensure_ascii=False), flush=True)
        if not args.watch:
            break
        time.sleep(60)
except BaseException as error:
    if isinstance(error, KeyboardInterrupt):
        raise
    write(folder / f'{prefix}-finish-error.json', {'status': 'failed', 'error': repr(error),
          'at': datetime.datetime.now(datetime.timezone.utc).isoformat()})
    result = read('result.json')
    result.update(status='failed', stopReason='authorized optimization round did not pass final acceptance',
                  finalizationError=repr(error))
    write(folder / 'result.json', result)
    ledger_path = folder.parent / 'execution.json'
    ledger = json.loads(ledger_path.read_text())
    ledger['milestones'][-1].update(status='failed', blockedReason='additional_round_limit_reached',
                                  failedGate=repr(error))
    write(ledger_path, ledger)
    raise
