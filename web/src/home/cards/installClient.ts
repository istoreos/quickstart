import {installableCards} from './catalog'
import {cardRegistry} from './registryClient'
import type {CardInstaller,CardInstallState} from './installContext'
export function createCardInstaller(transport:typeof fetch=fetch,registry=cardRegistry,delay=(signal?:AbortSignal)=>new Promise<void>((resolve,reject)=>{
 const abort=()=>{clearTimeout(timer);reject(new DOMException('Aborted','AbortError'))},timer=setTimeout(()=>{signal?.removeEventListener('abort',abort);resolve()},3000)
 signal?.addEventListener('abort',abort,{once:true});if(signal?.aborted)abort()
})) :CardInstaller{
 const tr=(zh:string,en:string)=>typeof window!=='undefined'&&window.vue_lang==='en'?en:zh
 const packageName=(id:string)=>{const item=installableCards.find(c=>c.id===id);if(!item)throw Error('Unsupported card');return item.package}
 async function request(action:string,id:string,signal?:AbortSignal){
  const owner=new AbortController(),abort=()=>owner.abort(),timer=setTimeout(abort,8000);signal?.addEventListener('abort',abort,{once:true});if(signal?.aborted)abort()
  try{
   const r=await transport('/cgi-bin/luci/istore/app/'+action+'/',{method:'POST',credentials:'same-origin',redirect:'error',cache:'no-store',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:packageName(id)}),signal:owner.signal})
   if(!r.ok||!r.headers.get('Content-Type')?.includes('application/json'))throw Error([401,403].includes(r.status)?tr('登录已失效或权限不足，请重新登录','Session expired or access denied. Sign in again.'):tr('无法读取安装状态，请稍后重试','Unable to read installation status. Try again.'))
   const raw=await r.text();if(raw.length>8192)throw Error('Invalid installation response');const v=JSON.parse(raw)
   if(v.error||v.success!==undefined&&v.success!==0)throw Error(tr('安装状态读取失败，请检查软件中心','Unable to confirm installation. Check the software center.'))
   return v
  }finally{clearTimeout(timer);signal?.removeEventListener('abort',abort)}
 }
 async function inspect(id:string,signal?:AbortSignal):Promise<CardInstallState>{
  const v=await request('check',id,signal),installation=v.result?.installation
  if(!['idle','installing','completed','failed','unknown'].includes(installation))return 'unknown'
  if(installation==='installing')return 'installing'
  if(installation==='unknown')return 'unknown'
  if(installation==='failed')return 'failed'
  if(v.result?.status==='uninstalled')return 'uninstalled'
  if(v.result?.status!=='installed')return 'unknown'
  return (await registry.read(true)).some(c=>c.id===id)?'ready':'update_required'
 }
 return {inspect,async install(id,signal){
  const before=await inspect(id,signal);if(before==='ready')return
  if(before==='unknown')throw Error(tr('无法确认安装状态，请先在软件中心检查或升级 Quickstart','Installation status is unknown. Check the software center or update Quickstart.'))
  if(before!=='installing'){
   // A lost acknowledgement starts observation, never a second install POST.
   try{await request('install',id,signal)}catch(e){if(signal?.aborted)throw e}
  }
  for(let i=0;i<60;i++){
   await delay(signal);const current=await inspect(id,signal)
   if(current==='ready')return
   if(current==='update_required')throw Error(tr('插件已安装，但当前版本尚未提供可用卡片；请在软件中心升级或修复 FastNet','The installed version has no usable card. Update or repair FastNet in the software center.'))
   if(current==='failed')throw Error(tr('安装失败，请检查软件源后重试','Installation failed. Check the package source before retrying.'))
   if(current==='unknown')throw Error(tr('安装结果尚未确认，请先在软件中心检查','Installation is unconfirmed. Check the software center first.'))
  }
  throw Error(tr('安装结果尚未确认，请重新读取状态；后台安装可能仍在继续','Installation is unconfirmed. Reload its status; installation may still be running.'))
 }}
}
export const cardInstaller=createCardInstaller()
