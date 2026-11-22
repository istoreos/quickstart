// In-memory transport for the shipped SMART views. Never calls smartctl or writes router config.
import {requireFeatures} from './storageCapabilitiesPrototype'
import {workbench} from './storageWorkbenchPrototype'
const clone=<T>(value:T):T=>JSON.parse(JSON.stringify(value))
let config:PropsSmartConfig={global:{enable:true,powermode:'standby',tmpDiff:0,tmpMax:0},devices:[],tasks:[]}
const tests=new Set<string>()
const logs=['产品交互示例：尚未运行自检。提交测试不代表检查通过。']
const result=(value:unknown)=>({status:200,data:{success:0,result:clone(value),previewOnly:true}})
export function smartWorkbenchFixture(key:string,method:string,body:Record<string,any>){
 if(!/^\/(?:u\/)?smart\//.test(key))return
 if(method==='POST'&&['/smart/config/','/u/smart/test/'].includes(key)&&requireFeatures(['smart']))return {status:200,data:{error:requireFeatures(['smart']),previewOnly:true}}
 if(key==='/u/smart/list/'&&method==='GET')return result({disks:workbench.drives.map(d=>({name:d.id,path:'/dev/'+d.id,model:d.model,serial:d.serial,sizeStr:d.size+' GiB',temp:d.connected?d.temperature+'°C':'—',status:d.connected?'在线':'未连接',health:'未知 · 示例未采样'}))})
 if(key==='/smart/config/'&&method==='GET')return result(config)
 if(key==='/smart/config/'&&method==='POST'){config=clone(body as PropsSmartConfig);logs.push('示例配置已保存，未写入 smartd.conf。');return result(config)}
 if(key==='/smart/log/'&&method==='GET')return result({result:logs.join('\n')})
 if(key==='/u/smart/test/'&&method==='POST'){tests.add(body.devicePath);logs.push(body.devicePath+'：已提交 '+body.type+' 测试（示例），结果待读取。');return result({result:'测试已提交（示例）。这不是自检通过结果。'})}
 if(key==='/smart/test/result/'&&method==='POST')return result({result:body.devicePath+'\n'+(tests.has(body.devicePath)?'示例：测试已提交，尚无完成结果。':'示例：暂无近期自检记录。')})
 if(key==='/smart/attribute/result/'&&method==='POST')return result({result:body.devicePath+'\n示例属性视图；未读取真实设备。\nTemperature: '+(workbench.drives.find(d=>'/dev/'+d.id===body.devicePath)?.temperature??'未知')+'°C'})
 if(key==='/smart/extend/result/'&&method==='POST')return result({result:body.devicePath+'\n产品原型：原扩展信息接口已保留，实际指标需设备支持。'})
 return {status:403,data:{error:'未定义的 SMART 示例请求',previewOnly:true}}
}
