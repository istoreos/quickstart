const { chromium } = require('playwright')

const target = 'http://192.168.30.1/cgi-bin/luci/admin/quickstart/devicemanagement'
const api = '/cgi-bin/luci/istore/lanctrl/v2'
const cookie = process.env.M39_LUCI_COOKIE

if (!cookie) throw new Error('M39_LUCI_COOKIE is required')

;(async () => {
  const browser = await chromium.launch({ headless: true, args: ['--no-proxy-server'] })
  const context = await browser.newContext()
  await context.addCookies([{ name: 'sysauth_http', value: cookie, domain: '192.168.30.1', path: '/cgi-bin/luci/' }])
  const page = await context.newPage()
  await page.goto(target, { waitUntil: 'domcontentloaded', timeout: 15000 })
  await page.locator('.device-center').waitFor({ timeout: 15000 })

  const result = await page.evaluate(async ({ api }) => {
    const get = async path => {
      const response = await fetch(`${api}/${path}`, { cache: 'no-store' })
      const body = await response.json()
      if (!response.ok || body.error) throw new Error(body.error || `${path}: HTTP ${response.status}`)
      return body.result
    }
    const post = async (path, body) => {
      const response = await fetch(`${api}/${path}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
      const payload = await response.json()
      if (!response.ok || payload.error) throw new Error(payload.error || `${path}: HTTP ${response.status}`)
      return payload.result
    }
    const planApply = async (path, request) => {
      const plan = await post(`${path}/plan/`, request)
      if (plan.error || !plan.canApply) throw new Error(`${path} plan: ${plan.error?.code || 'not applicable'}`)
      const applied = await post(`${path}/apply/`, { ...request, expectedVersion: plan.version })
      if (applied.error || !applied.changed) throw new Error(`${path} apply: ${applied.error?.code || 'unchanged'}`)
      return applied
    }

    const inventory = await get('devices/')
    const byIPv4 = address => inventory.devices.find(device => device.addresses?.current?.some(item => item.address === address))
    const c = byIPv4('192.168.30.93')
    const d = byIPv4('192.168.30.7')
    if (!c || !d) throw new Error('C or D is missing from the inventory')

    const bundleBefore = (await get('policy-bundle/')).bundle
    let targetId = ''
    const evidence = { authenticated: true, c: c.deviceId, d: d.deviceId }
    try {
      const create = await planApply('gateway-target', { action: 'create', name: 'M39 测试路线', kind: 'bypass', gateway: '192.168.30.242', idempotencyKey: 'm39-route-create' })
      targetId = create.plan.target.id
      const update = await planApply('gateway-target', { action: 'update', targetId, name: 'M39 测试路线 2', kind: 'custom', gateway: '192.168.30.241', idempotencyKey: 'm39-route-update' })
      if (update.plan.target.id !== targetId) throw new Error('route identity changed during update')

      await planApply('gateway-assignment', { action: 'assign', deviceId: d.deviceId, targetId })
      const withoutReplacement = await post('gateway-target/plan/', { action: 'delete', targetId, idempotencyKey: 'm39-route-delete-check' })
      if (withoutReplacement.error?.code !== 'replacement_required' || withoutReplacement.canApply) throw new Error('referenced route did not require a replacement')
      const deleted = await planApply('gateway-target', { action: 'delete', targetId, replacementTargetId: 'default', idempotencyKey: 'm39-route-delete' })
      targetId = ''
      const dNetwork = await get(`device-network-policy/?deviceId=${encodeURIComponent(d.deviceId)}`)
      if (dNetwork.policy.path.targetId !== 'default') throw new Error('replacement did not restore D to the network default')
      evidence.route = { stableId: update.plan.target.id, replacementRequired: true, affectedDevices: deleted.plan.affectedDevices }

      let groups = await get('device-groups/')
      const group = {
        id: 'm39_group', name: 'M39 验收组', priority: 10, members: [c.deviceId],
        policy: { schedules: [{ id: 'rest', enabled: true, days: [0,1,2,3,4,5,6], startMinute: 1439, endMinute: 1, action: 'block' }], quota: { enabled: true, period: 'monthly', limitBytes: 1099511627776, action: 'notify' } },
      }
      groups = await post('device-groups/', { action: 'upsert_group', group, expectedVersion: groups.version })
      let effective = groups.effective.find(item => item.deviceId === c.deviceId)
      if (!effective?.sources?.includes('group:m39_group') || !effective.nextScheduleAt) throw new Error('group source or next transition is missing')

      const devicePolicy = {
        schedules: [{ id: 'device-rest', enabled: true, days: [0,1,2,3,4,5,6], startMinute: 1320, endMinute: 420, action: 'block' }],
        quota: { enabled: true, period: 'weekly', limitBytes: 1099511627776, action: 'notify' },
      }
      groups = await post('device-groups/', { action: 'set_device_policy', deviceId: c.deviceId, devicePolicy, expectedVersion: groups.version })
      effective = groups.effective.find(item => item.deviceId === c.deviceId)
      if (!effective?.sources?.includes('device') || !effective.nextScheduleAt) throw new Error('device override or next transition is missing')
      evidence.usage = { timezone: groups.timezone, sources: effective.sources, nextScheduleAt: effective.nextScheduleAt, quotaPeriod: effective.quota?.period }

      groups = await post('device-groups/', { action: 'set_device_policy', deviceId: c.deviceId, devicePolicy: null, expectedVersion: groups.version })
      groups = await post('device-groups/', { action: 'delete_group', groupId: 'm39_group', expectedVersion: groups.version })
      if (groups.groups.some(item => item.id === 'm39_group') || groups.devicePolicies[c.deviceId]) throw new Error('usage policy cleanup failed')

      const bundleAfter = (await get('policy-bundle/')).bundle
      evidence.restored = JSON.stringify({ groups: bundleAfter.groups, devicePolicies: bundleAfter.devicePolicies, quotas: bundleAfter.quotas }) === JSON.stringify({ groups: bundleBefore.groups, devicePolicies: bundleBefore.devicePolicies, quotas: bundleBefore.quotas })
      if (!evidence.restored) throw new Error('policy bundle did not return to its baseline')
      return evidence
    } finally {
      if (targetId) {
        const plan = await post('gateway-target/plan/', { action: 'delete', targetId, replacementTargetId: 'default', idempotencyKey: 'm39-route-finally' }).catch(() => null)
        if (plan?.canApply) await post('gateway-target/apply/', { action: 'delete', targetId, replacementTargetId: 'default', expectedVersion: plan.version, idempotencyKey: 'm39-route-finally' }).catch(() => null)
      }
      const current = await get('policy-bundle/').catch(() => null)
      if (current && JSON.stringify(current.bundle) !== JSON.stringify(bundleBefore)) {
        const restorePlan = await post('policy-import/plan/', { bundle: bundleBefore }).catch(() => null)
        if (restorePlan?.plan?.canApply) await post('policy-import/apply/', { bundle: bundleBefore, planChecksum: restorePlan.plan.checksum, expectedGroupVersion: restorePlan.plan.groupVersion, confirmed: true }).catch(() => null)
      }
    }
  }, { api })

  console.log(JSON.stringify(result, null, 2))
  await browser.close()
})().catch(error => { console.error(error); process.exitCode = 1 })
