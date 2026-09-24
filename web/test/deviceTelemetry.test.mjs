import assert from 'node:assert/strict'
import test from 'node:test'

import { formatTrafficBytes, telemetryBackoff, telemetryByDevice, telemetrySpeedLabel } from '../src/pages/device/deviceTelemetry.ts'

test('traffic values distinguish warming up from a real zero sample', () => {
    assert.equal(telemetrySpeedLabel(undefined, 'up'), '采集中')
    assert.equal(telemetrySpeedLabel({ deviceId: 'one', state: 'warming_up', uploadSpeed: 0, downloadSpeed: 0, uploadBytes: 0, downloadBytes: 0, connectionCount: 0 }, 'up'), '采集中')
    assert.equal(telemetrySpeedLabel({ deviceId: 'one', state: 'ready', uploadSpeed: 0, downloadSpeed: 0, uploadBytes: 0, downloadBytes: 0, connectionCount: 0 }, 'up'), '0 B/s')
})

test('traffic formatting and failure backoff remain bounded', () => {
    assert.equal(formatTrafficBytes(1530), '1.53 KB')
    assert.equal(formatTrafficBytes(1250000, '/s'), '1.25 MB/s')
    assert.deepEqual([1, 2, 3, 8].map(telemetryBackoff), [3000, 6000, 12000, 30000])
})

test('telemetry is indexed only by stable device identity', () => {
    const item = { deviceId: 'mac:aa', state: 'ready', uploadSpeed: 1, downloadSpeed: 2, uploadBytes: 3, downloadBytes: 4, connectionCount: 5 }
    const indexed = telemetryByDevice([item])
    assert.equal(indexed.get('mac:aa'), item)
    assert.equal(indexed.has('192.168.100.20'), false)
})
