const { chromium } = require('playwright')

const target = 'http://192.168.30.1/cgi-bin/luci/admin/quickstart/devicemanagement'
const cookie = process.env.M40_LUCI_COOKIE
if (!cookie) throw new Error('M40_LUCI_COOKIE is required')

const shots = [
  { name: 'desktop', width: 1440, height: 900 },
  { name: 'mobile', width: 390, height: 844 },
]

;(async () => {
  const browser = await chromium.launch({ headless: true, args: ['--no-proxy-server'] })
  for (const shot of shots) {
    const context = await browser.newContext({ viewport: { width: shot.width, height: shot.height } })
    await context.addCookies([{ name: 'sysauth_http', value: cookie, domain: '192.168.30.1', path: '/cgi-bin/luci/' }])
    const page = await context.newPage()
    await page.goto(target, { waitUntil: 'domcontentloaded', timeout: 15_000 })
    await page.locator('.device-center').waitFor({ timeout: 15_000 })
    if (await page.locator('input[name="luci_username"]').count()) throw new Error(`${shot.name} is not authenticated`)

    const primaryText = await page.locator('.primary-tabs').innerText()
    for (const label of ['Devices', 'Groups & schedules', 'LAN settings']) {
      if (!primaryText.includes(label)) throw new Error(`${shot.name} missing English label: ${label}`)
    }
    if (/分组与计划|局域网设置/.test(primaryText)) throw new Error(`${shot.name} has mixed primary navigation language`)
    await page.screenshot({ path: `docs/evidence/m40/${shot.name}-devices.png`, fullPage: true })

    const device = shot.width <= 700
      ? page.locator('.device-card').filter({ hasText: '192.168.30.7' }).first()
      : page.locator('.device-table tbody tr').filter({ hasText: '192.168.30.7' }).first()
    await device.click()
    await page.locator('.drawer-tabs button').nth(3).click()
    await page.locator('.usage-policy .effective-card').waitFor({ timeout: 15_000 })
    const drawerText = await page.locator('.device-drawer').innerText()
    for (const label of ['Usage controls', 'Schedule & quota', 'Effective now', 'Rule source']) {
      if (!drawerText.includes(label)) throw new Error(`${shot.name} missing English drawer label: ${label}`)
    }
    if (/使用管理|计划与额度|当前生效|规则来源/.test(drawerText)) throw new Error(`${shot.name} has mixed drawer language`)
    await page.screenshot({ path: `docs/evidence/m40/${shot.name}-usage.png`, fullPage: true })
    await page.locator('.drawer-close').click()

    await page.locator('.primary-tabs button').nth(1).click()
    await page.locator('.group-panel .evaluation').waitFor({ timeout: 15_000 })
    await page.screenshot({ path: `docs/evidence/m40/${shot.name}-groups.png`, fullPage: true })
    await page.locator('.primary-tabs button').nth(2).click()
    await page.locator('.lan-settings .context-card').waitFor({ timeout: 15_000 })
    const settingsText = await page.locator('.lan-settings').innerText()
    if (/启用设备限速|本机路由|自定义网关|旁路由|浮动网关/.test(settingsText)) {
      throw new Error(`${shot.name} has untranslated built-in settings labels`)
    }
    await page.screenshot({ path: `docs/evidence/m40/${shot.name}-settings.png`, fullPage: true })

    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1)
    if (overflow) throw new Error(`${shot.name} has horizontal page overflow`)
    await context.close()
  }
  await browser.close()
})().catch(error => { console.error(error); process.exitCode = 1 })
