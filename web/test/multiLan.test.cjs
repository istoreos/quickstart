const assert = require('node:assert/strict');
const test = require('node:test');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const ts = require('typescript');
function load(file, requireMock = () => ({}), globals = {}) {
 const output = ts.transpileModule(fs.readFileSync(path.join(__dirname, '../src', file), 'utf8'), {compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2020}}).outputText;
 const exports = {};vm.runInNewContext(output,{exports,require:requireMock,Headers,AbortSignal,AbortController,setTimeout,clearTimeout,fetch,Response,...globals});return exports;
}
test('immutable clients retain targets; aggregate cannot write',async()=>{
 const calls=[];
 const api=load('request/request.ts',name=>name==='./do'?{Do:async(url,data)=>{calls.push({url,data});return {data:{result:{}}}}}:{useGettextLazy:()=>({})});
 const a=api.createDeviceManagement('lan1'), b=api.createDeviceManagement('lan2');
 await Promise.all([a.devicePolicyV2.GET('same'),b.devicePolicyV2.GET('same')]);
 await a.devicePolicyV2.POST({deviceId:'same'});
 assert.match(calls[0].url,/deviceId=same&network=lan1$/);assert.match(calls[1].url,/network=lan2$/);assert.match(calls[2].url,/network=lan1$/);
 await assert.rejects(api.createDeviceManagement('all').devicePolicyV2.POST({deviceId:'same'}));assert.equal(calls.length,3);
 await api.DeviceMangement.deviceInventoryV2.GET();assert.ok(!calls[3].url.includes('network='));
});
test('network discovery falls back only for explicitly missing APIs',async()=>{
 const {discoverLANs}=load('pages/device/lanDiscovery.ts');
 for(const status of [404,501])assert.equal((await discoverLANs(async()=>new Response('',{status}))).legacy,true);
 for(const status of [401,403,500,503])await assert.rejects(discoverLANs(async()=>new Response('',{status})));
 await assert.rejects(discoverLANs(async()=>{throw new Error('timeout')}));
 await assert.rejects(discoverLANs(async()=>Response.json({result:{networks:'invalid'}})));
 const current=await discoverLANs(async()=>Response.json({result:{networks:[{networkId:'home',displayName:'Home',device:'br-home',state:'up'}]}}));
 assert.equal(current.networks[0].networkId,'home');assert.equal(current.legacy,false);
});
test('identical device identities remain distinct across networks',()=>{
 const {deviceScopeKey}=load('pages/device/lanIdentity.ts');
 const a={networkId:'lan1',deviceId:'mac:aa'},b={networkId:'lan2',deviceId:'mac:aa'};
 const map=new Map([[deviceScopeKey(a),1],[deviceScopeKey(b),2]]);assert.equal(map.size,2);assert.equal(map.get(deviceScopeKey(a)),1);
});

test('network discovery deadline includes a stalled response body',async()=>{
 const {discoverLANs}=load('pages/device/lanDiscovery.ts',undefined,{setTimeout:(fn)=>setTimeout(fn,5)});
 await assert.rejects(discoverLANs(async(_url,init)=>({status:200,ok:true,json:()=>new Promise((_resolve,reject)=>init.signal.addEventListener('abort',()=>reject(new DOMException('Aborted','AbortError'))))})),error=>error.code==='timeout');
});
