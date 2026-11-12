// Product R02: display capabilities are independent of installed router services.
import {widgetDefinitions,widget,type WidgetId} from './rowLayout'
import {definition,resolvePresentation} from './widgetPresentation'
export type DisplaySpec='full'|'standard'|'compact'
const summaries:WidgetId[]=['connection','networkInterface','system','sharing','docker','downloads','remote']
export const moduleCapabilities=widgetDefinitions.map(w=>({
  id:w.id,minWidth:w.minWidth,minHeight:w.minHeight,
  mobileMinWidth:240,stacked:summaries.includes(w.id),compactHeight:summaries.includes(w.id)?148:0,
  specs:(summaries.includes(w.id)||w.id==='applications'?['full','standard','compact']:['full','standard']) as DisplaySpec[],
  summary:w.id==='applications'?'launch-grid':summaries.includes(w.id)?'summary':'standard',
  forms:definition(w.id).forms,
}))
export const capability=(id:WidgetId)=>id.startsWith('plugin.')?{...widget(id),mobileMinWidth:240,stacked:true,compactHeight:148,specs:['full','standard','compact'] as DisplaySpec[],summary:'summary',forms:definition(id).forms}:moduleCapabilities.find(c=>c.id===id)!
export function spaceIssue(id:WidgetId,width:number,height:number,stacked=false,mobile=false):string{
  return resolvePresentation(id,{mode:'auto',iconRows:0},{width,height,role:stacked?'stacked':'ordinary',favoriteCount:4,mobile}).issue
}
