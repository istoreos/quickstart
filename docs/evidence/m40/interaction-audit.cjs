const { chromium } = require('playwright')
const fs = require('node:fs')

const target = process.env.M40_UI_URL || 'http://192.168.30.1/cgi-bin/luci/admin/quickstart/devicemanagement'
const targetURL = new URL(target)
const cookie = process.env.M40_LUCI_COOKIE
const output = process.env.M40_UI_OUTPUT || '/tmp/quickstart-ui-audit'
if (!cookie) throw new Error('M40_LUCI_COOKIE is required')
fs.mkdirSync(output, { recursive: true })
let browser

const viewports = [
  { name: 'desktop', width: 1440, height: 900 },
  { name: 'tablet', width: 1024, height: 768 },
  { name: 'mobile', width: 390, height: 844 },
]

const inspectPage = page => page.evaluate(() => {
  const visible = element => {
    const style = getComputedStyle(element)
    const rect = element.getBoundingClientRect()
    return style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0
  }
  const details = element => {
    const style = getComputedStyle(element)
    const pseudo = name => {
      const value = getComputedStyle(element, name)
      return { content: value.content, position: value.position, inset: `${value.top} ${value.right} ${value.bottom} ${value.left}`, width: value.width, height: value.height, background: value.backgroundColor, pointerEvents: value.pointerEvents, zIndex: value.zIndex }
    }
    const rect = element.getBoundingClientRect()
    return {
      tag: element.tagName.toLowerCase(),
      className: element.className,
      text: element.innerText.trim().replace(/\s+/g, ' ').slice(0, 180),
      rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height },
      color: style.color,
      background: style.backgroundColor,
      padding: style.padding,
      overflow: style.overflow,
      before: pseudo('::before'),
      after: pseudo('::after'),
    }
  }
  const controls = [...document.querySelectorAll('.device-management button,.device-management input,.device-management select,.device-management summary')].filter(visible)
  const unnamed = controls.filter(element => !(element.innerText || element.getAttribute('aria-label') || element.getAttribute('title') || element.getAttribute('placeholder') || element.getAttribute('name')))
  const small = controls.map(details).filter(item => item.rect.width < 32 || item.rect.height < 32)
  const builtInChinese = [
    '跟随网络默认', '设置速度上限', '设置流量额度', '自定义网关', '本机路由', '自动推荐', '自行选择',
    '当前无法确认地址分配设备', '本机 DHCP 服务异常', '检测到本机和外部 DHCP 证据', '请在主路由的 DHCP 设置中调整路线',
    '检测到多个 DHCP 服务证据', '暂时无法判断 DHCP 分配权',
  ]
    .filter(label => document.querySelector('.device-management')?.innerText.includes(label))
  return {
    url: location.href,
    title: document.title,
    documentOverflowX: document.documentElement.scrollWidth > document.documentElement.clientWidth + 1,
    scrollWidth: document.documentElement.scrollWidth,
    clientWidth: document.documentElement.clientWidth,
    root: details(document.querySelector('.device-management')),
    headers: [...document.querySelectorAll('.device-management header')].filter(visible).map(details),
    headings: [...document.querySelectorAll('.device-management h1,.device-management h2,.device-management h3')].filter(visible).map(details),
    unnamedControlCount: unnamed.length,
    unnamedControls: unnamed.map(details),
    smallControlCount: small.length,
    smallControls: small.slice(0, 20),
    primaryTabs: [...document.querySelectorAll('.primary-tabs button')].map(details),
    profileEditorVisible: [...document.querySelectorAll('.profile-editor')].some(visible),
    builtInChinese,
  }
})

