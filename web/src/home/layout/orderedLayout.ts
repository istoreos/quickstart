// Shared ordered layout model. No browser globals, requests or persistence.
import {widget,type WidgetId} from './rowLayout'
import {capability,moduleCapabilities} from './moduleCapability'
import {placementMinimum,resolvePresentation,type WidgetPreferences} from './widgetPresentation'
export type Template='single'|'pair'|'triple'|'left-big'|'right-big'
export type Column={weight:number;cards:Array<WidgetId|null>;split:number;upperShare?:number}
export type OrderedRow={id:string;template:Template;height:number;heightMode?:'content'|'chart';minHeight?:number;columns:Column[]}
// Preserve legacy dimensions for recovery, but never use them to constrain
// ordinary content in the v2 ordered homepage.
export function naturalRows(rows:OrderedRow[]):OrderedRow[]{return clone(rows).map(row=>({...row,heightMode:row.columns.some(c=>c.cards.includes('traffic'))?'chart':'content'}))}
export const templates:Array<{id:Template;zh:string;en:string;slots:number[]}>=([
  {id:'single',zh:'整行',en:'Single',slots:[1]},
  {id:'pair',zh:'两列',en:'Two columns',slots:[1,1]},
  {id:'triple',zh:'三列',en:'Three columns',slots:[1,1,1]},
  {id:'left-big',zh:'左单右双',en:'Single left, stacked right',slots:[1,2]},
  {id:'right-big',zh:'左双右单',en:'Stacked left, single right',slots:[2,1]},
])
export const compactIds:WidgetId[]=moduleCapabilities.filter(c=>c.stacked).map(c=>c.id)
export const clone=<T>(value:T):T=>JSON.parse(JSON.stringify(value))
let serial=0
export function makeRow(ids:WidgetId[]=[],template:Template='single',height=320,id='row-'+(++serial)):OrderedRow{
  let cursor=0
  const columns=templates.find(t=>t.id===template)!.slots.map((n,index)=>({weight:template==='triple'?[34,33,33][index]:template==='single'?100:index===0?60:40,split:50,cards:Array.from({length:n},()=>ids[cursor++]||null)}))
  return {id,template,height,columns}
}
export const recommended=():OrderedRow[]=>[
  makeRow(['traffic','connection','networkInterface'],'left-big',320,'network'),
  {...makeRow(['devices','applications'],'pair',340,'devices'),columns:[{weight:50,cards:['devices'],split:50},{weight:50,cards:['applications'],split:50}]},
  {...makeRow(['storage','sharing'],'pair',330,'storage'),columns:[{weight:50,cards:['storage'],split:50},{weight:50,cards:['sharing'],split:50}]},
  makeRow(['docker','downloads','remote'],'triple',280,'services'),
  makeRow(['system'],'single',280,'system'),
]
export const visibleIds=(rows:OrderedRow[])=>rows.flatMap(r=>r.columns.flatMap(c=>c.cards.filter((id):id is WidgetId=>Boolean(id))))
export function cardMinimum(id:WidgetId,stacked:boolean){return stacked&&(compactIds.includes(id)||id.startsWith('plugin.'))?capability(id).compactHeight:capability(id).minHeight}
export function rowMinimum(row:OrderedRow){return Math.max(0,...row.columns.map(c=>{
  const cards=c.cards.filter((id):id is WidgetId=>Boolean(id));return cards.reduce((sum,id)=>sum+cardMinimum(id,c.cards.length>1),0)+Math.max(0,cards.length-1)*18
}))}
export function layoutIssue(rows:OrderedRow[]){
  const ids=visibleIds(rows)
  if(new Set(ids).size!==ids.length)return 'duplicate'
  if(rows.some(row=>row.columns.some(c=>c.cards.length===2&&c.cards.some(id=>id&&!compactIds.includes(id)&&!id.startsWith('plugin.')))))return 'compact'
  for(const row of rows){const required=Math.ceil(Math.max(rowMinimum(row),...row.columns.flatMap(c=>c.cards.length===2?c.cards.map((id,i)=>id?cardMinimum(id,true)/((i===0?c.split:100-c.split)/100)+18:0):[0])));if(required>row.height)return String(required)}
  return ''
}
export function changeTemplate(rows:OrderedRow[],rowId:string,template:Template){
  const result=clone(rows),index=result.findIndex(r=>r.id===rowId),old=result[index],ids=visibleIds([old]),capacity=templates.find(t=>t.id===template)!.slots.reduce((a,b)=>a+b,0)
  if((old.template==='left-big'&&template==='right-big')||(old.template==='right-big'&&template==='left-big')){old.template=template;old.columns.reverse();return result}
  const replacement=makeRow(ids.slice(0,capacity),template==='right-big'?'left-big':template,old.height,old.id)
  if(template==='right-big'){replacement.template=template;replacement.columns.reverse()}
  result.splice(index,1,replacement,...ids.slice(capacity).map(id=>makeRow([id],'single',Math.max(320,widget(id).singleHeight))))
  return result
}
export type ActualColumn={source:number;weight:number;cards:Array<WidgetId|null>;slots:number[];split:number;compact:boolean}
export type ActualRow={id:string;source:string;height:number;natural?:boolean;chartHeight?:number;columns:ActualColumn[]}
export function renderRows(rows:OrderedRow[],width:number,favoriteCount=4,showEmpty=false,prefs?:WidgetPreferences):ActualRow[]{
  const result:ActualRow[]=[]
  const singleHeight=(id:WidgetId)=>prefs&&['connection','networkInterface'].includes(id)&&prefs[id].mode==='summary'?resolvePresentation(id,prefs[id],{width,height:10000,role:'ordinary',favoriteCount,mobile:true}).minHeight:prefs&&id==='applications'?Math.max(widget(id).singleHeight,resolvePresentation(id,prefs[id],{width,height:10000,role:'ordinary',favoriteCount,mobile:true}).minHeight):prefs&&id==='devices'&&prefs[id].mode==='summary'?resolvePresentation(id,prefs[id],{width,height:10000,role:'ordinary',favoriteCount,mobile:true}).minHeight:widget(id).singleHeight
  for(const row of rows){
    let group:ActualColumn[]=[]
    const colWidth=(c:ActualColumn)=>Math.max(...c.cards.map(id=>id?prefs?placementMinimum(id,prefs[id],c.cards.length===2).minWidth:widget(id).minWidth:260))
    const flush=()=>{
      if(!group.length)return
      if(width<650&&group.length===1){group[0].cards.forEach((id,index)=>result.push({id:row.id+'-'+(id||'empty-'+group[0].source+'-'+index),source:row.id,height:id?singleHeight(id):148,natural:!!row.heightMode,chartHeight:id==='traffic'?300:undefined,columns:[{...group[0],cards:[id],slots:[group[0].slots[index]],compact:false}]}))}
      else result.push({id:row.id+'-'+group[0].source,source:row.id,height:row.height,natural:!!row.heightMode,chartHeight:group.some(c=>c.cards.includes('traffic'))?320:undefined,columns:group})
      group=[]
    }
    row.columns.forEach((column,source)=>{
      const slots=column.cards.map((id,index)=>showEmpty||id?index:-1).filter(index=>index>=0),cards=slots.map(index=>column.cards[index]);if(!cards.length)return
      const next={source,cards,slots,weight:column.weight,split:column.split,compact:column.cards.length>1}
      if(group.length&&group.reduce((sum,c)=>sum+colWidth(c),0)+colWidth(next)+group.length*18>width)flush()
      group.push(next)
    });flush()
  }
  return result
}
