// One owned Node process, no proxy to any router, strict port ownership.
const path = require('node:path');
const root = path.resolve(__dirname, '..');
const fs=require('node:fs'),{execFileSync}=require('node:child_process');
const { build, preview } = require(path.join(root, 'web/node_modules/vite'));
(async () => {
  const configFile=path.join(root,'web/vite.design.config.ts');
  await build({configFile});
  const fastnet=process.env.FASTNET_SOURCE_DIR||'/projects/workspace-linkease-ubuntu/linkease-github/fastnet';
  const result=execFileSync('bash',[path.join(fastnet,'scripts/build-quickstart-card.sh')],{encoding:'utf8'});
  const source=result.match(/Card output: (.+)/)[1].trim(),entry=fs.readdirSync(source).find(name=>/^card\.[a-f0-9]+\.js$/.test(name));
  const target=path.join(process.env.BUILD_CACHE_WORK_DIR,'vue-preview/luci-static/fastnet/quickstart');fs.mkdirSync(target,{recursive:true});for(const asset of fs.readdirSync(source).filter(name=>name.endsWith('.js')))fs.copyFileSync(path.join(source,asset),path.join(target,asset));
  const fixture=JSON.parse(fs.readFileSync(path.join(root,'docs/contracts/fastnet-card.json'),'utf8')).manifest;fixture.entry='/luci-static/fastnet/quickstart/'+entry;
  fs.writeFileSync(path.join(target,'preview-manifest.json'),JSON.stringify(fixture));
  // Build the real full FastNet frontend for iframe interaction. The bridge is
  // preview-only: every WAN request goes to the parent's simulated card API.
  const fullOut=path.join(process.env.BUILD_CACHE_WORK_DIR,'vue-preview/apps/fastnet/webui');
  execFileSync('/var/cache/build-cache/run.sh',['--','npm','run','build','--','--outDir',fullOut],{cwd:path.join(fastnet,'cmd/fastnet/webui-src'),stdio:'pipe'});
  const version=fs.readFileSync(path.join(fastnet,'cmd/fastnet/version.go'),'utf8').match(/const VERSION = "([^"]+)"/)[1];
  const html=path.join(fullOut,'index.html');
  fs.writeFileSync(html,fs.readFileSync(html,'utf8').replace('<head>',`<head><script>
  const originalFetch=window.fetch.bind(window);
  window.fetch=(input,options)=>{const url=new URL(input,location.href),match=url.pathname.match(/\\/api\\/card\\/(.*)/);if(match)return parent.fetch('/cgi-bin/luci/admin/services/fastnet/card/'+match[1]+url.search,options);return Promise.resolve(new Response(JSON.stringify(url.pathname.includes('/api/version')?{version:${JSON.stringify(version)}}:url.pathname.includes('test-usage')?{used:0,limit:5,remaining:5}:{}),{headers:{'Content-Type':'application/json'}}))};
  </script>`));
  const server=await preview({configFile});
  console.log('Quickstart Vue design: http://0.0.0.0:5180/');
  console.log(`Design artifacts: ${server.config.build.outDir}`);
  for (const signal of ['SIGTERM', 'SIGINT']) process.on(signal, () => { server.httpServer.close(()=>process.exit(0)); });
})().catch(error => { console.error(error); process.exit(1); });
