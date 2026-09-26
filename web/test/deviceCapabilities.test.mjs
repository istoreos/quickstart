import assert from 'node:assert/strict'
import test from 'node:test'

import { capabilityAllowsConfiguration, resolveCapability } from '../src/pages/device/deviceCapabilities.ts'

test('resolveCapability reads the canonical capability map', () => {
    assert.deepEqual(resolveCapability({
        capabilities: {
            items: {
                device_speed_limit: { state: 'error', reason: 'status_unavailable' },
            },
            speedLimit: { state: 'available' },
        },
    }, 'speedLimit'), { state: 'error', reason: 'status_unavailable' })
})

test('resolveCapability fails closed when the capability contract is absent', () => {
    assert.deepEqual(resolveCapability({}, 'speedLimit'), { state: 'error', reason: 'capability_missing' })
})

test('resolveCapability fails closed for an unknown server state', () => {
    assert.deepEqual(resolveCapability({ capabilities: { speedLimit: { state: 'future_state' } } }, 'speedLimit'), { state: 'error', reason: 'capability_state_unknown' })
})

test('only installed capabilities allow configuration', () => {
    assert.equal(capabilityAllowsConfiguration({ state: 'available' }), true)
    assert.equal(capabilityAllowsConfiguration({ state: 'disabled' }), true)
    assert.equal(capabilityAllowsConfiguration({ state: 'not_installed' }), false)
    assert.equal(capabilityAllowsConfiguration({ state: 'unsupported' }), false)
    assert.equal(capabilityAllowsConfiguration({ state: 'error' }), false)
})
