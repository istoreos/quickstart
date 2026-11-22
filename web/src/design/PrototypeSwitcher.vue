<!-- Design entry only; production main.ts never imports this component. -->
<template>
  <nav class="prototype-switcher" aria-label="交互稿方案">
    <button aria-label="上一个方案" @click="cycle(-1)">←</button>
    <span>{{ current }} · {{ variants.find(v => v.key === current)?.name }}</span>
    <button aria-label="下一个方案" @click="cycle(1)">→</button>
  </nav>
</template>
<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
const props=defineProps<{variants:{key:string;name:string}[]}>()
const route=useRoute(),router=useRouter()
const current=computed(()=>String(route.query.variant || 'A'))
function cycle(offset:number){const at=props.variants.findIndex(v=>v.key===current.value);router.replace({query:{...route.query,variant:props.variants[(at+offset+props.variants.length)%props.variants.length].key}})}
function keydown(e:KeyboardEvent){if((e.target as HTMLElement)?.closest('input,textarea,select,[contenteditable]')||e.altKey||e.ctrlKey||e.metaKey)return;if(['ArrowLeft','ArrowRight'].includes(e.key)){e.preventDefault();cycle(e.key==='ArrowRight'?1:-1)}}
onMounted(()=>document.addEventListener('keydown',keydown))
onUnmounted(()=>document.removeEventListener('keydown',keydown))
</script>
<style scoped>
.prototype-switcher{position:fixed;bottom:18px;left:calc(50% + var(--design-system-reserve,0px)/2);transform:translateX(-50%);z-index:1250;display:flex;align-items:center;gap:20px;padding:8px 12px;border-radius:30px;background:#25243d;color:#fff;box-shadow:0 6px 25px #1113;white-space:nowrap;font-size:13px}.prototype-switcher button{border:0;background:#ffffff18;color:#fff;border-radius:50%;width:32px;height:32px;cursor:pointer}
</style>