;(async () => {
  const report = { target, generatedAt: new Date().toISOString(), views: [], consoleErrors: [], requestFailures: [], interactionFailures: [], blockers: [] }
  browser = await chromium.launch({ headless: true, args: ['--no-proxy-server'] })
  for (const viewport of viewports) {
    const context = await browser.newContext({ viewport: { width: viewport.width, height: viewport.height } })
    await context.addCookies([{ name: 'sysauth_http', value: cookie, domain: targetURL.hostname, path: '/cgi-bin/luci/' }])
    const page = await context.newPage()
    page.on('console', message => { if (message.type() === 'error') report.consoleErrors.push(`${viewport.name}: ${message.text()}`) })
    page.on('pageerror', error => report.consoleErrors.push(`${viewport.name}: ${error.message}`))
    page.on('requestfailed', request => report.requestFailures.push(`${viewport.name}: ${request.method()} ${request.url()} ${request.failure()?.errorText || ''}`))
    const response = await page.goto(target, { waitUntil: 'domcontentloaded', timeout: 20_000 })
    await page.locator('.device-center').waitFor({ timeout: 20_000 })
    if (await page.locator('input[name="luci_username"]').count()) throw new Error(`${viewport.name} authentication failed`)

    const capture = async name => {
      await page.waitForTimeout(250)
      await page.screenshot({ path: `${output}/${viewport.name}-${name}.png`, fullPage: true })
      report.views.push({ viewport, name, ...(await inspectPage(page)) })
    }
    const clickOrRecord = async (locator, label) => {
      try {
        await locator.click({ timeout: 2_000 })
      } catch (error) {
        report.interactionFailures.push(`${viewport.name}: ${label}: ${error.message.split('\n')[0]}`)
        await locator.evaluate(element => element.click())
      }
    }

    await capture('devices')
    const device = viewport.width <= 700
      ? page.locator('.device-card').filter({ hasText: '192.168.30.7' }).first()
      : page.locator('.device-table tbody tr').filter({ hasText: '192.168.30.7' }).first()
    await device.click()
    await page.locator('.device-drawer').waitFor()
    for (const [index, name] of ['overview', 'profile', 'network', 'usage'].entries()) {
      await page.locator('.drawer-tabs button').nth(index).click()
      await capture(`device-${name}`)
    }
    await page.locator('.drawer-close').click()

    await page.locator('.primary-tabs button').nth(1).click()
    await page.locator('.group-panel .evaluation').waitFor({ timeout: 15_000 })
    await capture('groups-empty')
    await clickOrRecord(page.locator('.group-header .header-actions .primary'), 'create group button blocked')
    await page.locator('.group-panel .editor').waitFor()
    await capture('groups-editor')
    await page.locator('.group-panel .editor .close').click()

    await page.locator('.primary-tabs button').nth(2).click()
    await page.locator('.lan-settings .context-card').waitFor({ timeout: 15_000 })
    await capture('settings-services')
    await page.locator('.settings-nav button').nth(1).click()
    await capture('settings-routes')
    const routeButton = page.locator('.settings-stack .setting-card').first().locator('.card-heading button')
    if (await routeButton.isEnabled()) {
      await clickOrRecord(routeButton, 'add route button blocked')
      await page.locator('.route-form').waitFor()
      await capture('settings-route-editor')
      await clickOrRecord(routeButton, 'cancel route editor button blocked')
    } else {
      await capture('settings-route-readonly')
    }
    await page.locator('.settings-nav button').nth(2).click()
    await page.locator('.rules-hub').waitFor()
    await capture('settings-rules')

    report.views.at(-1).httpStatus = response?.status()
    await context.close()
  }
  await browser.close()
  const overlayViews = report.views.filter(view => view.headers.some(header =>
    !['none', 'normal'].includes(header.after.content) &&
    header.after.pointerEvents !== 'none' &&
    header.after.height !== '0px'
  ))
  if (overlayViews.length) {
    report.blockers.push(`LuCI header pseudo-element overlays content in ${overlayViews.length}/${report.views.length} captured states`)
  }
  const blankProfiles = report.views.filter(view => view.name === 'device-profile' && !view.profileEditorVisible)
  if (blankProfiles.length) {
    report.blockers.push(`device profile editor is blank in ${blankProfiles.length}/${viewports.length} viewports`)
  }
  const untranslated = [...new Set(report.views.flatMap(view => view.builtInChinese))]
  if (untranslated.length) {
    report.blockers.push(`built-in labels do not follow the active locale: ${untranslated.join(', ')}`)
  }
  if (report.consoleErrors.length) report.blockers.push(`${report.consoleErrors.length} browser console error(s)`)
  if (report.requestFailures.length) report.blockers.push(`${report.requestFailures.length} failed browser request(s)`)
  if (report.interactionFailures.length) report.blockers.push(`${report.interactionFailures.length} blocked primary interaction(s)`)
  fs.writeFileSync(`${output}/report.json`, JSON.stringify(report, null, 2))
  console.log(JSON.stringify({ output, views: report.views.length, blockers: report.blockers, consoleErrors: report.consoleErrors, requestFailures: report.requestFailures, interactionFailures: report.interactionFailures }, null, 2))
  if (report.blockers.length) process.exitCode = 1
})().catch(async error => { if (browser) await browser.close().catch(() => {}); console.error(error); process.exitCode = 1 })
