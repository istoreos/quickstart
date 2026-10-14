import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('M20 keeps advanced network tools out of the default six columns', async () => {
    const list = await readFile(new URL('../src/pages/device/deviceCenterList.vue', import.meta.url), 'utf8')
    const advanced = await readFile(new URL('../src/pages/device/components/advancedNetworkTools.vue', import.meta.url), 'utf8')
    assert.match(list, /<AdvancedNetworkTools/)
    assert.match(advanced, /<details class="advanced-tools"/)
    assert.match(advanced, /高级网络工具/)
    assert.doesNotMatch(list.match(/<thead>[\s\S]*?<\/thead>/)?.[0] || '', /DNS|IPv6|管理入口|审计/)
})

test('M20 presents capability truth and requires import dry-run before confirmation', async () => {
    const source = await readFile(new URL('../src/pages/device/components/advancedNetworkTools.vue', import.meta.url), 'utf8')
    assert.match(source, /当前固件尚未验证，未展示假控件/)
    assert.match(source, /不会扫描其他地址，也不会跟随跳转/)
    assert.match(source, /IMPORT_PLAN/)
    assert.match(source, /importPlan/)
    assert.match(source, /planChecksum:importPlan\.value\.checksum/)
    assert.match(source, /expectedGroupVersion:importPlan\.value\.groupVersion/)
    assert.match(source, /window\.confirm/)
    assert.match(source, /原配置已回滚/)
    assert.match(source, /新设备与网关切换通知/)
    assert.match(source, /WEBHOOK/)
    assert.match(source, /new_device','gateway_failover/)
    assert.match(source, /address_conflict/)
})

test('M20 client exposes bounded probe audit and policy bundle endpoints', async () => {
    const source = await readFile(new URL('../src/request/request.ts', import.meta.url), 'utf8')
    for (const endpoint of ['advanced-network', 'management-probe', 'network-webhook', 'policy-bundle', 'policy-import/plan', 'policy-import/apply']) assert.match(source, new RegExp(endpoint.replace('/', '\\/')))
})
