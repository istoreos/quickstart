// Only the design Vite entry imports this bridge. Production factories stay intact.
export type NativeAction = (kind:string, props:Record<string,any>)=>{Close:()=>void}
let host:NativeAction|undefined
export function registerNativeActions(next:NativeAction){host=next;return ()=>{if(host===next)host=undefined}}
export function dispatchNativeAction(kind:string,props:Record<string,any>={},fallback:(props:any)=>any){if(['action-disk','action-disk-part','action-disk-format','action-disk-mount'].includes(kind)){window.dispatchEvent(new CustomEvent('quickstart-storage-open',{detail:{tab:'disks',part:(props.mount?.path||props.disk?.path||'').replace('/dev/','')}}));return {Close:()=>{}}}return host?host(kind,props):fallback(props)}
