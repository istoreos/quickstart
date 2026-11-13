import test from 'node:test'
import assert from 'node:assert/strict'
import {homeModule} from './homeModule.mjs'
test('FN03 card scope has live readonly theme/form and aborts only its own pending requests',async()=>{
 const saved=Object.fromEntries(['window','document','MutationObserver','CustomEvent','fetch'].map(k=>[k,globalThis[k]]))
 const attrs=new Map()
 globalThis.document={createElement:()=>({}),createEvent:()=>({timeStamp:Date.now(),initEvent(){}}),body:{dataset:{},getAttribute:k=>attrs.get(k)||null,setAttribute:(k,v)=>attrs.set(k,v)},documentElement:{dataset:{},style:{}}}
 globalThis.MutationObserver=class{observe(){}}
 globalThis.CustomEvent=class{}
 globalThis.window={matchMedia:()=>({matches:false,addEventListener(){}}),localStorage:{getItem(){return null},setItem(){}},addEventListener(){},dispatchEvent(){}}
 try{
  const {createCardContext}=await homeModule('../src/home/cards/context.ts')
  const owner=createCardContext('/cgi-bin/luci/admin/services/fastnet/card','zh-cn','standard')
  assert.equal(owner.context.form,'standard');owner.setForm('summary');assert.equal(owner.context.form,'summary')
  assert.equal(owner.context.theme.__v_isReadonly,true)
  let closes=0
  owner.context.openDetails({render(){}},{message:'wide iframe'},{wide:true,headerAction:{href:'/cgi-bin/luci/admin/services/fastnet/open',label:'新窗口打开'},onClose(){closes++}})
  assert.equal(owner.details.value.wide,true);assert.equal(owner.details.value.headerAction.label,'新窗口打开');owner.closeDetails();owner.closeDetails();assert.equal(closes,1)
  window.token='fixture CSRF=+/'
  globalThis.fetch=async(url,options)=>{
   assert.equal(new URL(url,'http://router.invalid').searchParams.get('token'),window.token)
   assert.equal(options.headers.get('X-CSRF-Token'),window.token)
   assert.equal(options.headers.has('Authorization'),false)
   return new Response('{"ok":true}',{headers:{'Content-Type':'application/json'}})
  }
  await owner.context.request('/run',{method:'POST',body:'{"idempotencyKey":"fixture-key-0001"}'})
  globalThis.fetch=async(_url,options)=>{assert.equal(options.redirect,'manual');const response=new Response('');Object.defineProperty(response,'type',{value:'opaqueredirect'});return response}
  await assert.rejects(owner.context.request('/summary'),e=>e.status===401)
  let signal,calls=0
  globalThis.fetch=(_url,options)=>{calls++;signal=options.signal;return new Promise((_,reject)=>signal.addEventListener('abort',()=>reject(new Error('aborted')),{once:true}))}
  await assert.rejects(owner.context.request('/https://foreign/path'),/invalid_card_path/);assert.equal(calls,0)
  const pending=owner.context.request('/summary');assert.equal(signal.aborted,false);owner.dispose();assert.equal(signal.aborted,true);await assert.rejects(pending,/aborted/)
  await assert.rejects(owner.context.request('/summary'),/unmounted/)
 }finally{for(const[k,v]of Object.entries(saved)){if(v===undefined)delete globalThis[k];else globalThis[k]=v}}
})
