const { chromium } = require('playwright')
const target = 'http://192.168.9.215/cgi-bin/luci/admin/quickstart/devicemanagement'
const viewports = [['desktop-1440x900',1440,900],['compact-1024x768',1024,768],['mobile-390x844',390,844]]

;(async () => {
  const browser = await chromium.launch({ headless: true })
  for (const [name,width,height] of viewports) {
    const page = await browser.newPage({ viewport: { width, height } })
    await page.goto(target, { waitUntil: 'domcontentloaded' })
    const username = page.locator('input[name="luci_username"]')
    if (await username.count()) { await username.fill('root'); await page.locator('input[name="luci_password"]').fill(''); await page.locator('button[type="submit"], input[type="submit"]').click() }
    await page.waitForURL(/quickstart\/devicemanagement/, { timeout: 15000 })
    await page.getByRole('button', { name: '网络规则', exact: true }).click()
    await page.getByText('家庭分组与上网时段', { exact: true }).waitFor()
    const create = page.getByRole('button', { name: '新建分组', exact: true })
    await create.click()
    await page.screenshot({ path: `docs/evidence/m18/${name}.png`, fullPage: true })
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1)
    if (overflow) throw new Error(`${name} has horizontal overflow`)
    await page.close()
  }
  await browser.close()
})().catch(error => { console.error(error); process.exitCode = 1 })
