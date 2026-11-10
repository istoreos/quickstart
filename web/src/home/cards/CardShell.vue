<template>
 <section ref="root" class="qs-card-shell" :aria-busy="state==='loading'" :data-card-state="state">
  <component v-if="component" :is="component" :context="owner.context"/>
  <div v-else class="qs-card-placeholder" :role="state==='error'?'status':undefined"><h2>{{title}}</h2><p>{{state==='error'?'卡片暂时无法加载':'正在准备卡片…'}}</p><button v-if="state==='error'" @click="retry">重试</button></div>
  <NetworkWidgetDialog v-if="owner.details.value" :title="title" :wide="owner.details.value.wide" @close="owner.closeDetails()"><component :is="owner.details.value.component" v-bind="owner.details.value.props" @close="owner.closeDetails()"/></NetworkWidgetDialog>
 </section>
</template>
<script setup lang="ts">
import {externalWidget} from './catalog'
import {computed,markRaw,onErrorCaptured,onMounted,onUnmounted,ref,shallowRef,watch,type Component} from 'vue'
import NetworkWidgetDialog from '../NetworkWidgetDialog.vue'
import WidgetPrimaryAction from '../WidgetPrimaryAction.vue'
import WidgetMenu from '../WidgetMenu.vue'
import {createCardContext} from './context'
import {loadCard} from './loader'
import type {CardForm,CardManifest} from './contract'
const props=defineProps<{manifest:CardManifest;form:CardForm;rediscover?:()=>Promise<CardManifest|undefined>}>()
const title=computed(()=>window.vue_lang==='en'?externalWidget(props.manifest.id as `plugin.${string}`).en:props.manifest.title)
const root=ref<HTMLElement|null>(null),component=shallowRef<Component|null>(null),state=ref('waiting'),owner=createCardContext(props.manifest.apiBase,window.vue_lang||'zh-cn',props.form,{PrimaryAction:markRaw(WidgetPrimaryAction),Menu:markRaw(WidgetMenu)})
watch(()=>props.form,form=>owner.setForm(form))
let alive=true,sequence=0,rediscovered=false,observer:IntersectionObserver|undefined
async function mount(manifest=props.manifest){
 const token=++sequence;state.value='loading'
 try{const definition=await loadCard(manifest);if(alive&&sequence===token){component.value=markRaw(definition);state.value='ready'}}
 catch{if(!alive||sequence!==token)return;if(!rediscovered&&props.rediscover){rediscovered=true;try{const fresh=await props.rediscover();if(alive&&sequence===token&&fresh&&fresh.entry!==manifest.entry){await mount(fresh);return}}catch{}}
  if(alive&&sequence===token)state.value='error'
 }
}
function retry(){component.value=null;void mount()}
onErrorCaptured(()=>{component.value=null;state.value='error';return false})
onMounted(()=>{observer=new IntersectionObserver(entries=>{if(entries.some(e=>e.isIntersecting)){observer?.disconnect();void mount()}},{rootMargin:'100px'});if(root.value)observer.observe(root.value)})
onUnmounted(()=>{alive=false;sequence++;observer?.disconnect();owner.dispose()})
</script>
<style src="./card.css"></style>
