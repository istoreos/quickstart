import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'
import test from 'node:test'
import {homeModule} from './homeModule.mjs'

test('theme preference resolves the three supported modes', async()=>{
 const theme=await homeModule('../src/utils/theme.ts')
 assert.equal(theme.normalizeThemePreference('normal'),'system')
 assert.equal(theme.normalizeThemePreference('system'),'system')
 assert.equal(theme.normalizeThemePreference('light'),'light')
 assert.equal(theme.normalizeThemePreference('dark'),'dark')
 assert.equal(theme.normalizeThemePreference('sepia'),null)
 assert.equal(theme.resolveTheme('system',false),'light')
 assert.equal(theme.resolveTheme('system',true),'dark')
 assert.equal(theme.resolveTheme('light',true),'light')
 assert.equal(theme.resolveTheme('dark',false),'dark')
 assert.equal(theme.bodyThemeForPreference('system'),'normal')
})

test('production shell exposes an accessible persistent three-state selector',async()=>{
 const [main,selector,layout]=await Promise.all([
  readFile(new URL('../src/main.ts',import.meta.url),'utf8'),
  readFile(new URL('../src/home/ThemeSelector.vue',import.meta.url),'utf8'),
  readFile(new URL('../src/home/HomeLayout.vue',import.meta.url),'utf8'),
 ])
 assert.match(main,/initializeTheme\(\)/)
 assert.match(layout,/<ThemeSelector\/>/)
 for(const value of ['system','light','dark'])assert.match(selector,new RegExp("value:'"+value+"'"))
 assert.match(selector,/role="radiogroup"/)
 assert.match(selector,/role="radio"/)
 assert.match(selector,/setThemePreference/)
})

test('resolved theme drives semantic surfaces and every ECharts owner',async()=>{
 const [root,semantic,visual,canvas,homeFlow,deviceFlow,pie]=await Promise.all([
  readFile(new URL('../src/style/root.scss',import.meta.url),'utf8'),
  readFile(new URL('../src/home/homeSemanticColors.css',import.meta.url),'utf8'),
  readFile(new URL('../src/home/homeVisualTokens.css',import.meta.url),'utf8'),
  readFile(new URL('../src/home/canvas.css',import.meta.url),'utf8'),
  readFile(new URL('../src/pages/index/network/NetworkFlow.vue',import.meta.url),'utf8'),
  readFile(new URL('../src/pages/device/components/flow.vue',import.meta.url),'utf8'),
  readFile(new URL('../src/pages/index/components/PieChart.vue',import.meta.url),'utf8'),
 ])
 for(const source of [root,semantic,visual,canvas])assert.match(source,/data-quickstart-theme[=\"]+dark/)
 for(const source of [homeFlow,deviceFlow,pie]){
  assert.match(source,/subscribeTheme/)
  assert.match(source,/\.dispose\(\)/)
 }
})
