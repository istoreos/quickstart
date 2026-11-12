import type {OrderedRow} from './layout/orderedLayout'
import type {WidgetPreferences} from './layout/widgetPresentation'
import {externalId} from './cards/contract'
import {widgetDefinitions} from './layout/rowLayout'
import {preferenceIssue,resolvePresentation} from './layout/widgetPresentation'
import {validate} from './layout/slotLayout'

export type LayoutConfig={schemaVersion:4;rows:OrderedRow[];widgetPreferences:WidgetPreferences}
export type CustomLink={id:string;name:string;url:string;glyph:string}
export type LauncherConfig={schemaVersion:1;favorites:string[];links:CustomLink[]}
export type ConfigDomain='layout'|'launcher'
export type ConfigValue=LayoutConfig|LauncherConfig
export const MAX_CONFIG_BYTES=128*1024
export class HomeConfigError extends Error{
  constructor(public code:'invalid'|'future'|'conflict'|'forbidden'|'unsupported'|'unavailable'|'uncertain'|'readonly',message=code){super(message);this.name='HomeConfigError'}
}
const fail=():never=>{throw new HomeConfigError('invalid')}
const object=(value:any,keys:string[],required=keys)=>{if(!value||typeof value!=='object'||Array.isArray(value)||Object.keys(value).some(k=>!keys.includes(k))||required.some(k=>!Object.prototype.hasOwnProperty.call(value,k)))fail()}
const finite=(v:any,min:number,max:number)=>typeof v==='number'&&Number.isFinite(v)&&v>=min&&v<=max
const safeId=/^[A-Za-z0-9][A-Za-z0-9_.+-]{0,127}$/
const linkId=/^user-link:[A-Za-z0-9_-]{1,64}$/
const linkIcons=['globe','folder','layers','network','shield','download','terminal','grid']

// Shared fixtures exercise this and the backend validator. The backend remains
// authoritative, including rejection of duplicate JSON keys before decoding.
export function configValue(domain:ConfigDomain,input:unknown):ConfigValue{
 const raw:any=input
 if(!raw||typeof raw!=='object'||!Number.isInteger(raw.schemaVersion))fail()
 if(raw.schemaVersion>(domain==='layout'?4:1))throw new HomeConfigError('future')
 if(new TextEncoder().encode(JSON.stringify(raw)).length>MAX_CONFIG_BYTES)fail()
 if(domain==='layout'){
  object(raw,['schemaVersion','rows','widgetPreferences'])
  if(![3,4].includes(raw.schemaVersion)||!Array.isArray(raw.rows)||raw.rows.length>32)fail()
  const ids=widgetDefinitions.map(w=>w.id)
  const external=Object.keys(raw.widgetPreferences||{}).filter(id=>externalId.test(id))
  if(external.length>32||raw.schemaVersion===3&&external.length)fail()
  object(raw.widgetPreferences,[...ids,...external])
  for(const id of [...ids,...external] as import('./layout/rowLayout').WidgetId[]){const p=raw.widgetPreferences[id];object(p,['mode','iconRows','ruleVersion']);if(preferenceIssue(id,p)||!Number.isInteger(p.iconRows)||!Number.isInteger(p.ruleVersion))fail()}
  const seen=new Set<string>()
  for(const row of raw.rows){
   object(row,['id','template','height','heightMode','minHeight','columns'],['id','template','height','heightMode','columns'])
   if(typeof row.id!=='string'||!safeId.test(row.id)||seen.has(row.id)||!finite(row.height,1,1600)||!['content','chart'].includes(row.heightMode)||row.minHeight!==undefined&&!finite(row.minHeight,1,1600)||!Array.isArray(row.columns)||row.columns.length<1||row.columns.length>3)fail()
   seen.add(row.id)
   for(const column of row.columns){
    object(column,['weight','cards','split','upperShare'],['weight','cards','split'])
    if(!finite(column.weight,.01,100)||!finite(column.split,.01,99.99)||column.upperShare!==undefined&&!finite(column.upperShare,.01,99.99)||!Array.isArray(column.cards)||column.cards.length<1||column.cards.length>2)fail()
    for(const id of column.cards){
     if(id===null)continue
     if(!ids.includes(id)&&!(raw.schemaVersion===4&&external.includes(id)))fail()
     if(column.cards.length===2&&resolvePresentation(id,raw.widgetPreferences[id],{width:1000,height:100000,role:'stacked',favoriteCount:8,natural:true}).issue)fail()
    }
   }
  }
  if(validate(raw.rows).issue)fail()
 }else{
  object(raw,['schemaVersion','favorites','links'])
  if(raw.schemaVersion!==1||!Array.isArray(raw.favorites)||raw.favorites.length>24||!Array.isArray(raw.links)||raw.links.length>64)fail()
  const seen=new Set<string>()
  for(const link of raw.links){
   object(link,['id','name','url','glyph'])
   if(typeof link.id!=='string'||!linkId.test(link.id)||seen.has(link.id)||typeof link.name!=='string'||link.name!==link.name.trim()||!link.name||Array.from(link.name).length>64||/[\u0000-\u001f\u007f-\u009f]/.test(link.name)||typeof link.url!=='string'||!/^https?:\/\//.test(link.url)||new TextEncoder().encode(link.url).length>2048||/[\u0000-\u0020\u007f\\]/.test(link.url)||!linkIcons.includes(link.glyph))fail()
   try{const url=new URL(link.url);if(!['http:','https:'].includes(url.protocol)||!url.hostname||url.username||url.password)fail()}catch{fail()}
   seen.add(link.id)
  }
  const pins=new Set<string>()
  for(const id of raw.favorites){if(typeof id!=='string'||pins.has(id)||!(safeId.test(id)||linkId.test(id)&&seen.has(id)))fail();pins.add(id)}
 }
 const copy=JSON.parse(JSON.stringify(raw))
 if(domain==='layout')copy.schemaVersion=4
 return copy
}
