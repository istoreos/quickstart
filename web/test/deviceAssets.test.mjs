import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const testDir = path.dirname(fileURLToPath(import.meta.url))
const iconDir = path.join(testDir, '../public/luci-static/quickstart/device-icons')

test('M24 release contains 30 governed generic device icons', () => {
    const manifest = JSON.parse(fs.readFileSync(path.join(iconDir, 'manifest.json'), 'utf8'))
    assert.equal(manifest.icons.length, 30)
    assert.equal(new Set(manifest.icons.map(icon => icon.key)).size, 30)
    assert.equal(manifest.reviewRequiredBeforePublicRelease, true)
    const files = fs.readdirSync(iconDir).filter(file => file.endsWith('.webp')).sort()
    assert.deepEqual(files, manifest.icons.map(icon => icon.file).sort())
    for (const file of files) {
        assert.ok(fs.statSync(path.join(iconDir, file)).size > 100, `${file} is unexpectedly empty`)
        assert.doesNotMatch(file, /asus|apple|samsung|xiaomi|huawei|logo|brand/i)
    }
})

test('M24 keeps generation provenance, accessible labels and trademark boundaries', () => {
    const governance = fs.readFileSync(path.join(testDir, '../../docs/device-scene-icons.md'), 'utf8')
    assert.match(governance, /统一提示词/)
    assert.match(governance, /无文字、字母、商标、水印或品牌特征/)
    assert.match(governance, /通用电脑图标/)
    assert.match(governance, /每个新增形态均使用独立生成调用/)
    const manifest = JSON.parse(fs.readFileSync(path.join(iconDir, 'manifest.json'), 'utf8'))
    for (const icon of manifest.icons) {
        assert.ok(icon.label)
        assert.ok(icon.category)
        assert.match(icon.generated, /^2026-09-(24|25)$/)
    }
})
