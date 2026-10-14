const { chromium } = require('playwright')
const { execFileSync } = require('node:child_process')

const origin = 'http://192.168.9.215'
const pageURL = `${origin}/cgi-bin/luci/admin/quickstart/devicemanagement`

const percentile95 = values => {
    const sorted = [...values].sort((left, right) => left - right)
    return sorted[Math.max(0, Math.ceil(sorted.length * 0.95) - 1)]
}

const routerCPU = () => {
    const command = `pid=$(pidof quickstart | awk '{print $1}'); ticks=$(awk '{print $14+$15}' /proc/$pid/stat); rss=$(awk '/VmRSS:/ {print $2}' /proc/$pid/status); printf '%s %s %s %s\n' "$pid" "$ticks" "100" "$rss"`
    const [pid, ticks, hertz, rssKiB] = execFileSync('ssh', ['-o', 'BatchMode=yes', '-o', 'StrictHostKeyChecking=no', 'root@192.168.9.215', command], { encoding: 'utf8' }).trim().split(/\s+/).map(Number)
    return { pid, ticks, hertz, rssKiB }
}

;(async () => {
    const browser = await chromium.launch({ headless: true })
    const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
    const page = await context.newPage()
    const requests = []
    page.on('request', request => requests.push(new URL(request.url()).pathname))
    await page.goto(pageURL, { waitUntil: 'domcontentloaded' })
    const username = page.locator('input[name="luci_username"]')
    if (await username.count()) {
        await username.fill('root')
        await page.locator('input[name="luci_password"]').fill('')
        await page.locator('button[type="submit"], input[type="submit"]').click()
    }
    await page.waitForURL(/quickstart\/devicemanagement/, { timeout: 15000 })
    await page.locator('#device-center-title').waitFor({ timeout: 15000 })
    await page.waitForTimeout(7000)

    const cpuBefore = routerCPU()
    const timingStarted = Date.now()
    const timing = await page.evaluate(async () => {
        const run = async path => {
            const started = performance.now()
            const response = await fetch(path)
            const body = await response.json()
            if (!response.ok || body.error) throw new Error(`${path}: ${response.status} ${body.error || 'request failed'}`)
            return performance.now() - started
        }
        const inventory = []
        const traffic = []
        const inventorySampleResponse = await fetch('/cgi-bin/luci/istore/lanctrl/v2/devices/', { cache: 'no-store' })
        const inventorySampleText = await inventorySampleResponse.text()
        const inventorySample = JSON.parse(inventorySampleText)
        const trafficSampleResponse = await fetch('/cgi-bin/luci/istore/lanctrl/v2/device-traffic/', { cache: 'no-store' })
        const trafficSampleText = await trafficSampleResponse.text()
        for (let index = 0; index < 40; index += 1) {
            inventory.push(await run('/cgi-bin/luci/istore/lanctrl/v2/devices/'))
            traffic.push(await run('/cgi-bin/luci/istore/lanctrl/v2/device-traffic/'))
        }
        return {
            inventory,
            traffic,
            deviceCount: inventorySample.result?.devices?.length || 0,
            inventoryBytes: new TextEncoder().encode(inventorySampleText).length,
            trafficBytes: new TextEncoder().encode(trafficSampleText).length,
        }
    })
    const timingSeconds = Math.max((Date.now() - timingStarted) / 1000, 0.001)
    const cpuAfter = routerCPU()

    const diagnostics = await page.evaluate(async () => {
        const response = await fetch(`/cgi-bin/luci/istore/lanctrl/v2/device-diagnostics/?t=${Date.now()}`, { cache: 'no-store' })
        return response.json()
    })
    const legacyListRequests = requests.filter(path => path.endsWith('/lanctrl/listDevices/')).length

    const observer = await context.newPage()
    await observer.goto(`${origin}/cgi-bin/luci/admin/status/overview`, { waitUntil: 'domcontentloaded' })
    await page.close()
    await observer.waitForTimeout(35000)
    const stopped = await observer.evaluate(async () => {
        const response = await fetch(`/cgi-bin/luci/istore/lanctrl/v2/device-diagnostics/?t=${Date.now()}`, { cache: 'no-store' })
        return response.json()
    })

    const result = {
        inventoryP95Ms: Number(percentile95(timing.inventory).toFixed(2)),
        trafficP95Ms: Number(percentile95(timing.traffic).toFixed(2)),
        deviceCount: timing.deviceCount,
        inventoryResponseBytes: timing.inventoryBytes,
        trafficResponseBytes: timing.trafficBytes,
        quickstartPIDStable: cpuBefore.pid === cpuAfter.pid,
        quickstartCPUPercent: Number(((cpuAfter.ticks - cpuBefore.ticks) / cpuBefore.hertz / timingSeconds * 100).toFixed(2)),
        quickstartRSSBeforeKiB: cpuBefore.rssKiB,
        quickstartRSSAfterKiB: cpuAfter.rssKiB,
        legacyListRequests,
        diagnostics: diagnostics.result,
        samplingStopped: stopped.result?.sampler?.sampling === false,
    }
    console.log(JSON.stringify(result, null, 2))
    if (result.inventoryP95Ms > 500 || result.trafficP95Ms > 500) throw new Error('p95 latency exceeded 500ms')
    if (legacyListRequests !== 0) throw new Error('new device center still requested legacy listDevices')
    if (!result.quickstartPIDStable) throw new Error('quickstart restarted during runtime benchmark')
    if (!result.samplingStopped) console.warn('sampler still has another consumer; isolated auto-stop is covered by the Go test')
    if ((result.diagnostics?.oui?.heapAllocBytes || 0) > 8 * 1024 * 1024) throw new Error('OUI heap delta exceeded 8 MiB')
    await browser.close()
})().catch(error => {
    console.error(error)
    process.exitCode = 1
})
