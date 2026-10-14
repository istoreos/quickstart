const { chromium } = require('playwright')
const fs = require('node:fs')

const target = process.env.M40_UI_URL
const sid = process.env.M40_LUCI_COOKIE
const output = process.env.M40_UI_OUTPUT || '/tmp/quickstart-m62-ui'
if (!target || !sid) throw new Error('M40_UI_URL and M40_LUCI_COOKIE are required')
const targetURL = new URL(target)
fs.mkdirSync(output, { recursive: true })

const profiles = [
  { name: 'desktop-zh-light', width: 1440, height: 900, locale: 'zh-CN', colorScheme: 'light' },
  { name: 'tablet-en-dark', width: 1024, height: 768, locale: 'en-US', colorScheme: 'dark' },
  { name: 'mobile-zh-dark', width: 390, height: 844, locale: 'zh-CN', colorScheme: 'dark' },
  { name: 'mobile-en-light', width: 390, height: 844, locale: 'en-US', colorScheme: 'light' },
]

;(async () => {
  const report = { target, generatedAt: new Date().toISOString(), profiles: [], failures: [] }
  const browser = await chromium.launch({ headless: true, args: ['--no-proxy-server'] })
  try {
    for (const profile of profiles) {
      const context = await browser.newContext({ viewport: { width: profile.width, height: profile.height }, locale: profile.locale, colorScheme: profile.colorScheme })
      await context.addCookies([{ name: 'sysauth_http', value: sid, domain: targetURL.hostname, path: '/cgi-bin/luci/' }])
      const page = await context.newPage()
      const consoleErrors = []
      const requestFailures = []
      let auditPayload
      page.on('console', message => { if (message.type() === 'error') consoleErrors.push(message.text()) })
      page.on('pageerror', error => consoleErrors.push(error.message))
      page.on('requestfailed', request => requestFailures.push(`${request.method()} ${request.url()} ${request.failure()?.errorText || ''}`))
      page.on('response', async response => {
        if (response.url().includes('/lanctrl/v2/advanced-network/')) auditPayload = await response.json().catch(() => null)
      })
      await page.goto(target, { waitUntil: 'domcontentloaded', timeout: 20_000 })
      await page.locator('.device-center').waitFor({ timeout: 20_000 })
      const device = page.locator('.device-table tbody tr:visible, .device-card:visible').first()
      await device.waitFor({ timeout: 20_000 })
      const initial = await page.evaluate(() => ({
        heading: document.querySelector('.page-heading h1')?.textContent?.trim(),
        overflowX: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
        internalTerms: (document.querySelector('.device-management')?.textContent || '').match(/DHCP 标签|option [36]|\bUCI\b|\bProvider\b/gi) || [],
        background: getComputedStyle(document.body).backgroundColor,
      }))
      if (initial.overflowX) report.failures.push(`${profile.name}: horizontal overflow`)
      if (initial.internalTerms.length) report.failures.push(`${profile.name}: internal terms visible: ${initial.internalTerms.join(', ')}`)
      if (profile.locale.startsWith('en') && initial.heading?.includes('局域网')) report.failures.push(`${profile.name}: English locale still shows Chinese heading`)

      await page.locator('.filter-panel > summary').click()
      const filterSelects = page.locator('.filter-panel select')
      if (await filterSelects.count() !== 3) report.failures.push(`${profile.name}: expected three composable filters`)
      await filterSelects.nth(0).selectOption('wifi')
      await filterSelects.nth(2).selectOption('none')
      const activeCount = await page.locator('.filter-panel > summary small').textContent()
      if (activeCount?.trim() !== '2') report.failures.push(`${profile.name}: active filter count=${activeCount}`)
      await page.locator('.clear-filters').click()

      await device.click()
      await page.locator('.device-drawer').waitFor()
      await page.evaluate(() => {
        const input = document.querySelector('.device-search input')
        input.value = '__m62_no_such_device__'
        input.dispatchEvent(new Event('input', { bubbles: true }))
      })
      await page.locator('.context-notice').waitFor()
      await page.locator('.context-notice button').click()
      if (await page.locator('.context-notice').count()) report.failures.push(`${profile.name}: clear filters did not restore selected context`)

      const advanced = page.locator('.advanced-tools')
      if (await advanced.getAttribute('open') !== null) report.failures.push(`${profile.name}: advanced audit opened by default`)
      await advanced.locator('summary').click()
      await page.locator('.advanced-body').waitFor({ timeout: 15_000 })
      const result = auditPayload?.result
      if (!result || !Array.isArray(result.events) || result.events.length > 20 || !(result.eventLimit > 0 && result.eventLimit <= 512)) {
        report.failures.push(`${profile.name}: audit response is missing or unbounded`)
      }
      if (/\bunsupported\b|\bnot_installed\b/.test(await advanced.innerText())) report.failures.push(`${profile.name}: raw capability state is visible`)
      await page.screenshot({ path: `${output}/${profile.name}-audit.png`, fullPage: false })
      await page.locator('.device-drawer').evaluate(element => { element.scrollTop = 0 })
      await page.screenshot({ path: `${output}/${profile.name}.png`, fullPage: false })
      await page.locator('.drawer-close').click()
      const focusRestored = await page.evaluate(() => document.activeElement?.matches('.device-table tbody tr,.device-card'))
      if (!focusRestored) report.failures.push(`${profile.name}: focus not restored to opened device`)
      if (consoleErrors.length) report.failures.push(`${profile.name}: ${consoleErrors.length} console error(s)`)
      if (requestFailures.length) report.failures.push(`${profile.name}: ${requestFailures.length} failed request(s)`)
      report.profiles.push({ ...profile, ...initial, auditEvents: result?.events?.length, auditLimit: result?.eventLimit, focusRestored, consoleErrors, requestFailures })
      await context.close()
    }
  } finally {
    await browser.close()
  }
  fs.writeFileSync(`${output}/report.json`, JSON.stringify(report, null, 2))
  console.log(JSON.stringify({ output, profiles: report.profiles.length, failures: report.failures }, null, 2))
  if (report.failures.length) process.exitCode = 1
})().catch(error => { console.error(error); process.exitCode = 1 })
