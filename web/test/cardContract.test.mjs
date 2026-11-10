import test from 'node:test'
import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'
import {homeModule} from './homeModule.mjs'
const fixture=JSON.parse(await readFile(new URL('../../docs/contracts/fastnet-card.json',import.meta.url)))
const api=await homeModule('../src/home/cards/contract.ts')
test('FN01-T01 registered card validates; unknown fields, versions and executable paths fail',()=>{
 assert.deepEqual(api.cardManifest(fixture.manifest),fixture.manifest)
 for(const patch of [{id:'traffic'},{schemaVersion:2},{abi:'v2'},{entry:'https://other/card.12345678.js'},{entry:'/luci-static/fastnet/../card.12345678.js'},{entry:'/luci-static/fastnet/card.12345678.js?x=1'},{entry:'/luci-static/fastnet/card.%32.js'},{forms:{standard:{minWidth:0,minHeight:1}}},{stacked:['standard']},{unknown:1}])assert.throws(()=>api.cardManifest({...fixture.manifest,...patch}))
 for(const key of Object.keys(fixture.manifest)){const m={...fixture.manifest};delete m[key];assert.throws(()=>api.cardManifest(m),key)}
 assert.deepEqual(api.cardManifests([fixture.manifest,fixture.manifest]),[])
 assert.equal(api.cardManifests([{...fixture.manifest,abi:'future'},fixture.manifest]).length,0)
 assert.throws(()=>api.cardManifests(Array(33).fill(fixture.manifest)))
})
test('FN01-T02 absent and unavailable results preserve independent detection times and units',()=>{
 const r=fixture.summary.latest
 assert.equal(r.download.value/1e6,950)
 assert.equal(r.ipv6.status,'unavailable');assert.equal(r.ipv6.value,null);assert.equal(r.ipv6.error,null)
 assert.notEqual(r.nat.measuredAt,r.measuredAt)
 assert.notEqual(r.ipv6.measuredAt,r.measuredAt)
 assert.ok(Date.parse(r.measuredAt))
})
test('FN01-T03 task transitions cannot resurrect terminal or cancelled work',()=>{
 for(const [from,tos]of Object.entries(fixture.transitions))for(const to of tos)assert.equal(api.taskTransition(from,to),true)
 for(const from of fixture.terminal)for(const to of ['running',...fixture.terminal])assert.equal(api.taskTransition(from,to),false)
 assert.equal(api.taskTransition('cancel_requested','succeeded'),false)
 assert.equal(Date.parse(fixture.task.deadlineAt)-Date.parse(fixture.task.startedAt),120000)
})
test('FN01-T04 version matrix specifies every expected downgrade and recovery',()=>{
 assert.equal(fixture.compatibility.length,7)
 assert.ok(fixture.compatibility.every(c=>c.expected.length>20))
 assert.equal(fixture.vue,'3.5.44');assert.equal(fixture.compiler,fixture.vue)
})
