import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

test('M30 primary navigation is the unique devices groups and LAN settings IA', async () => {
    const source = await readFile(new URL('../src/pages/device/index.vue', import.meta.url), 'utf8')
    for (const label of ['设备', '分组与计划', '局域网设置']) assert.match(source, new RegExp(label))
    for (const component of ['DeviceCenterList', 'DeviceGroupsPanel', 'LanSettingsPanel']) assert.match(source, new RegExp(component))
    assert.doesNotMatch(source, /DeviceListVue|legacyMode|经典列表|StaticStateList|SpeedLimitList/)
})

test('M30 rule ledger keeps unified filters and bulk preview without duplicate editors', async () => {
    const source = await readFile(new URL('../src/pages/device/networkRulesHub.vue', import.meta.url), 'utf8')
    for (const label of ['地址预留', '上网路线', '设备限速', '联网权限', '离线或孤立', '规则台账']) assert.match(source, new RegExp(label))
    assert.match(source, /networkRulesV2\.PLAN/)
    assert.match(source, /affectedCount/)
    assert.match(source, /rollbackScope/)
    assert.doesNotMatch(source, /StaticStateList|SpeedLimitList|经典编辑/)
    assert.doesNotMatch(source, /tagName|dhcpOption|DHCP 标签|option 3|eqos|UCI/)
})
