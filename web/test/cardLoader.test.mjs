import test from 'node:test'
import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'
import {homeModule} from './homeModule.mjs'
const {manifest}=JSON.parse(await readFile(new URL('../../docs/contracts/fastnet-card.json',import.meta.url)))
const {createCardLoader}=await homeModule('../src/home/cards/loader.ts')
test('FN03 concurrent loads and later mounts share one validated module',async()=>{
 let calls=0,finish
 const component={setup(){}}
 const load=createCardLoader(()=>{calls++;return new Promise(resolve=>finish=resolve)})
 const a=load(manifest),b=load(manifest);assert.equal(a,b);await Promise.resolve();assert.equal(calls,1)
 finish({apiVersion:1,default:component});assert.equal(await a,component);assert.equal(await load(manifest),component);assert.equal(calls,1)
})
test('FN03 failed import retries with bounded deterministic URLs and rejects incompatible exports',async()=>{
 const urls=[]
 const load=createCardLoader(async url=>{urls.push(url);return {apiVersion:2,default:{setup(){}}}})
 for(let i=0;i<4;i++)await assert.rejects(load(manifest),/incompatible/)
 await assert.rejects(load(manifest),/reload_required/)
 assert.equal(urls.length,4);assert.equal(urls[1],manifest.entry+'?qs-card-retry=1')
 let attempts=0
 const recover=createCardLoader(async()=>{if(!attempts++)throw Error('404');return {apiVersion:1,default:{setup(){}}}})
 await assert.rejects(recover(manifest),/404/);await recover(manifest);assert.equal(attempts,2)
 const invalid=createCardLoader(async()=>({apiVersion:1,default:{}}));await assert.rejects(invalid(manifest),/incompatible/)
})
test('FN03 slow imports expire, unregistered addresses and unknown ABI never execute',async()=>{
 let count=0
 const load=createCardLoader(()=>{count++;return new Promise(()=>{})},5)
 await assert.rejects(load(manifest),/timeout/)
 assert.throws(()=>load({...manifest,entry:'https://foreign/card.js'}));assert.throws(()=>load({...manifest,abi:'unknown'}));assert.equal(count,1)
})
