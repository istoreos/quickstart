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
    await page.locator('.device-table tbody tr:visible').first().click()
    const editor = page.locator('.alias-editor')
    await editor.waitFor({ timeout: 10000 })
    const input = editor.locator('input[maxlength="64"]')
    const originalAlias = await input.inputValue()
    await input.fill('客厅网络设备')
    await editor.locator('button').last().click()
    await page.getByText('设备备注已更新').waitFor({ timeout: 10000 })

    const results = []
    for (const viewport of viewports) {
        await page.setViewportSize({ width: viewport.width, height: viewport.height })
        const overflowInfo = await page.evaluate(() => ({
            overflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
            clientWidth: document.documentElement.clientWidth,
            scrollWidth: document.documentElement.scrollWidth,
            offenders: [...document.querySelectorAll('body *')].map(element => {
                const rect = element.getBoundingClientRect()
                return { tag: element.tagName, className: typeof element.className === 'string' ? element.className : '', parentClass: typeof element.parentElement?.className === 'string' ? element.parentElement.className : '', href: element.getAttribute('href') || '', text: (element.textContent || '').trim().slice(0, 40), left: Math.round(rect.left), right: Math.round(rect.right), width: Math.round(rect.width) }
            }).filter(rect => rect.right > document.documentElement.clientWidth + 1 || rect.left < -1).sort((left, right) => right.right - left.right).slice(0, 8),
        }))
        const aliasVisible = await page.getByRole('heading', { name: '客厅网络设备' }).isVisible()
        await page.screenshot({ path: `docs/evidence/m13/${viewport.name}-${viewport.width}x${viewport.height}.png`, fullPage: true })
        results.push({ viewport: viewport.name, ...overflowInfo, aliasVisible })
    }

    await input.fill(originalAlias)
    await editor.locator('button').last().click()
    await page.waitForTimeout(1000)
    console.log(JSON.stringify(results))
    if (results.some(result => result.overflow || !result.aliasVisible)) throw new Error('visual acceptance failed')
    await browser.close()
})().catch(error => {
    console.error(error)
    process.exitCode = 1
})
