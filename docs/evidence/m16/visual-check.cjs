const { chromium } = require('playwright')
const target = 'http://192.168.9.215/cgi-bin/luci/admin/quickstart/devicemanagement'
const viewports = [['desktop-1440x900',1440,900],['compact-1024x768',1024,768],['mobile-390x844',390,844]]
;(async () => {
    const browser = await chromium.launch({ headless: true }); const page = await browser.newPage({ viewport: { width: 1440, height: 900 } }); await page.goto(target, { waitUntil: 'domcontentloaded' })
    const username = page.locator('input[name="luci_username"]'); if (await username.count()) { await username.fill('root'); await page.locator('input[name="luci_password"]').fill(''); await page.locator('button[type="submit"], input[type="submit"]').click() }
    await page.waitForURL(/quickstart\/devicemanagement/, { timeout: 15000 }); await page.getByRole('button', { name: '全局设置' }).click(); await page.getByRole('button', { name: '浮动网关' }).click(); await page.getByText('尚未安装浮动网关组件').waitFor({ timeout: 15000 })
    for (const [name,width,height] of viewports) { await page.setViewportSize({ width, height }); const body = await page.locator('body').innerText(); const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1); if (overflow || !body.includes('立即安装') || /\bmain\b|\bfallback\b/.test(body)) throw new Error(`${name}: overflow=${overflow}`); await page.screenshot({ path: `docs/evidence/m16/${name}.png`, fullPage: true }) }
    await browser.close()
})().catch(error => { console.error(error); process.exitCode = 1 })
