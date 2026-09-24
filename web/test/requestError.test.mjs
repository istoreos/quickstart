import assert from 'node:assert/strict'
import test from 'node:test'

import { requestErrorMessage } from '../src/pages/device/requestError.ts'

test('requestErrorMessage prefers an API response error', () => {
    const error = {
        message: 'generic transport error',
        response: { data: { error: '设备服务暂时不可用' } },
    }

    assert.equal(requestErrorMessage(error, 'fallback'), '设备服务暂时不可用')
})

test('requestErrorMessage falls back for opaque errors', () => {
    assert.equal(requestErrorMessage({ code: 'UNKNOWN' }, '加载失败，请重试'), '加载失败，请重试')
    assert.equal(requestErrorMessage(null, '加载失败，请重试'), '加载失败，请重试')
})
