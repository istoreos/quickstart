const { chromium } = require('playwright')

const target = 'http://192.168.9.215/cgi-bin/luci/admin/quickstart/devicemanagement'
const viewports = [
    { name: 'desktop', width: 1440, height: 900 },
    { name: 'compact', width: 1024, height: 768 },
    { name: 'mobile', width: 390, height: 844 },
]

;(async () => {
    const browser = await chromium.launch({ headless: true })
    const page = await browser.newPage({ viewport: viewports[0] })
    await page.goto(target, { waitUntil: 'domcontentloaded' })
    const username = page.locator('input[name="luci_username"]')
    if (await username.count()) {
        await username.fill('root')
        await page.locator('input[name="luci_password"]').fill('')
        await page.locator('button[type="submit"], input[type="submit"]').click()
    }
    await page.waitForURL(/quickstart\/devicemanagement/, { timeout: 15000 })
    await page.locator('#device-center-title').waitFor({ timeout: 15000 })

    const results = []
    for (const viewport of viewports) {
        await page.setViewportSize({ width: viewport.width, height: viewport.height })
        if (!(await page.locator('.device-drawer').count())) {
            const device = page.locator('.device-table tbody tr:visible, .device-card:visible').first()
            await device.click()
        }
        await page.locator('.device-drawer').waitFor({ timeout: 10000 })
        const hostname = page.locator('.policy-panel input[maxlength="63"]')
        if (!(await hostname.isVisible())) {
            await page.locator('.policy-panel .policy-card').first().locator('summary').click()
        }
        await hostname.waitFor({ timeout: 10000 })
        await hostname.fill('客厅电视')
        const saveButton = page.locator('.policy-panel .policy-card').first().locator('button')
        const invalidDisabled = await saveButton.isDisabled()
        await hostname.fill('living-room-tv')
        const validDisabled = await saveButton.isDisabled()
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth)
        await page.screenshot({ path: `docs/evidence/m11/${viewport.name}-${viewport.width}x${viewport.height}.png`, fullPage: true })
        results.push({ viewport: viewport.name, invalidDisabled, validDisabled, overflow })
    }
    console.log(JSON.stringify(results))
    await browser.close()
})().catch(error => {
    console.error(error)
    process.exitCode = 1
})
