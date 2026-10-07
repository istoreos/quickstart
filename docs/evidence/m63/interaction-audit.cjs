const { chromium } = require('playwright')
const fs = require('node:fs')

const target = process.env.M40_UI_URL
const sid = process.env.M40_LUCI_COOKIE
const output = process.env.M40_UI_OUTPUT || '/tmp/quickstart-m63-ui'
if (!target || !sid) throw new Error('M40_UI_URL and M40_LUCI_COOKIE are required')
const targetURL = new URL(target)
fs.mkdirSync(output, { recursive: true })

;(async () => {
  const report = { target, generatedAt: new Date().toISOString(), views: [], failures: [] }
  const browser = await chromium.launch({ headless: true, args: ['--no-proxy-server'] })
  try {
    for (const viewport of [{ name: 'desktop', width: 1440, height: 900 }, { name: 'mobile', width: 390, height: 844 }]) {
      const context = await browser.newContext({ viewport, locale: 'zh-CN', acceptDownloads: true })
      await context.addCookies([{ name: 'sysauth_http', value: sid, domain: targetURL.hostname, path: '/cgi-bin/luci/' }])
      const page = await context.newPage()
      const consoleErrors = []
      const requestFailures = []
      page.on('console', message => { if (message.type() === 'error') consoleErrors.push(message.text()) })
      page.on('pageerror', error => consoleErrors.push(error.message))
      page.on('requestfailed', request => requestFailures.push(`${request.method()} ${request.url()}`))
      await page.goto(target, { waitUntil: 'domcontentloaded', timeout: 20_000 })
      await page.locator('.device-center').waitFor({ timeout: 20_000 })
      const device = viewport.width <= 700
        ? page.locator('.device-card').filter({ hasText: '192.168.30.1' }).first()
        : page.locator('.device-table tbody tr').filter({ hasText: '192.168.30.1' }).first()
      await device.click()
      const advanced = page.locator('.advanced-tools')
      if (await advanced.getAttribute('open') !== null) report.failures.push(`${viewport.name}: advanced tools open by default`)
      await advanced.locator('summary').click()
      await page.locator('.advanced-body').waitFor({ timeout: 15_000 })

      await page.locator('.probe-actions button').first().click()
      await page.locator('.probe-actions + .message.success').waitFor({ timeout: 10_000 })

      const downloadPromise = page.waitForEvent('download')
      await page.locator('.bundle-actions button').click()
      const download = await downloadPromise
      const bundlePath = `/tmp/quickstart-m63-${viewport.name}.json`
      await download.saveAs(bundlePath)
      const bundle = JSON.parse(fs.readFileSync(bundlePath, 'utf8'))
      if (bundle.schemaVersion !== 1 || bundle.scope !== 'device_groups_and_quotas') report.failures.push(`${viewport.name}: exported bundle contract mismatch`)
      await page.locator('.bundle-actions input[type=file]').setInputFiles(bundlePath)
      await page.locator('.import-plan').waitFor({ timeout: 10_000 })
      if (!await page.locator('.import-plan .danger').isVisible()) report.failures.push(`${viewport.name}: dry-run has no explicit confirmation`)
      await page.screenshot({ path: `${output}/${viewport.name}-dry-run.png`, fullPage: false })
      fs.unlinkSync(bundlePath)
      if (consoleErrors.length) report.failures.push(`${viewport.name}: ${consoleErrors.length} console errors`)
      if (requestFailures.length) report.failures.push(`${viewport.name}: ${requestFailures.length} request failures`)
      report.views.push({ ...viewport, exportedGroups: bundle.groups.length, exportedDevicePolicies: Object.keys(bundle.devicePolicies).length, exportedQuotas: Object.keys(bundle.quotas).length, consoleErrors, requestFailures })
      await context.close()
    }
  } finally {
    await browser.close()
  }
  fs.writeFileSync(`${output}/report.json`, JSON.stringify(report, null, 2))
  console.log(JSON.stringify({ output, views: report.views.length, failures: report.failures }, null, 2))
  if (report.failures.length) process.exitCode = 1
})().catch(error => { console.error(error); process.exitCode = 1 })
