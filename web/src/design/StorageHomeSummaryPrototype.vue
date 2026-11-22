<!-- Used only by the design Vite transform. Shares the workbench's memory model. -->
<template>
 <Card title="硬盘与存储" :home-menu="[{label:'硬盘与分区',href:'/storage?tab=disks'},{label:'RAID 管理',href:'/storage?tab=raid'},{label:'S.M.A.R.T.',href:'/smart'}]">
  <template #icon><span class="sh-icon" aria-hidden="true">▤</span></template>
  <template #settings><router-link to="/storage?tab=disks" data-home-storage-entry>管理硬盘 ›</router-link></template>
  <div class="sh-summary"><div class="sh-counts"><span><strong>{{workbench.drives.length}}</strong>块硬盘</span><span><strong>{{files}}</strong>处普通文件存储</span><span><strong>{{workbench.pools.length}}</strong>个存储池</span></div><router-link v-if="warning" class="sh-warning" to="/storage">有存储状态需要关注，请查看详情 ›</router-link><router-link v-for="disk in workbench.drives.slice(0,3)" :key="disk.id" :to="{path:'/storage',query:{tab:'disks',part:disk.parts[0].id}}" class="sh-drive"><div><strong>{{disk.id.toUpperCase()}} <small>{{disk.model}}</small></strong><span>{{disk.size}} GiB</span></div><div class="sh-bar"><i v-for="part in disk.parts" :key="part.id" :class="part.kind" :style="{flex:part.size}"></i></div><small>{{disk.connected?'在线':'未连接'}} · {{disk.parts.some(p=>p.protected)?'系统分区受保护':disk.parts.some(p=>p.kind==='hosts')?'含 LightHost 主机存储':'普通文件 / 应用存储'}}</small></router-link><router-link class="sh-footer" to="/storage" data-home-storage-all>查看全部硬盘、分区、卷与 RAID ›</router-link></div>
 </Card>
</template>
<script setup lang="ts">
import {computed} from 'vue'
import Card from '../pages/index/components/Card.vue'
import {careAlerts} from './storageCarePrototype'
import {workbench} from './storageWorkbenchPrototype'
const files=computed(()=>workbench.drives.reduce((n,d)=>n+d.parts.filter(p=>p.kind==='files').length,0))
const warning=computed(()=>careAlerts.value.length>0)
</script>
<style scoped>
.sh-icon{font-size:23px;color:var(--tit-color,#45556c)}.sh-summary{padding:15px 18px;font:13px/1.6 system-ui;color:var(--tit-color,#334155)}.sh-counts{display:flex;gap:22px;padding-bottom:14px;border-bottom:1px solid var(--line-color,#e4e8ef)}.sh-counts span{display:flex;align-items:baseline;gap:5px;font-size:11px;color:var(--text-color,#738095)}.sh-counts strong{font-size:22px;color:var(--tit-color,#334155);font-weight:600}.sh-drive{display:block;text-decoration:none;color:inherit;margin-top:15px}.sh-drive>div:first-child{display:flex;justify-content:space-between;gap:10px}.sh-drive strong{font-size:12px}.sh-drive small{font-weight:400;color:var(--text-color,#738095);font-size:10px}.sh-drive strong small{margin-left:8px}.sh-drive span{font-size:11px;color:var(--text-color,#738095)}.sh-bar{display:flex;gap:2px;height:7px;margin:7px 0 4px;border-radius:3px;overflow:hidden;background:#e3e8ef}.sh-bar i{min-width:3px;background:#adb6c5}.sh-bar .files{background:#6eaf8f}.sh-bar .hosts{background:#8088db}.sh-bar .apps{background:#7ca3d8}.sh-bar .raid{background:#b08aca}.sh-bar .free{background:#e3e8ef}.sh-footer{display:block;text-align:center;padding-top:14px;margin-top:16px;border-top:1px solid var(--line-color,#e4e8ef);color:#6473cf;text-decoration:none;font-size:12px}.sh-warning{display:block;padding:10px;margin-top:12px;background:#fff4de;color:#93632d;font-size:12px}a[data-home-storage-entry]{font-size:12px;color:#6473cf;text-decoration:none}@media(max-width:430px){.sh-counts{gap:12px}.sh-counts strong{font-size:19px}.sh-counts span{font-size:10px}}
</style>
