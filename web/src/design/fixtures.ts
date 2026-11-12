import {cardFixture} from './cardFixture'
// Synthetic fixtures, used exclusively by the independently loaded design entry.
import { reactive } from 'vue'
import { inventory } from './devices'
import { deviceFixture } from './deviceFixtures'
import { networkFixture } from './networkFixtures'
import { businessFixture } from './businessFixtures'
import { launcherFixture } from './launcherFixtures'
const params=new URLSearchParams(location.search)
export const demo = reactive({ scenario: params.get('scenario') || 'normal', authority: params.get('authority') || 'external', recovered: false, layout: params.get('layout') || 'baseline' })
const demoNetworks = () => params.get('lans') === '2' ? ['lan1', 'lan2'] : [params.get('lans') === 'custom' ? 'home' : 'lan']
const scopedInventory = (network: string | null) => {
  const ids = demoNetworks()
  return (network === 'all' ? ids : [network || ids[0]]).filter(id => ids.includes(id)).flatMap((networkId) => inventory().map(device => ({ ...device, networkId, displayName: ids.length > 1 ? `${networkId} · ${device.displayName}` : device.displayName, addresses: { ...device.addresses, current: device.addresses.current.map(address => ({ ...address, address: address.family === 4 ? address.address.replace('192.168.30.', `192.168.${networkId === 'lan2' ? 32 : 31}.`) : address.address })) } })))
}
export const requests: Array<{ path: string; method: string; status: number }> = []
export const fixturePaths = new Set<string>()
let hiddenModules: string[] = []
const at = 1791972000
const base = '/cgi-bin/luci/istore'
const simple: Record<string, unknown> = {
  '/u/network/status/': { defaultInterface: 'wan', internetConnected: true, networkInfo: 'netSuccess', proto: 'static', ipv4addr: '192.168.30.7', ipv4mask: 24, gateway: '192.168.30.1', dnsList: ['192.168.30.1'], dnsProto: 'manual', uptimeStamp: 109000, dnsReady: true },
  '/u/network/statistics/': { slots: 10, items: [210,410,430,140,260,380,430,420,270,360].map((n,i) => ({ startTime: at-60+i*6, endTime: at-54+i*6, downloadSpeed: n*1024, uploadSpeed: (n/20+2)*1024 })) },
  '/network/device/list/': { devices: [1,2,3].map(i=>({ name: `家庭设备 ${i}`, ipv4addr:`192.168.30.${100+i}`,mac:`02:00:00:00:00:0${i}` })) },
  '/network/port/list/': { ports: [0,1,2].map(i=>({ name: `eth${i}`,macAddress:`02:00:00:00:01:0${i}`,linkSpeed:'',linkState:i===0?'UP':'DOWN',rx_packets:'10',tx_packets:'10',interfaceNames:i===0?['WAN','WAN6']:['LAN'],master:'',duplex:'full' })) },
  '/u/system/version/': { model:'QEMU Standard PC (Q35 + ICH9, 2009)',kernelVersion:'6.6.86',firmwareVersion:'iStoreOS 24.10.1 2025011019' },
  '/system/status/': { cpuTemperature:0,cpuUsage:1,memAvailablePercentage:73,memTotal:'4 GiB',memAvailable:'3 GiB',uptime:109000,localtime:'2026-10-14 12:52:59' },
  '/system/check-update/': {needUpdate:false,msg:'preview'},
  '/nas/disk/status/': { disks: [
    {name:'sda',path:'/dev/sda',venderModel:'QEMU QEMU HARDDISK',total:'2.4 GiB',size:'2.4 GiB',used:'1.5 GiB',usage:64,isSystemRoot:true,childrens:[{name:'sda1',path:'/dev/sda1',total:'2.4 GiB',used:'1.5 GiB',usage:64,mountPoint:'/',isSystemRoot:true}]},
    ...[32,16,24].map((n,i)=>({name:`sd${'bcd'[i]}`,path:`/dev/sd${'bcd'[i]}`,total:`${n}.0 GiB`,size:`${n}.0 GiB`,used:['23.3 GiB','13.9 GiB','10.2 GiB'][i],usage:[72,87,42][i],venderModel:'QEMU QEMU HARDDISK',childrens:[{name:`sd${'bcd'[i]}1`,path:`/dev/sd${'bcd'[i]}1`,total:`${n}.0 GiB`,used:['23.3 GiB','13.9 GiB','10.2 GiB'][i],usage:[72,87,42][i],mountPoint:`/mnt/vio${i+2}-1`}] }))
  ]},
  '/raid/list/': {disks:[]},
  '/u/nas/service/status/': {linkease:{enabel:false},sambas:[{shareName:'test',path:'/mnt/vio4-1'}],webdav:{}},
  '/guide/docker/status/': {status:'running',path:'/mnt/vio3-1/docker',errorInfo:''},
  '/guide/download-service/status/': {aria2:{status:'stopped',rpcToken:'',rpcPort:6800,webPath:''},qbittorrent:{status:'stopped',webPath:''},transmission:{status:'stopped',webPath:''}},
  '/u/guide/ddns/': {ipv4Domain:'Stopped',ipv6Domain:'Stopped',ddnstoDomain:'https://demo.example.invalid:443'},
  '/nas/sandbox/': {status:'stopped'},
  '/guide/download/partition/list/':{partitionList:['/mnt/vio4-1']},
  '/guide/docker/partition/list/':{partitionList:['/mnt/vio3-1']},
}

