import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const testDir = path.dirname(fileURLToPath(import.meta.url))
const iconDir = path.join(testDir, '../public/luci-static/quickstart/device-icons')

test('M10 release contains only the 12 governed generic scene icons', () => {
    const files = fs.readdirSync(iconDir).sort()
    assert.deepEqual(files, [
        'camera.webp', 'computer.webp', 'gaming.webp', 'network.webp', 'phone.webp', 'printer.webp',
        'smart-home.webp', 'storage.webp', 'tablet.webp', 'tv.webp', 'unknown.webp', 'wearable.webp',
    ])
    for (const file of files) {
        assert.ok(fs.statSync(path.join(iconDir, file)).size > 100, `${file} is unexpectedly empty`)
        assert.doesNotMatch(file, /asus|apple|samsung|xiaomi|huawei|logo|brand/i)
    }
})

test('M10 keeps generation provenance and trademark boundaries with the assets', () => {
    const governance = fs.readFileSync(path.join(testDir, '../../docs/device-scene-icons.md'), 'utf8')
    assert.match(governance, /统一提示词/)
    assert.match(governance, /无文字、字母、商标、水印或品牌特征/)
    assert.match(governance, /通用电脑图标/)
})
