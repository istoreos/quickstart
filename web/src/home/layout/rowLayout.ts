// Shared finite widget definitions and pure layout helpers.
export type LayoutVariant='rows'|'grid'|'free'
import {externalWidget} from '../cards/catalog'
export type BuiltinWidgetId='traffic'|'connection'|'networkInterface'|'applications'|'devices'|'system'|'storage'|'sharing'|'docker'|'downloads'|'remote'
export type WidgetId=BuiltinWidgetId|`plugin.${string}`
export const layoutVariants:Array<{id:LayoutVariant;key:string;zh:string;en:string;description:string}>=[
  {id:'rows',key:'R',zh:'有序行',en:'Proportional rows',description:'分组与比例可调，同排等高，窄屏按顺序拆行。'},
  {id:'grid',key:'G',zh:'自动网格',en:'Auto grid',description:'统一网格自动排列，可调整列数与密度。'},
  {id:'free',key:'F',zh:'自由画布',en:'Free canvas',description:'示例支持基本移动与缩放；不演示自动碰撞。'},
]
export const widgetDefinitions:Array<{id:WidgetId;zh:string;en:string;glyph:string;minWidth:number;minHeight:number;singleHeight:number;extra?:boolean}>=[
  {id:'traffic',zh:'网络流量',en:'Network traffic',glyph:'activity',minWidth:440,minHeight:280,singleHeight:300},
  {id:'connection',zh:'网络连接',en:'Connection & IP',glyph:'network',minWidth:300,minHeight:220,singleHeight:240},
  {id:'networkInterface',zh:'网络接口',en:'Network interfaces',glyph:'network',minWidth:280,minHeight:260,singleHeight:280,extra:true},
  {id:'applications',zh:'应用工具',en:'Apps & tools',glyph:'grid',minWidth:240,minHeight:176,singleHeight:240},
  {id:'devices',zh:'设备列表',en:'LAN devices',glyph:'devices',minWidth:400,minHeight:300,singleHeight:340},
  {id:'system',zh:'系统信息',en:'System information',glyph:'chip',minWidth:260,minHeight:260,singleHeight:280},
  {id:'storage',zh:'磁盘信息',en:'Disks',glyph:'disk',minWidth:380,minHeight:280,singleHeight:330},
  {id:'sharing',zh:'存储服务',en:'Storage services',glyph:'folder',minWidth:280,minHeight:280,singleHeight:300},
  {id:'docker',zh:'Docker',en:'Docker',glyph:'layers',minWidth:280,minHeight:250,singleHeight:260,extra:true},
  {id:'downloads',zh:'下载服务',en:'Downloads',glyph:'download',minWidth:280,minHeight:250,singleHeight:260,extra:true},
  {id:'remote',zh:'远程域名',en:'Remote access',glyph:'globe',minWidth:280,minHeight:250,singleHeight:280,extra:true},
]
export type PreviewRow={id:string;cards:WidgetId[];weights:number[];height:number}
export const previewRows=():PreviewRow[]=>[
  {id:'network',cards:['traffic','connection'],weights:[60,40],height:320},
  {id:'applications',cards:['applications'],weights:[100],height:152},
  {id:'devices',cards:['devices','system'],weights:[65,35],height:340},
  {id:'storage',cards:['storage','sharing'],weights:[60,40],height:310},
  {id:'services',cards:['docker','downloads','remote'],weights:[34,33,33],height:270},
]
export const widget=(id:WidgetId)=>id.startsWith('plugin.')?externalWidget(id as `plugin.${string}`):widgetDefinitions.find(item=>item.id===id)!
// Preserve row order; split only into consecutive subrows. No dense gap filling.
export function splitPreviewRows(rows:PreviewRow[],width:number,visible:WidgetId[],gap=18):PreviewRow[]{
  const result:PreviewRow[]=[]
  for(const row of rows){
    let cards:WidgetId[]=[],weights:number[]=[]
    const flush=()=>{if(!cards.length)return;const lone=cards.length===1;result.push({id:row.id+'-'+result.length,cards,weights,height:lone?(cards[0]==='applications'?(width<650?240:152):width<650?widget(cards[0]).singleHeight:row.height):row.height});cards=[];weights=[]}
    row.cards.forEach((id,index)=>{if(!visible.includes(id))return;const needed=[...cards,id].reduce((sum,item)=>sum+widget(item).minWidth,0)+gap*cards.length;if(cards.length&&needed>width)flush();cards.push(id);weights.push(row.weights[index])});flush()
  }
  return result
}
export type FreePosition={x:number;y:number;width:number;height:number}
export const freeDefaults:Record<WidgetId,FreePosition>={
  traffic:{x:0,y:0,width:65,height:320},connection:{x:67,y:0,width:33,height:300},
  applications:{x:0,y:338,width:38,height:250},devices:{x:40,y:338,width:60,height:340},
  system:{x:0,y:606,width:38,height:270},storage:{x:40,y:696,width:60,height:330},
  sharing:{x:0,y:894,width:38,height:310},networkInterface:{x:0,y:1520,width:100,height:280},docker:{x:40,y:1044,width:28,height:270},
  downloads:{x:70,y:1044,width:30,height:270},remote:{x:0,y:1222,width:38,height:280},
}
