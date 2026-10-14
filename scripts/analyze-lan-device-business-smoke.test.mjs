import assert from 'node:assert/strict'
import test from 'node:test'
import { analyzeBusinessSnapshot } from './analyze-lan-device-business-smoke.mjs'

const device = (deviceId, address) => ({
  deviceId,
  online: true,
  addresses: { current: [{ family: 4, address, primary: true }], historical: [] },
})
const target = (id, kind, gateway) => ({ id, kind, gateway, supported: true })

const healthySnapshot = () => ({
  a: {
    routerContext: { topologyPosition: 'lan_gateway_candidate', dhcpAuthority: 'local', routeEditability: { editable: true } },
    devices: [device('c', '192.168.30.93'), device('d', '192.168.30.7')],
    targets: [target('self', 'self', '192.168.30.1'), target('bypass', 'bypass', '192.168.30.244'), target('floating', 'floating', '192.168.30.3')],
    floating: { config: { enabled: true }, status: { capability: 'available', serviceRunning: true, state: 'healthy', holder: 'peer' } },
    dhcp: { settings: { enabled: true }, editable: true },
    rules: { rules: [{ id: 'one', status: 'active', orphaned: false }] },
    groups: { groups: [], devicePolicies: {}, effective: [], timezone: 'UTC', version: 'v1' },
    traffic: { deviceId: 'c', range: 'today', buckets: [], uploadBytes: 1, downloadBytes: 2, local: { state: 'available' }, storageBytes: 10, storageBudget: 100 },
    cPolicy: { policy: { path: { effect: { desired: { targetId: 'floating' }, applied: { state: 'server_applied' }, observed: { state: 'lease_observed_unverifiable' }, needsAttention: false } } } },
    dPolicy: { policy: { path: { effect: { desired: { targetId: 'default' }, applied: { state: 'server_configuration_observed' }, observed: { state: 'unverifiable' }, needsAttention: false } } } },
  },
  b: {
    routerContext: { topologyPosition: 'downstream_router', dhcpAuthority: 'none_detected', routeEditability: { editable: false } },
    devices: [device('c', '192.168.30.93'), device('d', '192.168.30.7')],
    targets: [target('self', 'self', '192.168.30.244'), target('upstream', 'upstream', '192.168.30.1'), target('floating', 'floating', '192.168.30.3')],
    floating: { config: { enabled: true }, status: { capability: 'available', serviceRunning: true, state: 'healthy', holder: 'local' } },
    dhcp: { settings: { enabled: false }, editable: false },
    rules: { rules: [] },
    groups: { groups: [], devicePolicies: {}, effective: [], timezone: 'UTC', version: 'v1' },
    traffic: { deviceId: 'c', range: 'today', buckets: [], uploadBytes: 0, downloadBytes: 0, local: { state: 'available' }, storageBytes: 0, storageBudget: 100 },
    cPolicy: { policy: { path: { effect: { desired: { targetId: 'default' }, applied: { state: 'server_configuration_observed' }, observed: { state: 'failed' }, needsAttention: true, attentionReason: 'dhcp_authority_unavailable' } } } },
  },
  network: { aVip: 'absent', bVip: 'present', cGateway: '192.168.30.244', dGateway: '192.168.30.1', cPing: 'ok', dPing: 'ok' },
})

test('healthy four-host business snapshot passes', () => {
  const result = analyzeBusinessSnapshot(healthySnapshot())
  assert.equal(result.passed, true)
  assert.deepEqual(result.failures, [])
})

test('unsafe authority, false holder, duplicate identity and traffic error fail closed', () => {
  const snapshot = healthySnapshot()
  snapshot.b.routerContext.routeEditability.editable = true
  snapshot.a.floating.status.holder = 'unknown'
  snapshot.a.devices.push(device('c', '192.168.30.99'))
  snapshot.a.traffic.error = { code: 'validation_failed' }
  snapshot.network.aVip = 'present'
  const result = analyzeBusinessSnapshot(snapshot)
  assert.equal(result.passed, false)
  assert.ok(result.failures.some(value => value.includes('B route editing')))
  assert.ok(result.failures.some(value => value.includes('A floating holder')))
  assert.ok(result.failures.some(value => value.includes('duplicate device IDs')))
  assert.ok(result.failures.some(value => value.includes('traffic error')))
  assert.ok(result.failures.some(value => value.includes('exactly one VIP owner')))
})
