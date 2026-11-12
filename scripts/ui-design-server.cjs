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
  // preview-only: both entries share the parent's in-memory history fixtures.
  const fullOut=path.join(process.env.BUILD_CACHE_WORK_DIR,'vue-preview/apps/fastnet/webui');
  execFileSync('/var/cache/build-cache/run.sh',['--','npm','run','build','--','--outDir',fullOut],{cwd:path.join(fastnet,'cmd/fastnet/webui-src'),stdio:'pipe'});
  const version=fs.readFileSync(path.join(fastnet,'cmd/fastnet/version.go'),'utf8').match(/const VERSION = "([^"]+)"/)[1];
  const html=path.join(fullOut,'index.html');
  fs.copyFileSync(path.join(root,'scripts/ui-design-fastnet-bridge.js'),path.join(fullOut,'preview-bridge.js'));
  fs.writeFileSync(html,fs.readFileSync(html,'utf8').replace('<head>',`<head><script>document.documentElement.dataset.previewVersion=${JSON.stringify(version)}</script><script src="./preview-bridge.js"></script>`));
  const server=await preview({configFile});
  console.log('Quickstart Vue design: http://0.0.0.0:5180/');
  console.log(`Design artifacts: ${server.config.build.outDir}`);
  for (const signal of ['SIGTERM', 'SIGINT']) process.on(signal, () => { server.httpServer.close(()=>process.exit(0)); });
})().catch(error => { console.error(error); process.exit(1); });
