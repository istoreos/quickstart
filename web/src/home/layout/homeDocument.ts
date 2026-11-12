import {externalId} from '../cards/contract'
import {widgetDefinitions,type WidgetId} from './rowLayout'
import {clone,naturalRows,type OrderedRow} from './orderedLayout'
import {validate,type LayoutResult} from './slotLayout'
import {sizingIssue} from './rowSizing'
import {defaultPreferences,preferenceIssue,presentationHeight,definition,resolvePresentation,type WidgetPreferences} from './widgetPresentation'
export type HomeDocument={schemaVersion:4;revision:number;rows:OrderedRow[];widgetPreferences:WidgetPreferences}
export type LayoutContext={width:number;favoriteCount:number;favoritesKey:string}
export type DisplayPlan={status:'ready'|'needs-confirmation'|'blocked';issue:string;baseRevision:number;baseFingerprint:string;contextKey:string;next:HomeDocument;adjustments:Array<{row:string;from:number;to:number}>;hidden:WidgetId[];changed:boolean}
export const documentOf=(rows:OrderedRow[],prefs=defaultPreferences(),revision=0):HomeDocument=>({schemaVersion:4,revision,rows:naturalRows(rows),widgetPreferences:Object.fromEntries(Object.entries(prefs).map(([id,p])=>[id,{...clone(p),ruleVersion:p.ruleVersion??2}])) as WidgetPreferences})
export function migrateDocument(input:unknown):{document:HomeDocument|null;issues:string[];original:unknown}{
 const original=input,issues:string[]=[]
 try{
  const raw:any=Array.isArray(input)?{rows:input}:input
  if(!raw||raw.schemaVersion!==undefined&&![1,2,3,4].includes(raw.schemaVersion))throw Error('schema')
  if(!Array.isArray(raw.rows)||raw.revision!==undefined&&(!Number.isSafeInteger(raw.revision)||raw.revision<0))throw Error('document')
  const ids=[...widgetDefinitions.map(w=>w.id),...Object.keys(raw.widgetPreferences||{}).filter(id=>raw.schemaVersion===4&&externalId.test(id))] as WidgetId[],prefs=defaultPreferences(raw.schemaVersion>=3?2:raw.schemaVersion===2?1:0)
  if(raw.widgetPreferences){if(typeof raw.widgetPreferences!=='object'||Array.isArray(raw.widgetPreferences))throw Error('preference');for(const [key,value]of Object.entries(raw.widgetPreferences)){if(!ids.includes(key as WidgetId)||preferenceIssue(key as WidgetId,value as any)||Object.keys(value as object).some(k=>!['mode','iconRows','ruleVersion'].includes(k)))throw Error('preference');prefs[key as WidgetId]={...clone(value as any),ruleVersion:raw.schemaVersion>=3?((value as any).ruleVersion??2):raw.schemaVersion===2?1:0}}}
  const rowIds=new Set<string>();for(const r of raw.rows){if(!r||typeof r.id!=='string'||rowIds.has(r.id)||!Number.isFinite(r.height)||r.height<1||r.heightMode!==undefined&&!['content','chart'].includes(r.heightMode)||!Array.isArray(r.columns))throw Error('structure');rowIds.add(r.id);for(const c of r.columns){if(!c||!Array.isArray(c.cards)||!Number.isFinite(c.weight)||!Number.isFinite(c.split)||c.cards.some((id:unknown)=>id!==null&&!ids.includes(id as WidgetId)))throw Error('identity')}}
  for(const row of raw.rows){const issue=sizingIssue(row);if(issue)throw Error(issue)}
  const structural=validate(raw.rows);if(structural.issue&&!structural.issue.startsWith('height:'))throw Error(structural.issue)
  return {document:documentOf(raw.rows,prefs,raw.revision||0),issues,original}
 }catch(e){issues.push((e as Error).message);return {document:null,issues,original}}
}
export const fingerprint=(doc:HomeDocument)=>JSON.stringify([doc.rows,doc.widgetPreferences])
export function validateDisplayRows(rows:OrderedRow[],prefs:WidgetPreferences,context:LayoutContext){
 const structural=validate(rows);if(structural.issue&&!structural.issue.startsWith('height:'))return structural
 for(const w of widgetDefinitions)if(preferenceIssue(w.id,prefs[w.id]))return {issue:'preference'}
 for(const row of rows){const minimum=presentationHeight(row,prefs,context.width,context.favoriteCount);if(minimum.issue)return {issue:minimum.issue,row:row.id};if(!row.heightMode&&minimum.height>row.height)return {issue:'height:'+minimum.height,row:row.id,requiredHeight:minimum.height}}
 return {issue:''}
}
export function planDocument(before:HomeDocument,result:LayoutResult,context:LayoutContext,prefs=before.widgetPreferences,allowAdjustment=true):DisplayPlan{
 const next=documentOf(result.candidateRows||result.rows,prefs,before.revision),adjustments:DisplayPlan['adjustments']=[]
 let issue=result.issue&&!result.issue.startsWith('height:')?result.issue:''
 // A legacy minimum may reject a legitimate new summary. Structural validation
 // is retained, but actual display minima come only from the shared contract.
 const structural=validate(next.rows);if(structural.issue&&!structural.issue.startsWith('height:'))issue=structural.issue
 for(const w of widgetDefinitions)if(preferenceIssue(w.id,prefs[w.id]))issue='preference'
 for(const row of next.rows){const minimum=presentationHeight(row,prefs,context.width,context.favoriteCount);if(minimum.issue)issue=minimum.issue;if(!row.heightMode&&minimum.height>row.height){if(!allowAdjustment)issue='height:'+minimum.height;else{adjustments.push({row:row.id,from:row.height,to:minimum.height});row.height=minimum.height}}}
 const changed=fingerprint(before)!==fingerprint(next);if(changed)next.revision=before.revision+1
 return {status:issue?'blocked':adjustments.length?'needs-confirmation':'ready',issue,baseRevision:before.revision,baseFingerprint:fingerprint(before),contextKey:JSON.stringify(context),next,adjustments,hidden:result.hidden,changed}
}
export function applyPlan(current:HomeDocument,plan:DisplayPlan,context:LayoutContext):{document:HomeDocument;issue:string}{
 const stale=current.revision!==plan.baseRevision||fingerprint(current)!==plan.baseFingerprint||JSON.stringify(context)!==plan.contextKey
 return stale||plan.status==='blocked'?{document:current,issue:stale?'stale-plan':plan.issue}:{document:clone(plan.next),issue:''}
}
// Alternatives are explicit plans for the same operation, never silent writes.
export function summaryAlternative(before:HomeDocument,result:LayoutResult,context:LayoutContext,id?:WidgetId){
 const prefs=clone(before.widgetPreferences),rows=result.candidateRows||result.rows
 for(const row of rows)for(const c of row.columns)for(const card of c.cards){if(!card)continue;const incompatibleStack=c.cards.length===2&&resolvePresentation(card,prefs[card],{width:1000,height:100000,role:'stacked',favoriteCount:context.favoriteCount,natural:true}).issue==='compact';if(definition(card).forms.includes('summary')&&(card===id||incompatibleStack))prefs[card]={...prefs[card],mode:'summary',ruleVersion:2}}
 return fingerprint(documentOf(before.rows,prefs))===fingerprint(before)?null:planDocument(before,result,context,prefs)
}