export function installFixtures() {
  const realFetch = window.fetch.bind(window)
  window.fetch = async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const url = new URL(typeof input === 'string' ? input : input instanceof Request ? input.url : String(input), location.href)
    const method = (init?.method || (input instanceof Request ? input.method : 'GET')).toUpperCase()
    // Only same-origin static assets can leave this adapter. No router API fallback.
    if (!url.pathname.startsWith('/cgi-bin/') && url.origin === location.origin) return realFetch(input, init)
    let status = 200, data: unknown
    const key = url.pathname.startsWith(base) ? url.pathname.slice(base.length) : url.pathname
    const cardReply=url.origin===location.origin?cardFixture(key,method,JSON.parse(String(init?.body||'{}'))):undefined
    const networkReply=url.origin===location.origin&&(demo.layout==='ordered'||!params.has('layout'))?networkFixture(key,method,JSON.parse(String(init?.body||'{}')),demo.scenario,demo.recovered):undefined
    if(method==='GET'&&key==='/u/network/status/'&&demo.scenario==='network-loading'&&!demo.recovered)await new Promise(resolve=>setTimeout(resolve,1800))
    if(method==='GET'&&key==='/network/port/list/'&&demo.scenario==='ports-loading'&&!demo.recovered)await new Promise(resolve=>setTimeout(resolve,1800))
    const deviceReply=url.origin===location.origin?deviceFixture(key,method,url,JSON.parse(String(init?.body||'{}'))):undefined
    const businessReply=url.origin===location.origin&&(demo.layout==='ordered'||!params.has('layout'))?businessFixture(key,method,JSON.parse(String(init?.body||'{}')),demo.scenario,demo.recovered):undefined
    const launcherReply=url.origin===location.origin?launcherFixture(key,method,JSON.parse(String(init?.body||'{}')),url.searchParams.has('launcher-retry')?'normal':demo.scenario):undefined
    if (url.origin !== location.origin) { status=403; data={error:'External requests are disabled in design mode'} }
    else if(cardReply){status=cardReply.status;data=cardReply.data}
    else if(launcherReply){status=launcherReply.status;data=launcherReply.data}
    else if(networkReply){status=networkReply.status;data=networkReply.data}
    else if(businessReply&&!(method==='GET'&&!demo.recovered&&demo.scenario==='system-error'&&key==='/system/status/')){status=businessReply.status;data=businessReply.data}
    else if (key === '/app/check/' && method === 'POST') data={result:{status:'installed'}}
    else if (key === '/system/module-settings/' && method === 'GET') data={result:{diableDisplay:hiddenModules}}
    else if (key === '/system/module-settings/' && method === 'POST') { hiddenModules=JSON.parse(String(init?.body || '{}')).diableDisplay || []; data={result:{diableDisplay:hiddenModules},previewOnly:true} }
    else if(deviceReply){status=deviceReply.status;data=deviceReply.data}
    else if (key==='/lanctrl/v2/networks/' && method==='GET') {
      const scenario=params.get('discovery')
      if (['404','401','403','500'].includes(scenario||'')) { status=Number(scenario);data={error:'Synthetic discovery failure'} }
      else data={result:{scopeVersion:1,networks:demoNetworks().map(networkId=>({networkId,displayName:networkId,device:networkId==='home'?'br-home':`br-${networkId}`,state:'up',ipv4Prefixes:[`192.168.${networkId==='lan2'?32:31}.0/24`]}))}}
    }
    else if (key==='/lanctrl/v2/devices/' && method==='GET') {
      if(!demo.recovered && ['error','stale','old','permission'].includes(demo.scenario)){status=demo.scenario==='old'?404:demo.scenario==='permission'?403:503;data={error:demo.scenario==='old'?'API v2 not supported':demo.scenario==='permission'?'Device read access denied':'Inventory read failed'}}
      else data={result:{devices:scopedInventory(url.searchParams.get('network')),health:{state:'ready',reasons:[]}}}
    }
    else if (key==='/lanctrl/v2/network-rules/' && method==='GET') data={result:{rules:[{kind:'access',deviceId:'mac:02:00:00:00:00:06'}]}}
    else if (key==='/lanctrl/v2/device-traffic/' && method==='GET') {const available=!['no-speed','disabled','speed-error','unsupported'].includes(demo.scenario);data={result:{health:{state:available?'ready':'partial'},items:scopedInventory(url.searchParams.get('network')).map((d,i)=>({networkId:d.networkId,deviceId:d.deviceId,uploadSpeed:available?(i===5?0:82000):-1,downloadSpeed:available?(i===5?0:2400000):-1,uploadBytes:available?1000000:-1,downloadBytes:available?10000000:-1,connectionCount:available?4:0,state:available?'ready':'partial',sampledAt:'2026-10-14T04:00:00Z'}))}}}
    else if (key==='/network/device/list/' && method==='GET' && demo.layout!=='baseline') data={result:{devices:!demo.recovered&&['loading','error','old','permission'].includes(demo.scenario)?[]:inventory().filter(d=>d.online).map(d=>({name:d.displayName,ipv4addr:d.addresses.current[0]?.address,mac:d.mac}))}}
    else if(method==='GET'&&((demo.scenario==='network-error'&&['/u/network/status/','/u/network/statistics/','/network/port/list/'].includes(key))||(demo.scenario==='storage-error'&&key==='/nas/disk/status/')||(demo.scenario==='services-error'&&['/u/nas/service/status/','/guide/docker/status/','/guide/download-service/status/','/u/guide/ddns/'].includes(key)))){status=503;data={error:'Synthetic module failure'}}
    else if(method==='GET'&&!demo.recovered&&demo.scenario==='system-error'&&['/system/status/','/u/system/version/'].includes(key)){status=503;data={error:'Synthetic system read failure'}}
    else if (method === 'GET' && key==='/nas/disk/status/'&&demo.scenario==='many-disks'){const original=(simple[key] as {disks:Record<string,unknown>[]}).disks;data={result:{disks:Array.from({length:24},(_,i)=>({...original[i%original.length],name:'disk-'+(i+1),isSystemRoot:i===0}))}}}
    else if (method === 'GET' && key in simple) data={result:simple[key]}
    else { status=method==='GET'?501:403; data={error:`Preview fixture is not defined for ${method} ${key}`,previewOnly:true} }
    fixturePaths.add(`${method} ${key}`)
    requests.push({path:key+url.search,method,status}); if(requests.length>120) requests.shift()
    return new Response(JSON.stringify(data),{status,headers:{'Content-Type':'application/json','X-Quickstart-Design':'fixture'}})
  }
}

// Bounded diagnostic state for automated review; contains synthetic request metadata only.
Object.defineProperty(window,'__quickstartDesign',{value:{demo,requests,fixturePaths},configurable:true})
