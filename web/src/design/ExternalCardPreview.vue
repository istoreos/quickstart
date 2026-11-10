<template>
 <main class="ordered-home home-semantic-colors" style="padding:24px;min-height:100vh;background:var(--c-bg);color:var(--c-text)">
  <ThemeSelector/><button @click="visible=!visible" data-toggle-external>显示/隐藏</button><button @click="form=form==='standard'?'summary':'standard'" data-external-form>切换形态</button>
  <div v-if="far" style="height:2400px">滚动后加载</div>
  <div style="max-width:560px;min-height:148px" :data-rediscoveries="rediscoveries"><CardShell v-if="visible" :manifest="manifest" :form="form" :rediscover="rediscover"/></div>
 </main>
</template>
<script setup lang="ts">
import {ref} from 'vue'
import ThemeSelector from '../home/ThemeSelector.vue'
import CardShell from '../home/cards/CardShell.vue'
import {cardManifest,type CardForm} from '../home/cards/contract'
import fixtures from '../../../docs/contracts/fastnet-card.json'
const query=new URLSearchParams(location.search),visible=ref(query.get('hidden')!=='1'),far=query.get('far')==='1',form=ref<CardForm>(query.get('form')==='summary'?'summary':'standard')
const manifest=cardManifest({...((window as any).__quickstartPreviewCardManifest||fixtures.manifest),entry:query.get('entry')||(window as any).__quickstartPreviewCardManifest?.entry||fixtures.manifest.entry})
const rediscoveries=ref(0)
async function rediscover(){rediscoveries.value++;return cardManifest({...manifest,entry:query.get('replacement')||manifest.entry})}
</script>
<style src="../home/homeSemanticColors.css"></style>
