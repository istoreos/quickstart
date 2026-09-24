import assert from 'node:assert/strict'
import test from 'node:test'

import { capabilityAllowsConfiguration, resolveCapability } from '../src/pages/device/deviceCapabilities.ts'

test('resolveCapability prefers the explicit backend contract', () => {
    assert.deepEqual(resolveCapability({
        capabilities: { speedLimit: { state: 'error', reason: 'status_unavailable' } },
        speedLimit: { installed: true, enabled: true },
    }, 'speedLimit'), { state: 'error', reason: 'status_unavailable' })
})

test('resolveCapability remains compatible with the legacy installed flags', () => {
    assert.deepEqual(resolveCapability({ speedLimit: { installed: false } }, 'speedLimit'), {
        state: 'not_installed',
        reason: 'dependency_not_installed',
    })
    assert.deepEqual(resolveCapability({ floatGateway: { installed: true, enabled: false } }, 'floatGateway'), {
        state: 'disabled',
    })
})

test('only installed capabilities allow configuration', () => {
    assert.equal(capabilityAllowsConfiguration({ state: 'available' }), true)
    assert.equal(capabilityAllowsConfiguration({ state: 'disabled' }), true)
    assert.equal(capabilityAllowsConfiguration({ state: 'not_installed' }), false)
    assert.equal(capabilityAllowsConfiguration({ state: 'error' }), false)
})
