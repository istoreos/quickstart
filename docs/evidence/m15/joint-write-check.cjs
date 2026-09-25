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
        const device = inventory.devices.find(item => item.identity?.scope === 'persistent' && item.mac && item.addresses?.current?.some(address => address.family === 4))
        const address = device.addresses.current.find(item => item.family === 4).address
        const network = await get(`/cgi-bin/luci/istore/lanctrl/v2/device-network-policy/?deviceId=${encodeURIComponent(device.deviceId)}`)
        const applied = await post('/cgi-bin/luci/istore/lanctrl/v2/device-network-policy/', {
            deviceId: device.deviceId,
            static: { enabled: true, assignedIP: address, bindIP: true, hostname: 'm15-accept' },
            targetId: network.policy.path.targetId,
            expectedVersion: network.policy.version,
        })
        return { deviceId: device.deviceId, address, changed: applied.changed, path: applied.policy?.path, static: applied.policy?.static, error: applied.error || null }
    })
    console.log(JSON.stringify(result, null, 2))
    if (!result.changed || result.error || result.path?.effectState !== 'pending_renewal' || result.static?.hostname !== 'm15-accept') {
        throw new Error('M15 joint write acceptance failed')
    }
    await browser.close()
})().catch(error => {
    console.error(error)
    process.exitCode = 1
})
