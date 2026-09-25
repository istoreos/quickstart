const { chromium } = require('playwright')

const target = 'http://192.168.9.215/cgi-bin/luci/admin/quickstart/devicemanagement'

;(async () => {
    const browser = await chromium.launch({ headless: true })
    const page = await browser.newPage()
    await page.goto(target, { waitUntil: 'domcontentloaded' })
    const username = page.locator('input[name="luci_username"]')
    if (await username.count()) {
        await username.fill('root')
        await page.locator('input[name="luci_password"]').fill('')
        await page.locator('button[type="submit"], input[type="submit"]').click()
    }
    await page.waitForURL(/quickstart\/devicemanagement/, { timeout: 15000 })
    const result = await page.evaluate(async () => {
        const get = async path => (await (await fetch(path, { cache: 'no-store' })).json()).result
        const post = async (path, body) => (await (await fetch(path, {
            method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
        })).json()).result
        const targets = await get('/cgi-bin/luci/istore/lanctrl/v2/gateway-targets/')
        const inventory = await get('/cgi-bin/luci/istore/lanctrl/v2/devices/')
        const device = inventory.devices.find(item => item.identity?.scope === 'persistent' && item.mac)
        const probe = await post('/cgi-bin/luci/istore/lanctrl/v2/gateway-assignment/plan/', {
            action: 'assign', deviceId: device.deviceId, targetId: 'default',
        })
        const currentTargetId = probe.currentTargetId === 'unknown' ? 'default' : probe.currentTargetId
        const plan = await post('/cgi-bin/luci/istore/lanctrl/v2/gateway-assignment/plan/', {
            action: 'assign', deviceId: device.deviceId, targetId: currentTargetId,
        })
        const applied = await post('/cgi-bin/luci/istore/lanctrl/v2/gateway-assignment/apply/', {
            action: 'assign', deviceId: device.deviceId, targetId: currentTargetId, expectedVersion: plan.version,
        })
        const references = await get(`/cgi-bin/luci/istore/lanctrl/v2/gateway-target-references/?targetId=${encodeURIComponent(currentTargetId)}`)
        return {
            targetKinds: targets.targets.map(item => item.kind), targetCount: targets.targets.length,
            leaksInternalTag: /tagName|t_auto_/.test(JSON.stringify(targets)),
            planCanApply: plan.canApply, planChanges: plan.changes.length,
            applyChanged: applied.changed, effectState: applied.effectState,
            referenceCount: references.references.length,
        }
    })
    console.log(JSON.stringify(result, null, 2))
    if (result.leaksInternalTag || !result.planCanApply || result.planChanges !== 0 || result.applyChanged) throw new Error('gateway API acceptance failed')
    await browser.close()
})().catch(error => {
    console.error(error)
    process.exitCode = 1
})
