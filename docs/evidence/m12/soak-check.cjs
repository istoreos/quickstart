const { execFileSync } = require('node:child_process')
const fs = require('node:fs')
const { chromium } = require('playwright')

const target = 'http://192.168.9.215/cgi-bin/luci/admin/quickstart/devicemanagement'
const samples = Number(process.env.M12_SOAK_SAMPLES || 120)
const intervalMs = Number(process.env.M12_SOAK_INTERVAL_MS || 60_000)
const output = process.env.M12_SOAK_OUTPUT || 'docs/evidence/m12/soak-result.json'
const sleep = milliseconds => new Promise(resolve => setTimeout(resolve, milliseconds))

const routerSample = () => {
  const command = [
    'pid=$(pidof quickstart | awk \'{print $1}\')',
    'test -n "$pid"',
    'rss=$(awk \'/VmRSS:/ {print $2}\' "/proc/$pid/status")',
    'printf \'%s %s %s\\n\' "$(date +%s)" "$pid" "$rss"',
  ].join('; ')
  const line = execFileSync('ssh', ['-o', 'BatchMode=yes', '-o', 'StrictHostKeyChecking=no', 'root@192.168.9.215', command], { encoding: 'utf8' }).trim()
  const [at, pid, rssKiB] = line.split(/\s+/).map(Number)
  if (![at, pid, rssKiB].every(Number.isFinite)) throw new Error(`invalid router sample: ${line}`)
  return { at, pid, rssKiB }
}

;(async () => {
  if (!Number.isInteger(samples) || samples < 1 || !Number.isFinite(intervalMs) || intervalMs < 1000) throw new Error('invalid soak settings')
  const browser = await chromium.launch({ headless: true })
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
  await page.goto(target, { waitUntil: 'domcontentloaded' })
  const username = page.locator('input[name="luci_username"]')
  if (await username.count()) {
    await username.fill('root')
    await page.locator('input[name="luci_password"]').fill('')
    await page.locator('button[type="submit"], input[type="submit"]').click()
  }
  await page.waitForURL(/quickstart\/devicemanagement/, { timeout: 15_000 })
  await page.locator('#device-center-title').waitFor({ timeout: 15_000 })
  await page.waitForTimeout(10_000)

  const readings = [routerSample()]
  for (let index = 1; index <= samples; index += 1) {
    await sleep(intervalMs)
    readings.push(routerSample())
    if (index % 10 === 0) console.log(`sample ${index}/${samples}: ${readings.at(-1).rssKiB} KiB`)
  }

  const baselineIndex = Math.min(5, readings.length - 1)
  const baselineRssKiB = readings[baselineIndex].rssKiB
  const maxRssKiB = Math.max(...readings.slice(baselineIndex).map(item => item.rssKiB))
  const pidStable = readings.every(item => item.pid === readings[0].pid)
  const result = {
    startedAt: new Date(readings[0].at * 1000).toISOString(),
    finishedAt: new Date(readings.at(-1).at * 1000).toISOString(),
    durationSeconds: readings.at(-1).at - readings[0].at,
    sampleCount: readings.length,
    intervalMs,
    pidStable,
    baselineRssKiB,
    maxRssKiB,
    growthKiB: maxRssKiB - baselineRssKiB,
    readings,
  }
  fs.writeFileSync(output, `${JSON.stringify(result, null, 2)}\n`)
  console.log(JSON.stringify({ ...result, readings: undefined }, null, 2))
  await browser.close()
  if (!pidStable) throw new Error('quickstart restarted during soak')
  if (result.durationSeconds < samples * intervalMs / 1000) throw new Error('soak duration was shorter than requested')
  if (result.growthKiB > 10 * 1024) throw new Error(`RSS grew by ${result.growthKiB} KiB`)
})().catch(error => {
  console.error(error)
  process.exitCode = 1
})
