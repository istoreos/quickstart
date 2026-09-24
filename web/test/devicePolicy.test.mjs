import assert from 'node:assert/strict'
import test from 'node:test'

import { policyAvailable, policyErrorLabel, policyLabelsFromPolicy, policyMacSuffix, policyUnavailableReason } from '../src/pages/device/devicePolicy.ts'

const policy = {
    deviceId: 'mac:aa:bb:cc:dd:ee:ff', mac: 'AA:BB:CC:DD:EE:FF', currentIPv4: '192.168.100.20',
    static: { enabled: true, assignedIP: '192.168.100.20', bindIP: true, hostname: '', tagName: '' },
    speed: { enabled: true, uploadSpeed: 100, downloadSpeed: 1000 },
    access: { networkAccess: false },
    capabilities: {
        static: { state: 'available' },
        speed: { state: 'not_installed', reason: 'dependency_not_installed' },
        access: { state: 'disabled' },
    },
}

test('policy capabilities gate only their own controls', () => {
    assert.equal(policyAvailable(policy, 'static'), true)
    assert.equal(policyAvailable(policy, 'speed'), false)
    assert.equal(policyUnavailableReason(policy, 'speed'), '所需组件尚未安装')
    assert.equal(policyUnavailableReason(policy, 'access'), '请先在全局设置中启用该能力')
})

test('policy summaries and dangerous confirmation suffix are deterministic', () => {
    assert.deepEqual(policyLabelsFromPolicy(policy), ['static', 'blocked'])
    assert.equal(policyMacSuffix(policy.mac), 'EEFF')
})

test('policy error codes map to actionable messages', () => {
    assert.equal(policyErrorLabel('conflict'), '该设置与现有规则冲突')
    assert.equal(policyErrorLabel('apply_failed'), '应用失败，原设置已恢复')
    assert.equal(policyErrorLabel('unknown', 'details'), 'details')
})
