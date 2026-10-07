import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'

import { classificationErrorLabel, editableDeviceCategories } from '../src/pages/device/deviceClassification.ts'

test('M9 offers each supported visual category exactly once', () => {
    assert.equal(editableDeviceCategories.length, 11)
    assert.equal(new Set(editableDeviceCategories).size, 11)
    assert.ok(editableDeviceCategories.includes('computer'))
    assert.ok(editableDeviceCategories.includes('network'))
    assert.ok(!editableDeviceCategories.includes('unknown'))
})

test('M9 translates stable backend errors and preserves an unknown message', () => {
    assert.match(classificationErrorLabel('not_found'), /身份发生变化/)
    assert.match(classificationErrorLabel('validation_failed'), /有效的设备类型/)
    assert.match(classificationErrorLabel('write_failed'), /保存失败/)
    assert.equal(classificationErrorLabel('future_error', '后端详情'), '后端详情')
})

test('M10 ships English labels for categories, evidence and manual correction', () => {
    const messages = JSON.parse(fs.readFileSync(new URL('../public/luci-static/quickstart/i18n/en.json', import.meta.url))).en
    const expected = {
        '电脑': 'Computer',
        '手机': 'Phone',
        '电视与影音': 'TV & media',
        '网络设备': 'Network device',
        '识别依据': 'Detection source',
        '识别可信度': 'Detection confidence',
        '修改设备类型': 'Change device type',
        '恢复自动判断': 'Restore automatic detection',
        '已手动设置': 'Set manually',
    }
    for (const [source, translation] of Object.entries(expected)) assert.equal(messages[source], translation)
})
