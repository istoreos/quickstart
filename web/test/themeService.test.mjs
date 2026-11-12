import assert from 'node:assert/strict'
import test from 'node:test'
import {homeModule} from './homeModule.mjs'

test('FN02 theme owner persists, follows OS/LuCI/storage, and releases 50 subscriptions',async()=>{
 const original=Object.fromEntries(['window','document','MutationObserver','CustomEvent'].map(k=>[k,globalThis[k]]))
 const attrs=new Map(),store=new Map();let mutation,storageListener,mediaListener,mediaAdds=0,storageAdds=0,blocked=false
 const body={dataset:{},getAttribute:k=>attrs.get(k)||null,setAttribute(k,v){attrs.set(k,v);queueMicrotask(()=>mutation?.())}}
 const media={matches:true,addEventListener(k,fn){assert.equal(k,'change');mediaAdds++;mediaListener=fn}}
 globalThis.document={body,documentElement:{dataset:{},style:{}}}
 globalThis.CustomEvent=class{constructor(type,init){this.type=type;this.detail=init.detail}}
 globalThis.MutationObserver=class{constructor(fn){mutation=fn}observe(){}}
 globalThis.window={matchMedia:()=>media,localStorage:{getItem(k){if(blocked)throw Error('denied');return store.get(k)||null},setItem(k,v){if(blocked)throw Error('denied');store.set(k,v)}},addEventListener(k,fn){assert.equal(k,'storage');storageAdds++;storageListener=fn},dispatchEvent(){}}
 try{
  const theme=await homeModule('../src/utils/theme.ts')
  assert.deepEqual(theme.initializeTheme(),{preference:'system',effective:'dark'})
  for(const explicit of ['light','dark']){
   theme.setThemePreference(explicit)
   for(const dark of [true,false]){media.matches=dark;mediaListener();assert.equal(theme.getTheme(),explicit)}
  }
  theme.setThemePreference('system')
  for(const dark of [true,false,true]){media.matches=dark;mediaListener();assert.equal(theme.getTheme(),dark?'dark':'light');assert.equal(attrs.get('theme'),'normal')}
  storageListener({key:'quickstart.theme.preference',newValue:'light'});await Promise.resolve();assert.equal(theme.getThemePreference(),'light')
  body.setAttribute('theme','dark');await Promise.resolve();assert.equal(theme.getThemePreference(),'dark');assert.equal(store.get('quickstart.theme.preference'),'dark')
  let calls=0
  for(let i=0;i<50;i++){const unsubscribe=theme.subscribeTheme(()=>calls++);unsubscribe();theme.setThemePreference(i%2?'light':'dark')}
  assert.equal(calls,50,'unsubscribed listeners receive no further events')
  assert.equal(mediaAdds,1);assert.equal(storageAdds,1)
  blocked=true;assert.doesNotThrow(()=>theme.setThemePreference('system'));assert.equal(theme.getThemePreference(),'system')
  await Promise.resolve() // Deliver pending attribute notifications before restoring the test DOM.
 }finally{for(const[k,v]of Object.entries(original)){if(v===undefined)delete globalThis[k];else globalThis[k]=v}}
})
