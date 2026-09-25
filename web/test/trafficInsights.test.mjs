import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('M19 keeps historical usage in device details and out of list columns', async () => {
    const details = await readFile(new URL('../src/pages/device/components/trafficInsightsPanel.vue', import.meta.url), 'utf8')
    const list = await readFile(new URL('../src/pages/device/deviceCenterList.vue', import.meta.url), 'utf8')
    assert.match(details, /历史用量/)
    assert.match(details, /今天/)
    assert.match(details, /本周/)
    assert.match(details, /本月/)
    assert.match(list, /<TrafficInsightsPanel/)
    assert.doesNotMatch(list.match(/device-card__facts[\s\S]*?<\/div>\s*<\/article>/)?.[0] || '', /历史用量|流量额度/)
})

test('M19 clearly separates quota from speed limits and exposes safe optional capability', async () => {
    const source = await readFile(new URL('../src/pages/device/components/trafficInsightsPanel.vue', import.meta.url), 'utf8')
    assert.match(source, /不是速度上限或定时限速/)
    assert.match(source, /仅提醒/)
    assert.match(source, /暂停联网/)
    assert.match(source, /硬件卸载限制/)
    assert.match(source, /每 5 分钟最多写入一次/)
    assert.doesNotMatch(source, /关闭硬件卸载|disable.*offload/i)
})
