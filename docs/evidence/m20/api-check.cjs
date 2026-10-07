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
    const get = async path => (await (await fetch(path, { cache: 'no-store' })).json()).result
    const post = async (path, body) => (await (await fetch(path, { method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body) })).json()).result
    const inventory = await get('/cgi-bin/luci/istore/lanctrl/v2/devices/')
    const device = inventory.devices?.find(item => item.addresses?.current?.length)
    if (!device) throw new Error('no addressable device')
    const address = device.addresses.current[0].address
    const advanced = await get(`/cgi-bin/luci/istore/lanctrl/v2/advanced-network/?deviceId=${encodeURIComponent(device.deviceId)}`)
    const unsafe = await post('/cgi-bin/luci/istore/lanctrl/v2/management-probe/', {deviceId:device.deviceId,address:'8.8.8.8',scheme:'http',port:80})
    const badPort = await post('/cgi-bin/luci/istore/lanctrl/v2/management-probe/', {deviceId:device.deviceId,address,scheme:'http',port:22})
    const probe = await post('/cgi-bin/luci/istore/lanctrl/v2/management-probe/', {deviceId:device.deviceId,address,scheme:'http',port:80})
    const exported = await get('/cgi-bin/luci/istore/lanctrl/v2/policy-bundle/')
    const testGroup = {id:'m20_acceptance',name:'M20 acceptance',priority:1,members:[],policy:{schedules:[]}}
    const testQuota = {deviceId:device.deviceId,enabled:true,period:'monthly',limitBytes:1099511627776,action:'notify'}
    const candidate = {...exported.bundle,groups:[...(exported.bundle.groups||[]).filter(item=>item.id!=='m20_acceptance'),testGroup],quotas:{...(exported.bundle.quotas||{}),[device.deviceId]:testQuota}}
    const plan = await post('/cgi-bin/luci/istore/lanctrl/v2/policy-import/plan/', {bundle:candidate})
    const incompatible = await post('/cgi-bin/luci/istore/lanctrl/v2/policy-import/plan/', {bundle:{...exported.bundle,schemaVersion:99}})
    const noConfirm = await post('/cgi-bin/luci/istore/lanctrl/v2/policy-import/apply/', {bundle:candidate})
    if (stageName === 'create') {
      const stalePlan = await post('/cgi-bin/luci/istore/lanctrl/v2/policy-import/apply/', {bundle:candidate,planChecksum:'stale',confirmed:true})
      const applied = await post('/cgi-bin/luci/istore/lanctrl/v2/policy-import/apply/', {bundle:candidate,planChecksum:plan.plan.checksum,expectedGroupVersion:plan.plan.groupVersion,confirmed:true})
      const webhook = await post('/cgi-bin/luci/istore/lanctrl/v2/network-webhook/', {enabled:false})
      return {stage:stageName,deviceId:device.deviceId,capabilities:advanced.capabilities,eventCount:advanced.events?.length,unsafe:unsafe.error?.code,badPort:badPort.error?.code,probe:probe.error?.code||probe.statusCode,plan:plan.plan,incompatible:incompatible.error?.code,noConfirm:noConfirm.error?.code,stalePlan:stalePlan.error?.code,applied:applied.changed,webhookEnabled:webhook.webhook?.enabled}
    }
    return {stage:stageName,scope:exported.bundle?.scope,testGroup:exported.bundle?.groups?.some(item=>item.id==='m20_acceptance'),testQuota:Boolean(exported.bundle?.quotas?.[device.deviceId]),eventCount:advanced.events?.length,webhookEnabled:advanced.webhook?.enabled}
  }, stage)
  console.log(JSON.stringify(result,null,2))
  if (stage === 'create') {
    if (result.unsafe !== 'validation_failed' || result.badPort !== 'validation_failed' || !['unreachable',200,204,301,302,401,403,404].includes(result.probe) || !result.plan?.canApply || result.incompatible !== 'incompatible_version' || result.noConfirm !== 'confirmation_required' || result.stalePlan !== 'conflict' || !result.applied || result.webhookEnabled) throw new Error('M20 API acceptance failed')
    for (const key of ['ipv6_reservation','connection_insights','dns_insights','management_probe','segmentation','multi_wan','identification_updates']) if (!result.capabilities?.[key]?.state) throw new Error(`missing capability ${key}`)
  } else if (result.scope !== 'device_groups_and_quotas' || !result.testGroup || !result.testQuota || result.webhookEnabled || result.eventCount < 1) throw new Error('M20 restart acceptance failed')
  await browser.close()
})().catch(error => { console.error(error); process.exitCode = 1 })
