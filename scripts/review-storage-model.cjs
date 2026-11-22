// Acceptance of destructive-operation boundaries in the in-memory design model.
// Run via /var/cache/build-cache/run.sh so the temporary bundle stays on SSD.
const assert=require('node:assert/strict'),path=require('node:path'),fs=require('node:fs');
const root=path.resolve(__dirname,'..'),esbuild=require(path.join(root,'web/node_modules/esbuild'));
(async()=>{
 if(!process.env.BUILD_CACHE_WORK_DIR)throw Error('Use /var/cache/build-cache/run.sh -- node scripts/review-storage-model.cjs');
 const output=path.join(process.env.BUILD_CACHE_WORK_DIR,'storage-model-review.cjs');
 await esbuild.build({entryPoints:[path.join(root,'web/src/design/storageWorkbenchPrototype.ts')],bundle:true,platform:'node',format:'cjs',outfile:output,logLevel:'silent'});
 const m=require(output),checks=[];async function check(name,fn){m.resetWorkbench();await fn();checks.push(name);console.log('PASS '+name)}
 await check('同路径换盘使原计划失效，不创建分区',async()=>{assert.equal(m.queueCreate('sdb-free','新文件',8),true);m.workbench.drives[1].serial='OTHER';await m.applyChanges();assert.equal(m.workbench.jobs[0].status,'失败');assert.match(m.workbench.jobs[0].detail,/身份/);assert.equal(m.findPart('sdb-free').part.size,32)});
 await check('执行前掉线或库存过期停止写入',async()=>{for(const state of ['offline','stale']){m.resetWorkbench();m.queueCreate('sdb-free','新文件',8);if(state==='offline')m.workbench.drives[1].connected=false;else m.workbench.readState='stale';await m.applyChanges();assert.equal(m.workbench.jobs[0].status,'失败');assert.equal(m.findPart('sdb-free').part.size,32)}});
 await check('卷、池和物理来源共用冲突锁，不接受父子并发计划',async()=>{const pool=m.workbench.pools[1],v=pool.volumes[0];assert.equal(m.queueGrow(pool,v,28),true);assert.equal(m.queueVolume(pool,'第二卷',4,'Docker 数据卷'),false);assert.equal(m.queueCreate('sdc-free','同盘文件',4),false);assert.equal(m.queueCreate('sdb-free','其他硬盘',4),true);await m.applyChanges();assert.ok(m.workbench.jobs.every(j=>j.status==='完成'));assert.equal(v.size,28)});
 await check('池元数据状态在排队后恶化时拒绝新分配',async()=>{const pool=m.workbench.pools[1];assert.equal(m.queueVolume(pool,'新卷',4,'Docker 数据卷'),true);pool.metadata=95;await m.applyChanges();assert.equal(m.workbench.jobs[0].status,'失败');assert.equal(pool.volumes.length,2)});
 await check('已有 EXT4 挂载卸载保留数据；应用占用阻止卸载',async()=>{const p=m.findPart('sdb1').part;const old=JSON.stringify({size:p.size,used:p.used,fs:p.fs});m.mountPart(p.id);assert.equal(p.mount,'');m.mountPart(p.id);assert.ok(p.mount);assert.equal(JSON.stringify({size:p.size,used:p.used,fs:p.fs}),old);p.consumers=['Docker'];m.mountPart(p.id);assert.ok(p.mount);assert.equal(m.queueDelete(p.id),false)});
 await check('外来结构和不可靠身份无法排队写入',async()=>{m.resetWorkbench('foreign');assert.equal(m.queueDelete('sdd1'),false);assert.equal(m.queueFormat('sdd1'),false);m.resetWorkbench('identity-unknown');assert.equal(m.queueCreate('sdd-free','文件',4),false)});
 await check('后续失败不回滚已完成步骤，也不执行余项',async()=>{let value=0;m.queue({title:'已完成',target:'one',impact:'demo',verify:()=>'',apply:()=>{value=1}});m.queue({title:'故障',target:'two',impact:'demo',verify:()=>'示例失败',apply:()=>{value=2}});m.queue({title:'未执行',target:'three',impact:'demo',verify:()=>'',apply:()=>{value=3}});await m.applyChanges();assert.equal(value,1);assert.deepEqual(m.workbench.jobs.map(j=>j.status),['完成','失败','未执行'])});
 const file=process.env.STORAGE_MODEL_REVIEW_OUTPUT||path.join(root,'docs/product/ui-design/storage-workbench/care-model-review.json');fs.writeFileSync(file,JSON.stringify({status:'passed',scope:'in-memory model only',checks},null,2)+'\n');
})().catch(e=>{console.error(e);process.exit(1)});
