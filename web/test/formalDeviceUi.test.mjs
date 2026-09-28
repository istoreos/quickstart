import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import test from 'node:test'

const require = createRequire(import.meta.url)
const PO = require('pofile')

test('M40 English catalog covers every device-management message', async () => {
    const source = await readFile(new URL('../translations/en/app.po', import.meta.url), 'utf8')
    const catalog = PO.parse(source)
    const missing = catalog.items.filter(item =>
        item.references?.some(reference => reference.includes('src/pages/device/')) &&
        ((!item.msgstr?.length || item.msgstr.every(value => !value.trim())) || item.flags?.fuzzy),
    )
    assert.deepEqual(missing.map(item => item.msgid), [])

    const translations = JSON.parse(await readFile(new URL('../public/luci-static/quickstart/i18n/en.json', import.meta.url), 'utf8')).en
    assert.equal(translations['上网路线'], 'Internet path')
    assert.equal(translations['网络与上网'], 'Network & Internet')
    assert.equal(translations['使用管理'], 'Usage controls')
    assert.equal(translations['计划与额度'], 'Schedule & quota')
    assert.equal(translations['已设置'], 'Configured')
    assert.equal(translations['自动推荐'], 'Automatic')
    assert.equal(translations['自行选择'], 'Choose manually')
})

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
    assert.match(source, /icon\.assetKey/)
    assert.match(source, /isDeviceIconKey/)
    assert.doesNotMatch(source, /icon\.resolvedKey\s*\|\|/)
})

test('M40 device management isolates semantic headers from LuCI theme decoration', async () => {
    const source = await readFile(new URL('../src/pages/device/index.vue', import.meta.url), 'utf8')
    assert.match(source, /\.device-management header::after/)
    assert.match(source, /content:\s*none!important/)
    assert.match(source, /pointer-events:\s*none!important/)
    assert.match(source, /\.device-management \.group-header/)
    assert.match(source, /\.device-management \.lan-settings > header/)
})

test('M40 follow-up keeps advanced workflows progressive and mobile rules readable', async () => {
    const groups = await readFile(new URL('../src/pages/device/deviceGroupsPanel.vue', import.meta.url), 'utf8')
    assert.match(groups, /class="advanced-group"/)
    assert.match(groups, /已选择.*draft\.members\.length.*台设备/)
    assert.doesNotMatch(groups, /draft\.members\.length \}\} \/ 2048/)
    assert.match(groups, /gatewayTargetLabel\(target\)/)

    const settings = await readFile(new URL('../src/pages/device/lanSettingsPanel.vue', import.meta.url), 'utf8')
    assert.match(settings, /<details[^>]+class="migration-card"/)
    assert.match(settings, /gatewayTargetLabel/)

    const floating = await readFile(new URL('../src/pages/device/components/floatingGatewayWizard.vue', import.meta.url), 'utf8')
    assert.match(floating, /configured && !editing/)
    assert.match(floating, /startEditing/)

    const rules = await readFile(new URL('../src/pages/device/networkRulesHub.vue', import.meta.url), 'utf8')
    assert.match(rules, /class="rule-cards"/)
    assert.match(rules, /ruleSummary\(rule\.summary\)/)
    assert.match(rules, /\.rules-table-wrap\{display:none\}/)
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
    assert.doesNotMatch(source, /DHCP 标签|option 3|option 6|UCI/)
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
    assert.match(source, /currentPolicyLabels/)
    assert.match(source, /labels\.includes\('route'\)/)
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

test('P0-P2 rate limits expose effect truth plans scheduled limits and progressive providers', async () => {
    const [policy, settings, groups, usage, types] = await Promise.all([
        readFile(new URL('../src/pages/device/components/devicePolicyPanel.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/pages/device/lanSettingsPanel.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/pages/device/deviceGroupsPanel.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/pages/device/components/deviceUsagePolicyEditor.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/pages/device/devicePolicy.ts', import.meta.url), 'utf8'),
    ])
    for (const token of ['rateLimitStateLabel', 'executionNode', 'addressState', 'offloadState', 'ipv6State']) assert.match(policy + types, new RegExp(token))
    for (const token of ['rateLimitSettingsV2.PLAN', 'rateLimitSettingsV2.APPLY', 'pendingSpeedPlan', '高级：执行方式与迁移', '智能设备限速（推荐）']) assert.match(settings, new RegExp(token))
    for (const source of [groups, usage]) {
        assert.match(source, /scheduleAction/)
        assert.match(source, /action:draft\.scheduleAction/)
        assert.match(source, /时段限速/)
    }
    assert.doesNotMatch(policy + settings, /UCI|eBPF|\/api\/traffic/)

    const catalog = JSON.parse(await readFile(new URL('../public/luci-static/quickstart/i18n/en.json', import.meta.url), 'utf8')).en
    for (const message of [
        '在指定时段执行规则', '时段内执行', '限制速度', '不设置时段规则',
        '高级：执行方式与迁移', '执行方式', '兼容模式（按 IPv4）', '智能设备限速（推荐）',
        '总带宽用于计算设备限速队列，不会自动平均分配给每台设备。',
        '预览限速服务设置', '需先预留当前地址', '当前实现按 IPv4 限速，预留当前地址后规则才不会漂移。',
    ]) {
        assert.equal(typeof catalog[message], 'string', `missing English rate-limit translation: ${message}`)
        assert.ok(catalog[message].length > 0, `empty English rate-limit translation: ${message}`)
        assert.doesNotMatch(catalog[message], /[\u3400-\u9fff]/, `Chinese leaked into English rate-limit copy: ${message}`)
    }
})

