import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

test('floating gateway wizard uses product responsibilities and a three-step flow', async () => {
    const source = await readFile(new URL('../src/pages/device/components/floatingGatewayWizard.vue', import.meta.url), 'utf8')
    assert.match(source, /优先服务节点/)
    assert.match(source, /故障接管节点/)
    assert.match(source, /节点职责/)
    assert.match(source, /地址与检测/)
    assert.match(source, /预检与应用/)
    assert.doesNotMatch(source, /['"]main['"]|['"]fallback['"]/)
})

test('floating gateway drill is explicit and never starts automatically', async () => {
    const source = await readFile(new URL('../src/pages/device/components/floatingGatewayWizard.vue', import.meta.url), 'utf8')
    assert.match(source, /系统不会自动开始/)
    assert.match(source, /DRILL\(\)/)
    assert.doesNotMatch(source, /DRILL_APPLY|startDrill/)
})
