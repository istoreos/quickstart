import assert from 'node:assert/strict'
import test from 'node:test'

import { normalizeDeviceAlias, validDeviceAlias } from '../src/pages/device/deviceProfile.ts'

test('device alias preserves safe Unicode independently of DHCP hostname', () => {
    assert.equal(normalizeDeviceAlias('  客厅电视  '), '客厅电视')
    assert.equal(validDeviceAlias('小明的 iPad'), true)
    assert.equal(validDeviceAlias('ASUS 路由器'), true)
    assert.equal(validDeviceAlias(''), true)
})

test('device alias rejects controls and bounded Unicode overflow', () => {
    assert.equal(validDeviceAlias('bad\nname'), false)
    assert.equal(validDeviceAlias('设'.repeat(65)), false)
    assert.equal(validDeviceAlias('😀'.repeat(64)), true)
    assert.equal(validDeviceAlias('😀'.repeat(65)), false)
})
