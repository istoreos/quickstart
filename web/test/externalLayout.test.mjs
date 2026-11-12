import test from 'node:test'
import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'
import {execFileSync} from 'node:child_process'
import {createRequire} from 'node:module'
import {fileURLToPath} from 'node:url'
import {homeModule} from './homeModule.mjs'
const {configValue}=await homeModule('../src/home/configContract.ts')
const {migrateDocument,documentOf,planDocument}=await homeModule('../src/home/layout/homeDocument.ts')
const {moveToNewRow}=await homeModule('../src/home/layout/slotLayout.ts')
const fixture=JSON.parse(await readFile(new URL('../../docs/contracts/fastnet-card.json',import.meta.url)))
const migration=JSON.parse(await readFile(new URL('../../docs/contracts/home-layout-v4.json',import.meta.url)))
test('FN04 v3 migrates without losing slots, proportions or preferences, and does not mutate input',()=>{
 const original=JSON.stringify(migration.legacy)
 assert.deepEqual(configValue('layout',migration.legacy),migration.migrated)
 assert.equal(JSON.stringify(migration.legacy),original)
 assert.deepEqual(migrateDocument(migration.legacy).document.widgetPreferences,migration.migrated.widgetPreferences)
})
test('FN04 missing plugins are valid stored identities; moving/hiding/summary preserves preferences',()=>{
 const layout=configValue('layout',fixture.layoutV4)
 assert.deepEqual(layout,fixture.layoutV4)
 assert.ok(migrateDocument(layout).document)
 const before=documentOf(layout.rows,layout.widgetPreferences),moved=moveToNewRow(before.rows,'plugin.fastnet',before.rows[0].id)
 const plan=planDocument(before,moved,{width:1160,favoriteCount:4,favoritesKey:''})
 assert.notEqual(plan.status,'blocked')
 const value={schemaVersion:4,rows:plan.next.rows,widgetPreferences:plan.next.widgetPreferences}
 assert.equal(configValue('layout',value).rows[0].columns[0].cards[0],'plugin.fastnet')
 value.widgetPreferences['plugin.fastnet'].mode='summary';assert.doesNotThrow(()=>configValue('layout',value))
 value.rows=value.rows.filter(r=>!r.columns.some(c=>c.cards.includes('plugin.fastnet')));assert.ok(configValue('layout',value).widgetPreferences['plugin.fastnet'])
})
test('FN04 layouts cannot carry URLs, malformed IDs, duplicate instances or unsupported forms',()=>{
 for(const change of [v=>v.entry=fixture.manifest.entry,v=>v.widgetPreferences['plugin.fastnet'].mode='iframe',v=>v.rows.push(structuredClone(v.rows.at(-1))),v=>v.widgetPreferences['plugin.fastnet'].iconRows=1,v=>v.widgetPreferences['plugin.bad/id']=v.widgetPreferences['plugin.fastnet'],v=>v.schemaVersion=3]){
  const v=structuredClone(fixture.layoutV4);change(v);assert.throws(()=>configValue('layout',v))
 }
 const v=structuredClone(migration.migrated);for(let i=0;i<33;i++)v.widgetPreferences['plugin.card'+i]={mode:'auto',iconRows:0,ruleVersion:2};assert.throws(()=>configValue('layout',v))
})
test('FN04 the original v3 client explicitly rejects v4 as future, rather than damaged',async()=>{
 const source=execFileSync('git',['show','81018a2:web/src/home/configContract.ts'],{encoding:'utf8'})
 const require=createRequire(import.meta.url),{buildSync}=require('esbuild')
 const result=buildSync({stdin:{contents:source,resolveDir:fileURLToPath(new URL('../src/home/',import.meta.url)),loader:'ts'},bundle:true,write:false,format:'esm',platform:'node',logLevel:'silent'})
 const old=await import('data:text/javascript;base64,'+Buffer.from(result.outputFiles[0].text).toString('base64'))
 assert.throws(()=>old.configValue('layout',fixture.layoutV4),e=>e.code==='future')
})
