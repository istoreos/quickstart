const { chromium } = require('playwright')

const target = 'http://192.168.30.1/cgi-bin/luci/admin/quickstart/devicemanagement'
const cookie = process.env.M39_LUCI_COOKIE
if (!cookie) throw new Error('M39_LUCI_COOKIE is required')

const shots = [
  { name: 'desktop-devices', width: 1440, height: 900 },
  { name: 'mobile-devices', width: 390, height: 844 },
]

;(async () => {
  const browser = await chromium.launch({ headless: true, args: ['--no-proxy-server'] })
  for (const shot of shots) {
    const context = await browser.newContext({ viewport: { width: shot.width, height: shot.height } })
    await context.addCookies([{ name: 'sysauth_http', value: cookie, domain: '192.168.30.1', path: '/cgi-bin/luci/' }])
    const page = await context.newPage()
    await page.goto(target, { waitUntil: 'domcontentloaded', timeout: 15000 })
    await page.locator('.device-center').waitFor({ timeout: 15000 })
    await page.locator('.device-table tbody tr:visible, .device-card:visible').first().waitFor({ timeout: 15000 })
    if (await page.locator('input[name="luci_username"]').count()) throw new Error(`${shot.name} is not authenticated`)

    await page.screenshot({ path: `docs/evidence/m39/${shot.name}.png`, fullPage: true })
    const device = shot.width <= 700
      ? page.locator('.device-card').filter({ hasText: '192.168.30.7' }).first()
      : page.locator('.device-table tbody tr').filter({ hasText: '192.168.30.7' }).first()
    await device.click()
    await page.locator('.device-drawer').waitFor()
    await page.locator('.drawer-tabs button').nth(3).click()
    await page.locator('.usage-policy .effective-card').waitFor({ timeout: 15000 })
    await page.screenshot({ path: `docs/evidence/m39/${shot.name}-management.png`, fullPage: true })
    await page.locator('.drawer-close').click()

    await page.locator('.primary-tabs button').nth(1).click()
    await page.locator('.group-panel .evaluation').waitFor({ timeout: 15000 })
    await page.screenshot({ path: `docs/evidence/m39/${shot.name}-groups.png`, fullPage: true })
    await page.locator('.primary-tabs button').nth(2).click()
    await page.locator('.lan-settings .context-card').waitFor({ timeout: 15000 })
    await page.screenshot({ path: `docs/evidence/m39/${shot.name}-settings.png`, fullPage: true })

    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1)
    if (overflow) throw new Error(`${shot.name} has horizontal page overflow`)
    await context.close()
  }
  await browser.close()
})().catch(error => { console.error(error); process.exitCode = 1 })
