import test from 'node:test'
import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'
import {homeModule} from './homeModule.mjs'
const {createCardRegistry}=await homeModule('../src/home/cards/registryClient.ts')
const {manifest}=JSON.parse(await readFile(new URL('../../docs/contracts/fastnet-card.json',import.meta.url)))
const response=cards=>new Response(JSON.stringify({schemaVersion:1,cards,issues:[]}),{headers:{'Content-Type':'application/json'}})
test('FN04 discovery merges callers, caches metadata, and explicitly rediscovers install/remove',async()=>{
 let calls=0,current=[]
 const client=createCardRegistry(async()=>{calls++;return response(current)})
 const a=client.read(),b=client.read();assert.equal(a,b);assert.deepEqual(await a,[])
 current=[manifest];assert.deepEqual(await client.read(),[]);assert.equal(calls,1)
 assert.deepEqual(await client.read(true),[manifest]);assert.equal(calls,2)
 current=[];assert.deepEqual(await client.read(true),[]);assert.equal(calls,3)
})
test('FN04 unknown/login/oversize/invalid registry is never reported as a successful empty list',async()=>{
 for(const res of [new Response('<html>Login</html>'),new Response('{}',{status:403}),new Response('x'.repeat(65537),{headers:{'Content-Type':'application/json'}}),new Response('{"schemaVersion":2,"cards":[]}',{headers:{'Content-Type':'application/json'}})]){
  const client=createCardRegistry(async()=>res);await assert.rejects(client.read())
 }
 let fail=true;const client=createCardRegistry(async()=>{if(fail)throw Error('offline');return response([manifest])})
 await assert.rejects(client.read());fail=false;assert.deepEqual(await client.read(),[manifest])
})
