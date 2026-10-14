const { chromium } = require('playwright')

const target = 'http://192.168.9.215/cgi-bin/luci/admin/quickstart/devicemanagement'
const viewports = [
    ['desktop-1440x900', 1440, 900],
    ['compact-1024x768', 1024, 768],
    ['mobile-390x844', 390, 844],
]

;(async () => {
    const browser = await chromium.launch({ headless: true })
    const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
    await page.goto(target, { waitUntil: 'domcontentloaded' })
    const username = page.locator('input[name="luci_username"]')
    if (await username.count()) {
        await username.fill('root')
        await page.locator('input[name="luci_password"]').fill('')
        await page.locator('button[type="submit"], input[type="submit"]').click()
    }
    await page.waitForURL(/quickstart\/devicemanagement/, { timeout: 15000 })
    await page.locator('#device-center-title').waitFor({ timeout: 15000 })
    await page.locator('.device-table tbody tr:visible').first().click()
    await page.waitForSelector('.policy-panel', { timeout: 15000 })
    for (const [name, width, height] of viewports) {
        await page.setViewportSize({ width, height })
        const body = await page.locator('body').innerText()
        const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1)
        const selectUsable = await page.locator('.policy-panel select').first().isVisible()
        if (!body.includes('上网路线') || body.includes('DHCP 标签') || overflow || !selectUsable) {
            throw new Error(`${name}: label=${body.includes('上网路线')} raw=${body.includes('DHCP 标签')} overflow=${overflow} select=${selectUsable}`)
        }
        await page.screenshot({ path: `docs/evidence/m15/${name}.png`, fullPage: true })
    }
    await page.close()
    await browser.close()
})().catch(error => {
    console.error(error)
    process.exitCode = 1
})
