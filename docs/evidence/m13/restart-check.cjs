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
    const prepared = await page.evaluate(async () => {
        const inventory = await (await fetch('/cgi-bin/luci/istore/lanctrl/v2/devices/')).json()
        const device = inventory.result.devices.find(item => item.identity?.scope === 'persistent' && item.mac)
        const path = `/cgi-bin/luci/istore/lanctrl/v2/device-profile/?deviceId=${encodeURIComponent(device.deviceId)}`
        const original = (await (await fetch(path)).json()).result.profile
        const response = await fetch('/cgi-bin/luci/istore/lanctrl/v2/device-profile/', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ deviceId: device.deviceId, action: 'patch', alias: 'M13 重启验收' }),
        })
        const applied = await response.json()
        if (applied.result?.profile?.alias !== 'M13 重启验收') throw new Error('profile write failed')
        return { deviceId: device.deviceId, originalAlias: original.alias || '' }
    })
    console.log('READY')
    await page.waitForTimeout(15000)
    const result = await page.evaluate(async prepared => {
        const path = `/cgi-bin/luci/istore/lanctrl/v2/device-profile/?deviceId=${encodeURIComponent(prepared.deviceId)}`
        let profile
        for (let attempt = 0; attempt < 15; attempt += 1) {
            try {
                profile = (await (await fetch(`${path}&t=${Date.now()}`, { cache: 'no-store' })).json()).result.profile
                if (profile?.alias === 'M13 重启验收') break
            } catch (_) {}
            await new Promise(resolve => setTimeout(resolve, 1000))
        }
        if (profile?.alias !== 'M13 重启验收') throw new Error('alias did not survive service restart')
        const restored = await (await fetch('/cgi-bin/luci/istore/lanctrl/v2/device-profile/', {
            method: 'POST', headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ deviceId: prepared.deviceId, action: 'patch', alias: prepared.originalAlias }),
        })).json()
        return { persisted: true, restored: (restored.result?.profile?.alias || '') === prepared.originalAlias }
    }, prepared)
    console.log(JSON.stringify(result))
    if (!result.restored) throw new Error('original alias was not restored')
    await browser.close()
})().catch(error => {
    console.error(error)
    process.exitCode = 1
})
