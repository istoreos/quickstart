import assert from 'node:assert/strict'
import test from 'node:test'

import { buildDeviceListItems, detectDeviceScene, deviceCounts, deviceIdentityPresentation, normalizeDeviceBrand, resolveDeviceClassification, selectDeviceListItems, splitHighlight } from '../src/pages/device/deviceInventory.ts'
import { deviceIconKeys, deviceIconLabels, deviceIconPath, deviceSceneIconPath } from '../src/pages/device/deviceScene.ts'

const inventory = [
    {
        deviceId: 'mac:aa:bb:cc:dd:ee:01', displayName: 'Living TV', online: true,
        lastSeenAt: '2026-09-24T01:00:00Z', mac: 'AA:BB:CC:DD:EE:01', vendor: 'Example',
        addresses: { current: [{ family: 4, address: '192.168.100.20', primary: true }], historical: [] },
        connection: { kind: 'wifi' },
    },
    {
        deviceId: 'mac:aa:bb:cc:dd:ee:02', displayName: 'Printer', online: false,
        lastSeenAt: '2026-09-23T01:00:00Z', mac: 'AA:BB:CC:DD:EE:02',
        addresses: { current: [], historical: [{ family: 4, address: '192.168.100.30' }] },
        connection: { kind: 'unknown' },
    },
]

test('buildDeviceListItems joins policy summaries by stable MAC identity', () => {
    const result = buildDeviceListItems(inventory, [{
        mac: 'aa:bb:cc:dd:ee:01', speedLimit: { enabled: true, networkAccess: false },
    }])
    assert.equal(result[0].primaryAddress, '192.168.100.20')
    assert.deepEqual(result[0].policyLabels, ['blocked'])
    assert.equal(result[0].controlled, true)
})

test('v2 policy rules replace the legacy device list dependency', () => {
    const result = buildDeviceListItems(inventory, [], {
        static: [{ assignedMac: 'AA:BB:CC:DD:EE:01' }],
        speed: [{ mac: 'AA:BB:CC:DD:EE:01', enabled: true, networkAccess: false }],
    })
    assert.deepEqual(result[0].policyLabels, ['static', 'blocked'])
    assert.equal(result[0].controlled, true)
    assert.equal(result[0].legacy, undefined)
})

test('selectDeviceListItems filters, searches all identity fields and sorts without mutating', () => {
    const items = buildDeviceListItems(inventory, [])
    assert.deepEqual(selectDeviceListItems(items, 'online', '', 'recent').map(item => item.deviceId), [inventory[0].deviceId])
    assert.deepEqual(selectDeviceListItems(items, 'all', '192.168.100.30', 'recent').map(item => item.deviceId), [inventory[1].deviceId])
    assert.deepEqual(selectDeviceListItems(items, 'all', '', 'name').map(item => item.displayName), ['Living TV', 'Printer'])
    assert.equal(items.length, 2)
})

test('deviceCounts and safe highlight segments support the list controls', () => {
    const items = buildDeviceListItems(inventory, [{ mac: 'AA:BB:CC:DD:EE:01', staticAssigned: { bindIP: true } }])
    assert.deepEqual(deviceCounts(items), { all: 2, online: 1, controlled: 1 })
    assert.deepEqual(splitHighlight('Living TV', 'tv'), [
        { text: 'Living ', matched: false },
        { text: 'TV', matched: true },
    ])
})

test('IPv6-only and multi-address devices keep a compact primary address', () => {
    const result = buildDeviceListItems([{
        deviceId: 'mac:aa:bb:cc:dd:ee:06', displayName: 'IPv6 sensor', online: true,
        lastSeenAt: '2026-09-24T02:00:00Z', mac: 'AA:BB:CC:DD:EE:06',
        identity: { kind: 'mac', scope: 'persistent' },
        addresses: {
            current: [
                { family: 6, address: 'fd5f:e357:7969::6', primary: true },
                { family: 6, address: 'fe80::6' },
            ],
            historical: [{ family: 6, address: 'fd5f:e357:7969::5' }],
        },
        connection: { kind: 'lan' },
    }], [])
    assert.equal(result[0].primaryAddress, 'fd5f:e357:7969::6')
    assert.equal(result[0].extraAddressCount, 2)
    assert.equal(selectDeviceListItems(result, 'all', 'fe80::6', 'recent').length, 1)
})

