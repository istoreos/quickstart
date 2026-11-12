<template>
  <IndexPage v-if="!legacy&&route.query.variant!=='ordered'&&!previous&&!oldCanvas&&route.query.stage!=='compare'&&!['grid','free'].includes(String(route.query.variant||''))"/>
  <OrderedHome v-else-if="!legacy&&!previous&&!oldCanvas&&route.query.stage!=='compare'&&!['grid','free'].includes(String(route.query.variant||''))"/>
  <LayoutDesignHome v-else-if="!legacy&&!previous&&!oldCanvas"/>
  <CanvasHome v-else-if="!legacy&&!previous"/>
  <StudioHome v-else-if="previous" />
  <IndexPage v-else>
    <template #after-shortcuts><DeviceSummary v-if="layout==='above-network' && summaryVisible" /></template>
    <template #after-network><DeviceSummary v-if="layout==='below-network' && summaryVisible" /></template>
  </IndexPage>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import IndexPage from '../pages/index/index.vue'
import DeviceSummary from './DeviceSummary.vue'
import StudioHome from './StudioHome.vue'
import CanvasHome from './CanvasHome.vue'
import LayoutDesignHome from './LayoutDesignHome.vue'
import OrderedHome from './OrderedHome.vue'
const route=useRoute()
const layout=computed(()=>String(route.query.layout || 'baseline'))
const legacy=computed(()=>['baseline','above-network','below-network'].includes(String(route.query.layout||'studio')))
const previous=computed(()=>route.query.layout==='studio'||route.query.layout!=='canvas'&&['studio','network','device'].includes(String(route.query.variant||'')))
const oldCanvas=computed(()=>['canvas','sidebar','stage'].includes(String(route.query.variant||'')))
const summaryVisible=computed(()=>route.query.summary!=='hidden')
</script>
