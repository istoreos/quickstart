<template>
  <div class="sw-disk-maps">
    <article v-for="disk in drives" :key="disk.id" class="sw-disk-row" :class="{offline:!disk.connected}">
      <button class="sw-disk-label" @click="$emit('select',disk.parts[0].id)"><span class="sw-drive-icon" aria-hidden="true">▤</span><strong>{{disk.id.toUpperCase()}}</strong><span>{{disk.size}} GiB · {{care.aliases[disk.id]||disk.model}}</span><small :class="disk.connected?'sw-ok':'sw-danger-text'">{{disk.connected?'● 在线 · '+disk.temperature+'°C':'● 未连接'}}</small></button>
      <div class="sw-map-body"><div class="sw-map"><button v-for="part in disk.parts" :key="part.id" :class="['sw-part',part.kind,{selected:selected===part.id,pending:pending(part.id,disk.id)}]" :style="{flexGrow:part.size}" :aria-label="disk.id+' '+part.name+' '+part.size+' GiB'" :aria-pressed="selected===part.id" :title="part.id+' · '+part.name+' · '+part.size+' GiB'" @click="$emit('select',part.id)"><strong>{{part.protected?'▣ ':''}}{{part.name}}</strong><span>{{part.size.toFixed(part.size<1?3:0)}} GiB</span><small>{{part.kind==='free'?'未分配':part.kind==='raid'?'成员 · '+part.owner:part.fs}}</small><i v-if="part.kind==='files'" class="sw-part-used"><i :style="{width:Math.min(100,part.used/part.size*100)+'%'}"></i></i><em v-if="pending(part.id,disk.id)">待执行</em></button></div><footer><span>{{disk.serial}}</span><span v-if="disk.parts.some(p=>p.protected)">系统分区受保护；可单独使用未分配范围</span><span v-else>点击分区或未分配空间查看可用操作</span></footer></div>
    </article>
    <p class="sw-map-note">容量为 GiB 示例。极小分区已放大显示，实际大小以数值为准；颜色区分用途，未分配空间不等于文件系统内的空闲容量。</p>
  </div>
</template>
<script setup lang="ts">
import {care} from './storageCarePrototype'
import {workbench,type Drive} from './storageWorkbenchPrototype'
defineProps<{drives:Drive[];selected:string}>()
defineEmits<{(e:'select',id:string):void}>()
const pending=(id:string,disk:string)=>workbench.changes.some(x=>x.target===id||x.target.split('+').includes(disk))
</script>
