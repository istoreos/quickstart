const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const vm = require('node:vm');
const ts = require('typescript');

const source = fs.readFileSync(path.join(__dirname, '../src/utils/utils.ts'), 'utf8');
const compiled = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 }
}).outputText;
const exportsObject = {};
vm.runInNewContext(compiled, {
    exports: exportsObject,
    require: (name) => {
        assert.equal(name, '/@/plugins/i18n');
        return { useGettextLazy: () => ({}) };
    }
});
const { byteToSize } = exportsObject;

for (const [name, value] of [
    ['omitted', undefined],
    ['null', null],
    ['NaN', NaN],
    ['infinity', Infinity],
    ['negative infinity', -Infinity],
    ['negative', -1],
    ['non-numeric', 'invalid']
]) {
    test(`byteToSize handles ${name} without NaN`, () => {
        assert.equal(byteToSize(value), '0 B');
    });
}

for (const [value, expected] of [
    [0, '0 B'], [42, '42 B'], [999, '999 B'],
    [1000, '1.0 KB'], [1500, '1.5 KB'],
    [1000000, '1.0 MB'], [1000000000, '1.0 GB']
]) {
    test(`byteToSize preserves formatting for ${value}`, () => {
        assert.equal(byteToSize(value), expected);
    });
}

test('legacy idle statistics response displays zero rates', () => {
    const item = JSON.parse('{"startTime":1710000000,"endTime":1710000005}');
    assert.equal(`${byteToSize(item.downloadSpeed)}/s`, '0 B/s');
    assert.equal(`${byteToSize(item.uploadSpeed)}/s`, '0 B/s');
});