test('device scenes use recognizable signals and safely fall back to unknown', () => {
    assert.equal(detectDeviceScene({ hostname: 'living-room-tv' }), 'tv')
    assert.equal(detectDeviceScene({ displayName: 'Kids iPad' }), 'tablet')
    assert.equal(detectDeviceScene({ hostname: 'nas-server' }), 'storage')
    assert.equal(detectDeviceScene({ displayName: 'Front door 摄像头' }), 'camera')
    assert.equal(detectDeviceScene({ hostname: 'office-printer' }), 'printer')
    assert.equal(detectDeviceScene({ vendor: 'Unrecognized Devices Ltd.' }), 'unknown')
    assert.equal(deviceSceneIconPath('smart-home'), '/luci-static/quickstart/device-icons/smart-home.webp')
    assert.equal(deviceSceneIconPath('unknown'), '/luci-static/quickstart/device-icons/computer.webp')
})

test('M24 exposes 30 stable selectable icons with accessible labels', () => {
    assert.equal(deviceIconKeys.length, 30)
    assert.equal(new Set(deviceIconKeys).size, 30)
    for (const key of deviceIconKeys) {
        assert.ok(deviceIconLabels[key])
        assert.match(deviceIconPath(key), /^\/luci-static\/quickstart\/device-icons\/.+\.webp$/)
    }
})

test('confirmed ASUS manufacturer fallback presents a network device while stronger names win', () => {
    assert.equal(normalizeDeviceBrand('ASUSTek COMPUTER INC.'), 'ASUS')
    assert.equal(normalizeDeviceBrand('Unreviewed Manufacturer LLC'), '')
    assert.equal(detectDeviceScene({ vendor: 'ASUSTek COMPUTER INC.' }), 'network')
    assert.equal(detectDeviceScene({ displayName: 'ASUS RT-AX88U', vendor: 'ASUSTek COMPUTER INC.' }), 'network')
    assert.equal(detectDeviceScene({ displayName: 'Family workstation', vendor: 'ASUSTek COMPUTER INC.' }), 'computer')
    assert.equal(detectDeviceScene({ displayName: '', hostname: '', vendor: '' }), 'unknown')
})

test('M8 prefers the backend classification contract and safely adapts old responses', () => {
    const classified = resolveDeviceClassification({
        ...inventory[0],
        classification: { brand: 'ASUS', manufacturer: 'ASUSTek COMPUTER INC.', category: 'network', source: 'model', confidence: 'high' },
    })
    assert.deepEqual(classified, { brand: 'ASUS', manufacturer: 'ASUSTek COMPUTER INC.', category: 'network', source: 'model', confidence: 'high' })

    const legacy = resolveDeviceClassification({ ...inventory[0], displayName: '', hostname: '', vendor: 'ASUSTek COMPUTER INC.' })
    assert.equal(legacy.brand, 'ASUS')
    assert.equal(legacy.category, 'network')
    assert.equal(legacy.source, 'manufacturer_default')

    const fallback = resolveDeviceClassification({ ...inventory[0], displayName: '', hostname: '', vendor: '' })
    assert.equal(fallback.category, 'computer')
    assert.equal(fallback.source, 'fallback')
})

test('confirmed brand-only devices do not repeat brand and category below the generated title', () => {
    const unnamedManual = buildDeviceListItems([{
        ...inventory[0], displayName: '', hostname: '', vendor: 'ASUSTek COMPUTER INC.',
        classification: { brand: 'ASUS', manufacturer: 'ASUSTek COMPUTER INC.', category: 'network', source: 'manual', confidence: 'high' },
    }], [])[0]
    assert.deepEqual(deviceIdentityPresentation(unnamedManual), { showBrandInMeta: false, metaMode: 'classification-source' })

    const named = { ...unnamedManual, displayName: 'Living Room Router' }
    assert.deepEqual(deviceIdentityPresentation(named), { showBrandInMeta: true, metaMode: 'category' })
})
