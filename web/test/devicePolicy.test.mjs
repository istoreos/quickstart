import assert from 'node:assert/strict'
import test from 'node:test'

import { readFile } from 'node:fs/promises'
import { normalizeDhcpHostname, internetPathStateLabel, policyAvailable, policyErrorLabel, policyLabelsFromPolicy, policyMacSuffix, policyUnavailableReason, validDhcpHostname } from '../src/pages/device/devicePolicy.ts'

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
    assert.equal(policyErrorLabel('rolled_back'), '应用失败，原设置已恢复，可以重试')
    assert.equal(policyErrorLabel('recovery_required'), '自动恢复未完成，请按提示处理')
    assert.equal(policyErrorLabel('unknown', 'details'), 'details')
})

test('DHCP hostname accepts one portable ASCII label and rejects display names', () => {
    assert.equal(normalizeDhcpHostname(' Living-Room-TV '), 'living-room-tv')
    assert.equal(validDhcpHostname('living-room-tv'), true)
    assert.equal(validDhcpHostname(''), true)
    for (const value of ['客厅电视', 'living room', 'living_room', '-router', 'router-', 'router.local', "router'", 'a'.repeat(64), '😀']) {
        assert.equal(validDhcpHostname(value), false, value)
    }
})

test('internet path effect states use user-facing language', () => {
    assert.equal(internetPathStateLabel('pending_renewal'), '等待设备重新获取地址')
    assert.equal(internetPathStateLabel('lease_observed_unverifiable'), '已观察到新租约，终端网关与 DNS 无法验证')
    assert.equal(internetPathStateLabel('unverifiable'), '服务器配置可读，终端效果无法验证')
    assert.equal(internetPathStateLabel('observation_error'), '暂时无法读取续租状态')
    assert.equal(internetPathStateLabel('failed'), '配置应用失败')
    assert.notEqual(internetPathStateLabel('active'), '已生效')
})

test('device policy panel exposes internet paths without raw DHCP tags', async () => {
    const source = await readFile(new URL('../src/pages/device/components/devicePolicyPanel.vue', import.meta.url), 'utf8')
    assert.match(source, /上网路线/)
    assert.match(source, /deviceNetworkPolicyV2/)
    assert.doesNotMatch(source, /DHCP 标签|tagName|dhcpOption/)
})