test('M51 native speed limits stay simple while failures and Bandix migration remain explicit', async () => {
    const [settings, requests] = await Promise.all([
        readFile(new URL('../src/pages/device/lanSettingsPanel.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/request/request.ts', import.meta.url), 'utf8'),
    ])
    for (const label of [
        '设备限速服务', '智能设备限速（推荐）', '高级：执行方式与迁移',
        '迁移现有 Bandix 规则', '扫描现有规则', '我已手动停止 Bandix，并启动 Quickstart 限速服务',
        '存在不支持或冲突的规则，全部处理完成前不会迁移。', '恢复迁移前设置',
    ]) assert.match(settings, new RegExp(label))
    assert.match(settings, /speedProviderReady/)
    assert.match(settings, /canInstallSpeedProvider/)
    assert.match(settings, /speed\.provider === 'quickstart-native'/)
    assert.match(settings, /bandixProvider\?\.installed && speed\.provider === 'bandix'/)
    assert.match(settings, /speed\.provider === 'eqos' \? speed\.enabled : true/)
    assert.match(settings, /rateLimitMigrationV2\.PLAN/)
    assert.match(settings, /rateLimitMigrationV2\.APPLY/)
    assert.match(settings, /rateLimitMigrationV2\.ROLLBACK/)
    assert.match(requests, /rate-limit-migration\/plan/)
    assert.match(requests, /rate-limit-migration\/apply/)
    assert.match(requests, /rate-limit-migration\/rollback/)
    assert.doesNotMatch(settings, /eBPF|tc qdisc|BPF map/)
})

test('M58 migration and profile tasks distinguish restored from manual recovery', async () => {
    const settings = await readFile(new URL('../src/pages/device/lanSettingsPanel.vue', import.meta.url), 'utf8')
    const profile = await readFile(new URL('../src/pages/device/components/deviceProfileEditor.vue', import.meta.url), 'utf8')
    assert.match(settings, /reason\?\.code==='rolled_back'/)
    assert.match(settings, /reason\?\.code==='recovery_required'/)
    assert.match(profile, /policyErrorLabel\(error\.code/)
})

test('M60 history stays lazy and quota actions remain distinct', async () => {
    const list = await readFile(new URL('../src/pages/device/deviceCenterList.vue', import.meta.url), 'utf8')
    const insights = await readFile(new URL('../src/pages/device/components/trafficInsightsPanel.vue', import.meta.url), 'utf8')
    assert.doesNotMatch(list, /trafficInsightsV2/)
    assert.match(insights, /仅提醒/)
    assert.match(insights, /达到后/)
    assert.match(insights, /value="block"[^>]*>\{\{ \$gettext\('暂停联网'\)/)
    assert.match(insights, /today.*week.*month/s)
})

test('M55 keeps Unicode device aliases separate from portable DHCP hostnames', async () => {
    const [panel, policy, profile] = await Promise.all([
        readFile(new URL('../src/pages/device/components/devicePolicyPanel.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/pages/device/devicePolicy.ts', import.meta.url), 'utf8'),
        readFile(new URL('../src/pages/device/components/deviceProfileEditor.vue', import.meta.url), 'utf8'),
    ])
    assert.match(profile, /设备备注名/)
    assert.match(profile, /例如：客厅电视/)
    assert.match(panel, /与可使用中文的设备备注名分开保存/)
    assert.match(panel, /maxlength="63"/)
    assert.match(policy, /hostname === '' \|\| \/\^\[a-z0-9\]/)
})

test('M56 capability failures stay local and native service preparation preserves the edit task', async () => {
    const [panel, settings, legacySettings] = await Promise.all([
        readFile(new URL('../src/pages/device/components/devicePolicyPanel.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/pages/device/lanSettingsPanel.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/pages/device/configure.vue', import.meta.url), 'utf8'),
    ])
    assert.match(panel, /const routeLocked = computed/)
    assert.match(panel, /:disabled="routeLocked \|\| saving/)
    assert.match(panel, /:disabled="!available\('speed'\)"/)
    assert.match(panel, /v-if="reason\('access'\)"/)
    assert.match(panel, /<button v-else[^>]+@click="toggleAccess"/)
    assert.match(panel, /rememberDraft\(\); saving\.value = 'capability'/)
    assert.match(panel, /尚未保存的限速输入会保留/)
    assert.match(panel, /await load\(\)/)
    const prepare = panel.match(/const prepareSpeedService = async \(\) => \{[\s\S]*?\n\}/)?.[0] || ''
    assert.doesNotMatch(prepare, /saveSpeed|deviceRestrictionsV2\.APPLY/)

    assert.match(settings, /其他设置仍可使用/)
    assert.match(settings, /capabilityActionV2\.PLAN/)
    assert.match(legacySettings, /openMode\('quickstart-netpolicy'\)/)
    assert.doesNotMatch(panel + settings + legacySettings, /app-meta-eqos/)
})

test('M57 route failures provide an actionable default-path recovery without exposing DHCP internals', async () => {
    const [rules, policy] = await Promise.all([
        readFile(new URL('../src/pages/device/networkRulesHub.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/pages/device/devicePolicy.ts', import.meta.url), 'utf8'),
    ])
    for (const label of ['网关不可达', '恢复默认路线', '确认网关设备已开机并接入当前局域网']) assert.match(rules, new RegExp(label))
    assert.match(rules, /rule\.nextAction/)
    assert.match(rules, /repairRule/)
    assert.match(rules, /networkRulesV2\.PLAN/)
    assert.match(rules, /networkRulesV2\.APPLY/)
    assert.match(policy, /gateway_unreachable/)
    assert.match(policy, /address_conflict/)
    assert.doesNotMatch(rules, /DHCP 标签|option 3|option 6|UCI/)
})
