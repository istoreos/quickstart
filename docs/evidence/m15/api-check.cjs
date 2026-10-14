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
        const inventory = await get('/cgi-bin/luci/istore/lanctrl/v2/devices/')
        const device = inventory.devices.find(item => item.identity?.scope === 'persistent' && item.mac)
        const policy = await get(`/cgi-bin/luci/istore/lanctrl/v2/device-network-policy/?deviceId=${encodeURIComponent(device.deviceId)}`)
        const raw = JSON.stringify(policy)
        const applied = await post('/cgi-bin/luci/istore/lanctrl/v2/device-network-policy/', {
            deviceId: device.deviceId,
            static: policy.policy.static,
            targetId: policy.policy.path.targetId,
            expectedVersion: policy.policy.version,
        })
        return {
            deviceId: device.deviceId,
            targetCount: policy.policy.targets.length,
            targetNames: policy.policy.targets.map(item => item.name),
            path: policy.policy.path,
            leakedInternalDetails: /tagName|tagTitle|dhcpOption|t_auto_/.test(raw),
            changed: applied.changed,
            effectState: applied.policy?.path?.effectState,
            error: applied.error || null,
        }
    })
    console.log(JSON.stringify(result, null, 2))
    if (result.leakedInternalDetails || result.targetCount < 1 || result.changed || result.error || result.effectState !== 'active') {
        throw new Error('M15 API acceptance failed')
    }
    await browser.close()
})().catch(error => {
    console.error(error)
    process.exitCode = 1
})
