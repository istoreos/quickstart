<template>
  <section class="fastnet-card" :class="{summary}" data-fastnet-card>
    <header><h2><StudioGlyph name="activity"/>{{t('外网测速','Internet speed')}}</h2><div class="fastnet-actions"><WidgetPrimaryAction v-if="fastnet.installed&&fastnet.enabled" icon="activity" :disabled="fastnet.running" @click="startFastnetTest" data-fastnet-test>{{fastnet.running?t('测速中','Testing'):t('测速','Test')}}</WidgetPrimaryAction><WidgetMenu :label="t('外网测速更多操作','Internet speed actions')" :items="[{id:'details',label:t('查看上次结果','Last test details')}]" @select="details=true"/></div></header>
    <template v-if="!fastnet.installed"><div class="fastnet-empty"><StudioGlyph name="activity"/><strong>{{t('安装 FastNet，了解外网表现','Install FastNet to test your internet')}}</strong><small>{{t('测速速率 · NAT · IPv6','Speed · NAT · IPv6')}}</small><button :disabled="fastnet.installing" @click="installFastnet" data-fastnet-install>{{fastnet.installing?t('正在安装…','Installing…'):fastnet.error?t('重试安装','Retry installation'):t('安装 FastNet','Install FastNet')}}</button></div><p v-if="fastnet.error" class="fastnet-notice" role="alert">{{fastnet.error}}</p></template>
    <template v-else>
      <div class="fastnet-speeds"><div><span>↓ {{t('上次下载','Last download')}}</span><strong>{{fastnet.down||'—'}}<small>Mbps</small></strong></div><div><span>↑ {{t('上次上传','Last upload')}}</span><strong>{{fastnet.down?fastnet.up:'—'}}<small>Mbps</small></strong></div></div>
      <p class="fastnet-time">{{t('上次测速','Last tested')}}<time>{{fastnet.down?fastnet.time:t('尚未测速','No test yet')}}</time></p>
      <div class="fastnet-statuses"><div><span>NAT</span><strong>{{fastnet.nat}}</strong></div><div><span>IPv6</span><strong :class="{warning:fastnet.ipv6==='部分可用'}"><i :class="{good:fastnet.ipv6==='可用'}"></i>{{fastnet.ipv6}}</strong></div></div>
      <div v-if="fastnet.running" class="fastnet-progress" role="status"><span>{{fastnetStages[Math.min(fastnet.progress,4)]}}<small>{{t('上次结果保持显示','Previous results remain visible')}}</small></span><button @click="stopFastnetTest">{{t('停止','Stop')}}</button><progress max="5" :value="fastnet.progress"></progress></div>
      <p v-else-if="!fastnet.enabled" class="fastnet-notice">{{t('服务未启用','Service disabled')}}<button @click="fastnet.enabled=true">{{t('启用服务','Enable service')}}</button></p>
      <p v-else-if="fastnet.error" class="fastnet-notice" role="alert">{{fastnet.error}}<button @click="startFastnetTest">{{t('重试','Retry')}}</button></p>
      <p v-else class="fastnet-footer" role="status"><span>{{fastnet.note||(fastnet.time.startsWith('3 天前')?t('结果较早，建议重新测速','Older result; consider testing again'):t('主宽带 · 默认出口','Primary WAN · Default route'))}}</span><small>FastNet</small></p>
    </template>
  </section>
  <NetworkWidgetDialog v-if="details" :title="t('上次外网测速','Last internet test')" @close="details=false"><div class="fastnet-details"><dl><dt>{{t('上次速率','Last speed')}}</dt><dd>{{fastnet.down?`↓ ${fastnet.down} / ↑ ${fastnet.up} Mbps`:t('尚未测速','No test yet')}}</dd><dt>{{t('测速时间','Test time')}}</dt><dd>{{fastnet.down?fastnet.time:'—'}}</dd><dt>NAT</dt><dd>{{fastnet.nat}}</dd><dt>IPv6</dt><dd>{{fastnet.ipv6}}</dd><dt>{{t('测试路径','Path')}}</dt><dd>{{t('路由器 → 互联网 · 默认出口','Router → Internet · Default route')}}</dd></dl><p>{{t('点击卡片上的“测速”会使用外网下载和上传带宽，并更新 NAT、IPv6 情况。结果代表上次检测时的网络表现。','Testing uses internet bandwidth and updates NAT and IPv6. Results describe conditions at the time of testing.')}}</p><p class="fastnet-demo">{{t('设计预览 · 模拟数据，不执行真实测速。','Design preview · Synthetic data, no real tests.')}}</p></div></NetworkWidgetDialog>
</template>
<script setup lang="ts">
import {ref} from 'vue'
import StudioGlyph from './StudioGlyph.vue'
import WidgetMenu from './WidgetMenu.vue'
import WidgetPrimaryAction from './WidgetPrimaryAction.vue'
import NetworkWidgetDialog from './NetworkWidgetDialog.vue'
import {fastnet,fastnetStages,startFastnetTest,stopFastnetTest,installFastnet} from './fastnetModel'
defineProps<{summary?:boolean}>()
const details=ref(false),english=window.vue_lang==='en',t=(zh:string,en:string)=>english?en:zh
</script>
<style src="./fastnetCard.css" scoped></style>
