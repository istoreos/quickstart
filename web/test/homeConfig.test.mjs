import assert from 'node:assert/strict'
import test from 'node:test'
import fs from 'node:fs'
import {homeModule} from './homeModule.mjs'
const {configValue}=await homeModule('../src/home/configContract.ts')
const {recommended,naturalRows}=await homeModule('../src/home/layout/orderedLayout.ts')
const {defaultPreferences}=await homeModule('../src/home/layout/widgetPresentation.ts')
const fixture=JSON.parse(fs.readFileSync(new URL('../../docs/contracts/home-config.json',import.meta.url)))
for(const item of fixture.cases)test('shared home contract: '+item.name,()=>{
 const value=structuredClone(fixture[item.domain])
 for(const change of item.changes){const path=[...change.path],last=path.pop();let target=value;for(const key of path)target=target[key];target[last]=change.value}
 const before=JSON.stringify(value)
 if(item.valid)assert.deepEqual(configValue(item.domain,value),value)
 else assert.throws(()=>configValue(item.domain,value))
 assert.equal(JSON.stringify(value),before,'validation changed user input')
})
test('published default matches approved pure model, without browser globals',()=>{
 assert.deepEqual({schemaVersion:4,rows:naturalRows(recommended()),widgetPreferences:defaultPreferences()},fixture.layout)
})
test('server and frontend bounds cover row, pin and link maximums',()=>{
 const pins=structuredClone(fixture.launcher);pins.favorites=Array.from({length:24},(_,i)=>'app-'+i)
 assert.doesNotThrow(()=>configValue('launcher',pins));pins.favorites.push('extra');assert.throws(()=>configValue('launcher',pins))
 const links=structuredClone(fixture.launcher);links.links=Array.from({length:64},(_,i)=>({...links.links[0],id:'user-link:'+i}))
 assert.doesNotThrow(()=>configValue('launcher',links));links.links.push({...links.links[0],id:'user-link:extra'});assert.throws(()=>configValue('launcher',links))
})
