const { chromium } = require('playwright')
const stage = process.argv[2] || 'create'
const target = 'http://192.168.9.215/cgi-bin/luci/admin/quickstart/devicemanagement'

;(async () => {
  const browser = await chromium.launch({ headless: true })
  const page = await browser.newPage()
  await page.goto(target, { waitUntil: 'domcontentloaded' })
  const username = page.locator('input[name="luci_username"]')
  if (await username.count()) { await username.fill('root'); await page.locator('input[name="luci_password"]').fill(''); await page.locator('button[type="submit"], input[type="submit"]').click() }
  await page.waitForURL(/quickstart\/devicemanagement/, { timeout: 15000 })
  const result = await page.evaluate(async stageName => {
    const inventory = (await (await fetch('/cgi-bin/luci/istore/lanctrl/v2/devices/')).json()).result
    const device = inventory.devices?.[0]
    if (!device) throw new Error('no device available')
    const path = `/cgi-bin/luci/istore/lanctrl/v2/traffic-insights/?deviceId=${encodeURIComponent(device.deviceId)}&range=month`
    const get = async () => (await (await fetch(path, { cache: 'no-store' })).json()).result
    const quota = async body => (await (await fetch('/cgi-bin/luci/istore/lanctrl/v2/traffic-quota/', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })).json()).result
    await fetch('/cgi-bin/luci/istore/lanctrl/v2/device-traffic/', { cache: 'no-store' })
    if (stageName === 'create') {
      const saved = await quota({ deviceId: device.deviceId, enabled: true, period: 'monthly', limitBytes: 1073741824, action: 'notify' })
      const insights = await get()
      return { stage: stageName, deviceId: device.deviceId, saved: saved.quota?.quota?.enabled, budget: insights.storageBudget, interval: insights.writeIntervalSeconds, local: insights.local, bandix: insights.bandix, timezone: insights.timezone, error: saved.error || insights.error }
    }
    const persisted = await get()
    const disabled = await quota({ deviceId: device.deviceId, enabled: false })
    return { stage: stageName, persisted: persisted.quota?.quota?.enabled, disabled: !disabled.quota?.quota, budget: persisted.storageBudget, error: persisted.error || disabled.error }
  }, stage)
  console.log(JSON.stringify(result, null, 2))
  if (result.error || result.budget !== 2097152 || (stage === 'create' && (!result.saved || result.interval !== 300 || result.local?.state !== 'available')) || (stage !== 'create' && (!result.persisted || !result.disabled))) throw new Error('M19 API acceptance failed')
  await browser.close()
})().catch(error => { console.error(error); process.exitCode = 1 })
