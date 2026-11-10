<template>
 <Teleport to="body"><div class="network-dialog-shade" @click.self="$emit('close')"><section ref="element" class="network-dialog home-semantic-colors" :class="{'network-dialog-wide':wide}" role="dialog" aria-modal="true" :aria-label="title" tabindex="-1" @keydown="key">
  <header><div><small v-if="!wide">QUICKSTART</small><h2>{{title}}</h2></div><div class="network-dialog-actions"><a v-if="headerAction" data-network-dialog-open :href="headerAction.href" target="_blank" rel="noopener">{{headerAction.label}} ↗</a><button data-network-dialog-close @click="$emit('close')" :aria-label="english?'Close details':'关闭详情'">×</button></div></header>
  <div class="network-dialog-body"><slot/></div>
 </section></div></Teleport>
</template>
<script setup lang="ts">
import {nextTick,onMounted,onUnmounted,ref} from 'vue'
defineProps<{title:string;wide?:boolean;headerAction?:{href:string;label:string}}>();const emit=defineEmits<{(e:'close'):void}>(),english=window.vue_lang==='en',element=ref<HTMLElement|null>(null)
let opener:HTMLElement|null=null,overflow=''
const focusable=()=>Array.from(element.value?.querySelectorAll<HTMLElement>('button:not(:disabled),a[href],input:not(:disabled),select:not(:disabled),iframe,[tabindex="0"]')||[]).filter(e=>e.getClientRects().length)
function key(event:KeyboardEvent){if(event.key==='Escape'){event.preventDefault();event.stopPropagation();emit('close')}else if(event.key==='Tab'){const nodes=focusable(),first=nodes[0],last=nodes[nodes.length-1];if(!first){event.preventDefault();element.value?.focus()}else if(event.shiftKey&&(document.activeElement===first||document.activeElement===element.value)){event.preventDefault();last.focus()}else if(!event.shiftKey&&(document.activeElement===last||document.activeElement===element.value)){event.preventDefault();first.focus()}}}
onMounted(async()=>{opener=document.activeElement as HTMLElement;overflow=document.body.style.overflow;document.body.style.overflow='hidden';await nextTick();element.value?.focus()})
onUnmounted(()=>{document.body.style.overflow=overflow;if(opener?.isConnected)opener.focus({preventScroll:true})})
</script>
<style scoped>
.network-dialog-shade{position:fixed;inset:0;z-index:4000;background:#07121e66;backdrop-filter:blur(5px);display:flex;align-items:center;justify-content:center;padding:16px;box-sizing:border-box;color:var(--c-text,var(--tit-color))}.network-dialog{width:560px;min-width:0;max-width:100%;max-height:calc(100vh - 32px);background:var(--c-panel,var(--card-bg-color));border:1px solid var(--c-line,var(--border-color));border-radius:18px;box-shadow:0 24px 80px #0003;display:flex;flex-direction:column;outline:none;box-sizing:border-box}.network-dialog>header{display:flex;align-items:center;justify-content:space-between;padding:22px 24px 18px;gap:12px;border-bottom:1px solid var(--c-line,var(--border-color))}.network-dialog h2{font-size:17px;background:none;margin:4px 0 0;padding:0;color:inherit;line-height:1.4}.network-dialog small{font-size:9px;letter-spacing:2px;color:var(--c-muted,var(--item-label_key-span-color))}.network-dialog-actions>button{border:0;background:var(--c-soft,#8881);color:inherit;border-radius:9px;width:32px;height:32px;font-size:23px;cursor:pointer;flex:none}.network-dialog-body{padding:20px 24px;overflow:auto;min-height:0;font-size:13px}.network-dialog :deep(button:focus-visible),.network-dialog :deep(a:focus-visible){outline:2px solid var(--c-accent,#2583bb);outline-offset:3px}@media(max-width:450px){.network-dialog>header{padding:18px}.network-dialog-body{padding:16px 18px}}
.network-dialog-actions{display:flex;align-items:center;gap:12px;flex:none}.network-dialog-actions a{font-size:12px;color:var(--c-accent,#2583bb);padding:6px 0}

.network-dialog-wide{width:1180px;height:calc(100dvh - 48px);max-height:900px}.network-dialog-wide>header{padding:10px 20px}.network-dialog-wide h2{margin:0}.network-dialog-wide .network-dialog-body{padding:0;flex:1;display:flex;overflow:hidden}.network-dialog-wide .network-dialog-actions>button{width:40px;height:40px}@media(max-width:600px){.network-dialog-wide{height:calc(100dvh - 32px);border-radius:14px}}
</style>
