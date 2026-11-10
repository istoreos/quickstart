// Explicit preview fixtures. This file never ships in FastNet or Quickstart.
(() => {
 const fixture=parent.__quickstartCardFixture;
 const api=(url,options)=>fixture.fullAPI(new URL(url,location.href).pathname+new URL(url,location.href).search,options?.body?JSON.parse(options.body):undefined);
 window.fetch=async(input,options={})=>{
  const url=new URL(input,location.href),pathname=url.pathname.replace(/^\/apps\/fastnet/,'');
  if(pathname==='/api/version')return new Response(JSON.stringify({version:document.documentElement.dataset.previewVersion}),{headers:{'Content-Type':'application/json'}});
  if(pathname==='/ping')return new Response('{"ok":true}',{headers:{'Content-Type':'application/json'}});
  if(pathname==='/download'){await new Promise(resolve=>setTimeout(resolve,20));return new Response(new Uint8Array(256*1024))}
  if(pathname==='/upload'){options.body?.cancel?.().catch(()=>{});await new Promise(resolve=>setTimeout(resolve,20));return new Response('{}',{headers:{'x-bytes-received':'262144','x-upload-duration-ms':'20'}})}
  try{return new Response(JSON.stringify(await api(input,options)),{headers:{'Content-Type':'application/json'}})}catch(error){return new Response(JSON.stringify({error:error.message}),{status:503,headers:{'Content-Type':'application/json'}})}
 };
 window.EventSource=class extends EventTarget{
  constructor(url){super();this.closed=false;setTimeout(async()=>{try{this.dispatchEvent(new MessageEvent('stage',{data:JSON.stringify({stage:'download',status:'starting'})}));const data=await api(url);if(!this.closed)this.dispatchEvent(new MessageEvent('summary',{data:JSON.stringify(data)}))}catch(error){if(!this.closed)this.onerror?.(error)}},20)}
  close(){this.closed=true}
 };
})();
