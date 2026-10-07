const { chromium } = require('playwright')
const stage = process.argv[2] || 'create'
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
  const result = await page.evaluate(async stageName => {
    const path = '/cgi-bin/luci/istore/lanctrl/v2/device-groups/'
    const get = async () => (await (await fetch(path, { cache: 'no-store' })).json()).result
    const post = async body => (await (await fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })).json()).result
    const before = await get()
    if (stageName === 'create') {
      const created = await post({ action: 'upsert_group', expectedVersion: before.version, group: { id: 'm18_acceptance', name: 'M18 验收组', priority: 20, members: [], policy: { schedules: [{ id: 'night', enabled: true, days: [1,2,3,4,5], startMinute: 1320, endMinute: 420, action: 'block' }] } } })
      return { stage: stageName, groups: created.groups?.map(group => group.id), timezone: created.timezone, version: created.version, error: created.error }
    }
    const foundAfterRestart = before.groups?.some(group => group.id === 'm18_acceptance')
    const removed = await post({ action: 'delete_group', groupId: 'm18_acceptance', expectedVersion: before.version })
    return { stage: stageName, foundAfterRestart, remaining: removed.groups?.map(group => group.id), error: removed.error, hasVersion: Boolean(removed.version) }
  }, stage)
  console.log(JSON.stringify(result, null, 2))
  if (result.error || (stage === 'create' && (!result.groups.includes('m18_acceptance') || !result.timezone || !result.version)) || (stage !== 'create' && (!result.foundAfterRestart || result.remaining.includes('m18_acceptance') || !result.hasVersion))) throw new Error('M18 API acceptance failed')
  await browser.close()
})().catch(error => { console.error(error); process.exitCode = 1 })
