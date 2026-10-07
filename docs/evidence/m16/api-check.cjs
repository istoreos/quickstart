const { chromium } = require('playwright')
const target = 'http://192.168.9.215/cgi-bin/luci/admin/quickstart/devicemanagement'
;(async () => {
    const browser = await chromium.launch({ headless: true }); const page = await browser.newPage(); await page.goto(target, { waitUntil: 'domcontentloaded' })
    const username = page.locator('input[name="luci_username"]'); if (await username.count()) { await username.fill('root'); await page.locator('input[name="luci_password"]').fill(''); await page.locator('button[type="submit"], input[type="submit"]').click() }
    await page.waitForURL(/quickstart\/devicemanagement/, { timeout: 15000 })
    const result = await page.evaluate(async () => {
        const get = async path => (await (await fetch(path, { cache: 'no-store' })).json()).result
        const post = async (path, body) => (await (await fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })).json()).result
        const current = await get('/cgi-bin/luci/istore/lanctrl/v2/floating-gateway/')
        const plan = await post('/cgi-bin/luci/istore/lanctrl/v2/floating-gateway/plan/', { config: { enabled: true, role: 'preferred', virtualIP: '192.168.100.250', peerIPs: ['192.168.100.2'], healthTimeoutSec: 5 } })
        const drill = await get('/cgi-bin/luci/istore/lanctrl/v2/floating-gateway/drill-plan/')
        const raw = JSON.stringify({ current, plan, drill })
        return { capability: current.status.capability, state: current.status.state, planError: plan.plan?.error?.code, drillCanRun: drill.canRun, drillError: drill.error?.code, leakedInternalRole: /"role":"(?:main|fallback)"/.test(raw) }
    })
    console.log(JSON.stringify(result, null, 2))
    if (result.capability !== 'not_installed' || result.state !== 'disabled' || result.planError !== 'dependency_not_installed' || result.drillCanRun || result.leakedInternalRole) throw new Error('M16 API acceptance failed')
    await browser.close()
})().catch(error => { console.error(error); process.exitCode = 1 })
