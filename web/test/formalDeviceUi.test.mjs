import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

test('M37 device details use four summary-first sections and have no classic escape hatch', async () => {
    const source = await readFile(new URL('../src/pages/device/deviceCenterList.vue', import.meta.url), 'utf8')
    for (const label of ['概览', '设备资料', '网络与上网', '使用管理', '识别与技术信息', '添加尚未上线的设备']) assert.match(source, new RegExp(label))
    const tabs = source.match(/const detailTabs[\s\S]*?\]\)/)?.[0] || ''
    assert.deepEqual([...tabs.matchAll(/id: '([^']+)'/g)].map(match => match[1]), ['overview', 'profile', 'network', 'management'])
    assert.match(source, /tabindex="0"/)
    assert.match(source, /@keydown\.esc/)
    assert.match(source, /DeviceProfileEditor/)
    assert.match(source, /DeviceUsagePolicyEditor/)
    assert.doesNotMatch(source, /经典列表|use-legacy|DeviceAliasEditor|DeviceClassificationEditor/)
})

test('M37 single-device usage management previews schedules and quotas without duplicating quota controls', async () => {
    const source = await readFile(new URL('../src/pages/device/components/deviceUsagePolicyEditor.vue', import.meta.url), 'utf8')
    for (const label of ['休息时段', '流量额度', '当前生效', '规则来源', '下次变化', '预览影响', '确认并应用', '自动跨到第二天']) assert.match(source, new RegExp(label))
    assert.match(source, /deviceGroupsV2\.GET/)
    assert.match(source, /action:'set_device_policy'/)
    assert.match(source, /expectedVersion:version\.value/)
    assert.match(source, /previewing/)

    const traffic = await readFile(new URL('../src/pages/device/components/trafficInsightsPanel.vue', import.meta.url), 'utf8')
    assert.match(traffic, /quotaEditable/)
})

test('M30 profile editor supports brand type automatic recommendation and all 30 manual icons', async () => {
    const source = await readFile(new URL('../src/pages/device/components/deviceProfileEditor.vue', import.meta.url), 'utf8')
    for (const label of ['品牌（可选）', '设备类型', '自动推荐', '自行选择', '恢复全部自动推荐']) assert.match(source, new RegExp(label))
    assert.match(source, /deviceIconKeys/)
    assert.match(source, /iconMode/)
    assert.match(source, /iconKey/)
    assert.match(source, /role="radiogroup"/)
})

test('M30 LAN settings use read-only role guidance local degradation and draft-preserving installation', async () => {
    const source = await readFile(new URL('../src/pages/device/lanSettingsPanel.vue', import.meta.url), 'utf8')
    for (const label of ['只读引导', '网络服务', '上网路线', '规则台账', 'DNS 跟随路线', '其他设置仍可使用', '发现现有设备设置', '确认接管现有设置']) assert.match(source, new RegExp(label))
    assert.match(source, /routerContextV2/)
    assert.match(source, /lanDeviceMigrationV2\.PLAN/)
    assert.match(source, /lanDeviceMigrationV2\.APPLY/)
    assert.match(source, /!migrationPlan\.canApply/)
    assert.match(source, /capabilityActionV2\.PLAN/)
    assert.match(source, /sessionStorage/)
    assert.match(source, /rememberDraft/)
    assert.doesNotMatch(source, /DHCP 标签|option 3|option 6|eqos|UCI/)
})

test('M33 Internet Path editor uses the domain plan and apply contract without DHCP implementation fields', async () => {
    const source = await readFile(new URL('../src/pages/device/lanSettingsPanel.vue', import.meta.url), 'utf8')
    for (const label of ['编辑', '影响范围', '替代路线', '受影响设备', '受影响分组', '确认应用']) assert.match(source, new RegExp(label))
    assert.match(source, /gatewayTargetsV2\.GET/)
    assert.match(source, /gatewayTargetV2\.PLAN/)
    assert.match(source, /gatewayTargetV2\.APPLY/)
    assert.match(source, /routeEditability/)
    assert.doesNotMatch(source, /dhcpTagsConfig|tagName|dhcpOption/)
})

test('M35 DHCP settings expose a safe pool plan and recovery path', async () => {
    const source = await readFile(new URL('../src/pages/device/lanSettingsPanel.vue', import.meta.url), 'utf8')
    for (const label of ['地址池起始地址', '地址池结束地址', '地址租期', '关闭后预计影响设备', '恢复路径', '我已了解影响，继续']) assert.match(source, new RegExp(label))
    assert.match(source, /lanDhcpSettingsV2\.GET/)
    assert.match(source, /lanDhcpSettingsV2\.PLAN/)
    assert.match(source, /lanDhcpSettingsV2\.APPLY/)
    assert.doesNotMatch(source, /dhcpGatewayConfig\.POST/)
})

test('M30 detail policy shows desired applied observed states and locks route only by DHCP authority', async () => {
    const source = await readFile(new URL('../src/pages/device/components/devicePolicyPanel.vue', import.meta.url), 'utf8')
    for (const label of ['想要的路线', '路由器配置', '实际生效情况', 'routeEditability', '安装限速服务', '启用限速服务']) assert.match(source, new RegExp(label))
    assert.match(source, /routeLocked/)
    assert.match(source, /sessionStorage/)
    assert.doesNotMatch(source, /tagName|dhcpOption|option 3|option 6|eqos|UCI/)
})

test('M34 network and restriction writes require visible plan before apply', async () => {
    const source = await readFile(new URL('../src/pages/device/components/devicePolicyPanel.vue', import.meta.url), 'utf8')
    for (const label of ['确认这次修改', '预计重载', '失败时恢复原设置', '确认应用']) assert.match(source, new RegExp(label))
    assert.match(source, /deviceNetworkPolicyV2\.PLAN/)
    assert.match(source, /deviceNetworkPolicyV2\.APPLY/)
    assert.match(source, /deviceRestrictionsV2\.PLAN/)
    assert.match(source, /deviceRestrictionsV2\.APPLY/)
    assert.doesNotMatch(source, /devicePolicyV2\.POST|deviceNetworkPolicyV2\.POST/)
})
