import {ref} from 'vue'
import {recommended as baseRecommended,makeRow,type OrderedRow} from '../home/layout/orderedLayout'
import {fastnetWidgetId} from './fastnetModel'
import {widgetDefinitions,type WidgetId} from '../home/layout/rowLayout'
export * from '../home/layout/orderedLayout'
export function recommended():OrderedRow[]{
 const rows=baseRecommended()
 if(!widgetDefinitions.some(w=>w.id===fastnetWidgetId))return rows
 rows[0]=makeRow(['traffic','connection'],'pair',320,'network')
 const internet=makeRow([fastnetWidgetId,'networkInterface'],'pair',260,'internet')
 internet.columns.forEach(c=>c.weight=50)
 rows.splice(1,0,internet)
 return rows
}
// Keep named design comparisons stable when the recommended layout changes.
const comparisonRows=():OrderedRow[]=>[
  makeRow(['traffic','connection'],'pair',320,'network'),makeRow(['applications'],'single',176,'applications'),
  {...makeRow(['devices','system'],'pair',340,'devices'),columns:[{weight:65,cards:['devices'],split:50},{weight:35,cards:['system'],split:50}]},
  makeRow(['storage','sharing'],'pair',330,'storage'),
]
function sampleLayout(){
 const rows=comparisonRows(),sample=new URLSearchParams(location.search).get('preset')
 if(sample==='stacked'||sample==='mirrored'){rows[0]=makeRow(['traffic','connection','system'],'left-big',320,'network');rows[2].columns[1].cards=[null];if(sample==='mirrored'){rows[0].template='right-big';rows[0].columns.reverse()}}
 else if(sample?.startsWith('network-stack')){rows[0]=makeRow(['traffic',...(sample==='network-stack-reverse'?['networkInterface','connection'] as WidgetId[]:['connection','networkInterface'] as WidgetId[])],'left-big',320,'network');if(sample==='network-stack-mirror'){rows[0].template='right-big';rows[0].columns.reverse()}}
 else if(sample==='services')rows.push(makeRow(['docker','downloads','remote'],'triple',280,'services'))
 else if(sample==='half'){rows[1]=makeRow(['applications','system'],'pair',300,'applications');rows[1].columns.forEach(c=>c.weight=50);rows[2]=makeRow(['devices'],'single',340,'devices')}
 else if(sample==='apps-devices'){rows[1]=makeRow(['applications','devices'],'pair',340,'applications');rows[1].columns.forEach(c=>c.weight=50);rows[2]=makeRow(['system'],'single',280,'system')}
 else if(sample==='network-business')rows.splice(1,0,makeRow(['networkInterface'],'single',260,'interfaces'))
 else return recommended()
 return rows
}
export const orderedSession=ref<OrderedRow[]>(sampleLayout())
