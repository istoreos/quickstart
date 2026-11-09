// Design-only registration: production's catalog and bundle stay unchanged.
import {reactive} from 'vue'
import {widgetDefinitions,freeDefaults,type WidgetId} from '../home/layout/rowLayout'
import {presentationDefinitions} from '../home/layout/widgetPresentation'
import {moduleCapabilities} from '../home/layout/moduleCapability'
import {compactIds} from '../home/layout/orderedLayout'
// The preview extends the finite production catalog at its bootstrap boundary.
export const fastnetWidgetId='fastnet' as WidgetId
widgetDefinitions.push({id:fastnetWidgetId,zh:'外网测速',en:'Internet speed',glyph:'activity',minWidth:280,minHeight:230,singleHeight:250,extra:true})
freeDefaults[fastnetWidgetId]={x:40,y:1340,width:38,height:250}
presentationDefinitions.push({id:fastnetWidgetId,forms:['standard','summary'],stacked:true})
moduleCapabilities.push({id:fastnetWidgetId,minWidth:280,minHeight:230,mobileMinWidth:240,stacked:true,compactHeight:148,specs:['full','standard','compact'],summary:'summary',forms:['standard','summary']})
compactIds.push(fastnetWidgetId)

export const fastnetScenes=[{id:'ready',label:'已有上次结果'},{id:'empty',label:'尚未测速'},{id:'missing',label:'组件未安装'},{id:'stale',label:'上次结果较早'},{id:'partial',label:'IPv6 异常'},{id:'stopped',label:'服务未启用'},{id:'error',label:'测速失败与重试'},{id:'install-error',label:'安装失败与重试'}]
export const fastnet=reactive({scene:'ready',installed:true,enabled:true,running:false,installing:false,progress:0,down:'942',up:'108',time:'今天 14:22',nat:'全锥形',ipv6:'可用',note:'',error:'',attempts:0,installAttempts:0})
let timer:ReturnType<typeof setInterval>|undefined
export function setFastnetScene(value:string){
  clearInterval(timer)
  const scene=fastnetScenes.some(s=>s.id===value)?value:'ready',empty=['empty','missing','install-error'].includes(scene)
  Object.assign(fastnet,{scene,installed:!['missing','install-error'].includes(scene),enabled:scene!=='stopped',running:false,installing:false,progress:0,down:empty?'':'942',up:'108',time:scene==='stale'?'3 天前 14:22':'今天 14:22',nat:empty?'未检测':'全锥形',ipv6:scene==='partial'?'部分可用':empty?'未检测':'可用',note:'',error:'',attempts:0,installAttempts:0})
}
setFastnetScene(new URLSearchParams(location.search).get('fastnet')||'ready')
export const fastnetStages=['准备测速','测试下载','测试上传','检测 NAT / IPv6','保存结果']
export function startFastnetTest(){
  if(fastnet.running||fastnet.installing||!fastnet.installed||!fastnet.enabled)return
  fastnet.running=true;fastnet.progress=0;fastnet.error='';fastnet.note='';fastnet.attempts++
  timer=setInterval(()=>{fastnet.progress++;if(fastnet.progress<5)return;clearInterval(timer);fastnet.running=false
    if(fastnet.scene==='error'&&fastnet.attempts===1){fastnet.error='测速节点连接超时，请重试。';return}
    Object.assign(fastnet,{down:'946',up:'109',time:'刚刚 · '+new Date().toLocaleTimeString('zh-CN',{hour:'2-digit',minute:'2-digit'}),nat:'全锥形',ipv6:'可用',note:'测速完成'})
  },900)
}
export function stopFastnetTest(){clearInterval(timer);fastnet.running=false;fastnet.note='已停止，保留上次结果'}
export function installFastnet(){
  if(fastnet.installing||fastnet.installed)return
  fastnet.installing=true;fastnet.progress=0;fastnet.error='';fastnet.installAttempts++
  timer=setInterval(()=>{fastnet.progress++;if(fastnet.progress<3)return;clearInterval(timer);fastnet.installing=false
    if(fastnet.scene==='install-error'&&fastnet.installAttempts===1){fastnet.error='组件下载失败，请检查软件源后重试。';return}
    fastnet.installed=true;fastnet.enabled=true;fastnet.note='已安装，点击测速开始第一次检测'
  },900)
}
