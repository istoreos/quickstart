import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('M30 exposes one complete group-plan editor with preview and source explanation', async () => {
    const source = await readFile(new URL('../src/pages/device/deviceGroupsPanel.vue', import.meta.url), 'utf8')
    for (const label of ['分组与计划', '在指定时段暂停联网', '自动跨到第二天', '单设备例外始终优先', '查看最终生效结果与来源', '上网路线', '设置速度上限', '设置流量额度', '预览影响']) {
        assert.match(source, new RegExp(label))
    }
    assert.match(source, /effective\.slice\(0,50\)/)
    assert.match(source, /deviceGroupsV2\.POST/)
    assert.match(source, /expectedVersion/)
    assert.doesNotMatch(source, /DHCP\s*标签|tagName|dhcpOption|option 3|eqos|UCI/)
})

test('M30 keeps group controls responsive bounded and in their only primary entry', async () => {
    const source = await readFile(new URL('../src/pages/device/deviceGroupsPanel.vue', import.meta.url), 'utf8')
    assert.match(source, /2048/)
    assert.match(source, /max-height:200px/)
    assert.match(source, /@media\(max-width:640px\)/)
    const index = await readFile(new URL('../src/pages/device/index.vue', import.meta.url), 'utf8')
    assert.match(index, /<DeviceGroupsPanel v-else-if=/)
    const ledger = await readFile(new URL('../src/pages/device/networkRulesHub.vue', import.meta.url), 'utf8')
    assert.doesNotMatch(ledger, /DeviceGroupsPanel/)
})

test('request client uses the versioned device-groups endpoint', async () => {
    const source = await readFile(new URL('../src/request/request.ts', import.meta.url), 'utf8')
    assert.match(source, /deviceGroupsV2/)
    assert.match(source, /\/lanctrl\/v2\/device-groups\//)
})
