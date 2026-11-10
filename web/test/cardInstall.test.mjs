import test from 'node:test'
import assert from 'node:assert/strict'
import {homeModule} from './homeModule.mjs'
const {createCardInstaller}=await homeModule('../src/home/cards/installClient.ts')
const reply=(value,status=200)=>new Response(JSON.stringify(value),{status,headers:{'Content-Type':'application/json'}})
test('FN09 unknown/permission/old backend never become uninstalled or install',async()=>{
 for(const result of [{status:'uninstalled'}, {status:'uninstalled',installation:'unknown'}]){
  let installs=0;const c=createCardInstaller(async u=>{if(u.includes('/install/'))installs++;return reply({result})},{read:async()=>[]},async()=>{})
  assert.equal(await c.inspect('plugin.fastnet'),'unknown');await assert.rejects(c.install('plugin.fastnet'));assert.equal(installs,0)
 }
 const c=createCardInstaller(async()=>reply({},403),{read:async()=>[]});await assert.rejects(c.inspect('plugin.fastnet'),/权限/)
})
test('FN09 a lost install acknowledgement is observed without replaying POST',async()=>{
 let posts=0,installed=false;const c=createCardInstaller(async u=>{
  if(u.includes('/install/')){posts++;installed=true;throw Error('acknowledgement lost')}
  return reply({result:{status:installed?'installed':'uninstalled',installation:installed?'completed':'idle'}})
 },{read:async()=>installed?[{id:'plugin.fastnet'}]:[]},async()=>{})
 assert.equal(await c.inspect('plugin.fastnet'),'uninstalled');await c.install('plugin.fastnet');assert.equal(posts,1)
})
test('FN09 an already running installation only observes; installed does not install again',async()=>{
 let checks=0,posts=0;const c=createCardInstaller(async u=>{if(u.includes('/install/'))posts++;checks++;return reply({result:{status:checks<2?'uninstalled':'installed',installation:checks<2?'installing':'completed'}})},{read:async()=>[{id:'plugin.fastnet'}]},async()=>{})
 await c.install('plugin.fastnet');assert.equal(posts,0);await c.install('plugin.fastnet');assert.equal(posts,0)
})
test('FN09 completed old/incomplete source reports repair instead of looping or running a test',async()=>{
 let posts=0;const c=createCardInstaller(async u=>{if(u.includes('/install/')){posts++;return reply({success:0})}return reply({result:{status:'installed',installation:'completed'}})},{read:async()=>[]},async()=>{})
 assert.equal(await c.inspect('plugin.fastnet'),'update_required');await assert.rejects(c.install('plugin.fastnet'),/当前版本/);assert.equal(posts,1)
})
test('FN09 closing observation cancels its request, not the installation',async()=>{
 const owner=new AbortController();let aborted=false,posts=0
 const c=createCardInstaller(async(u,options)=>{if(u.includes('/install/'))posts++;return new Promise((resolve,reject)=>{options.signal.addEventListener('abort',()=>{aborted=true;reject(Error('aborted'))},{once:true})})})
 const pending=c.inspect('plugin.fastnet',owner.signal);owner.abort();await assert.rejects(pending);assert.equal(aborted,true);assert.equal(posts,0)
})
