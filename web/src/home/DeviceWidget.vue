<template><article class="canvas-module module-devices" :class="{'is-device-summary':summary}" :data-device-form="summary?'summary':'standard'" data-canvas-module="devices"><header><div><StudioGlyph name="devices"/><h2>{{t('设备列表','Devices')}}</h2></div><div class="device-header-actions widget-action-group"><WidgetPrimaryAction icon="devices" @click="navigate('/devicemanagement')" data-canvas-view-all>{{t('查看全部','View all')}}</WidgetPrimaryAction><WidgetMenu :label="t('设备更多操作','More device actions')" :items="[{id:'parental',label:t('家长控制','Parental controls')}]" @select="parental"/></div></header>
      <div class="inventory-heading"><strong>{{loading||!devices.length&&errors.inventory?'—':devices.filter(d=>d.online).length}}<span>{{errors.inventory&&devices.length?t('上次在线','previously online'):t('在线','online')}}</span></strong><small>{{loading||!devices.length&&errors.inventory?t('数量待确认','Count unconfirmed'):`${devices.length} ${t('台 · 本次开机发现','discovered this boot')}`}}</small><button @click="$emit('retry')" :aria-label="t('刷新设备','Refresh devices')" data-refresh-inventory>↻</button></div>
      <div v-if="loading" class="module-state"><div class="skeleton" v-for="n in (summary?1:3)" :key="n"></div><span>{{t('正在读取设备清单','Loading devices')}}</span></div>
      <div v-else-if="errors.inventory&&!devices.length" class="module-state"><StudioGlyph v-if="!summary" name="devices"/><strong>{{inventoryError}}</strong><p v-if="!summary">{{t('其他首页模块仍可使用。','Other modules remain available.')}}</p><button @click="$emit('retry')">{{t('重试','Retry')}}</button></div>
      <div v-else-if="!devices.length" class="module-state"><StudioGlyph v-if="!summary" name="devices"/><strong>{{t('暂未发现设备','No devices discovered')}}</strong><p v-if="!summary">{{t('本局域网设备上线后会自动出现。','Devices appear when observed on this LAN.')}}</p><button v-if="!summary" @click="navigate('/devicemanagement')">{{t('添加或查看设备','Add or view devices')}}</button></div>
      <template v-else><p v-if="errors.inventory" class="module-notice">{{t('上次快照 · 当前状态待确认','Previous snapshot · Current state unconfirmed')}}</p><div v-if="!summary" class="canvas-devices"><button v-for="{device,address,description} in deviceRows" :key="`${device.networkId}:${device.deviceId}`" @click="openDevice(device)" :data-canvas-device="device.deviceId" :title="description" :aria-label="description"><DeviceSceneIcon :scene="device.scene" :icon-key="device.iconKey" :label="deviceCategoryLabel(device.scene,translate)"/><span><strong>{{deviceDisplayName(device,translate)}}</strong><small class="device-address" :data-device-address="address.kind" :title="address.description"><bdi dir="ltr">{{address.text}}</bdi></small></span><span class="device-online" :class="{offline:!device.online}"><i></i>{{errors.inventory?t('上次记录','Last recorded'):devicePresenceLabel(device,translate)}}</span></button></div></template>
      <p v-if="capabilityMissing" class="module-notice">{{capabilityText}} {{t('设备清单仍可使用。','Inventory remains available.')}}</p><footer class="inventory-footer"><small>{{t('仅显示本机观察到的设备','Only devices observed by this router')}}</small><details><summary>{{t('地址与路线权限','Address & path permissions')}}</summary><p>{{authorityText}}</p></details></footer>

</article></template>
<script setup lang="ts">
import WidgetPrimaryAction from './WidgetPrimaryAction.vue'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import StudioGlyph from './HomeGlyph.vue'
import WidgetMenu from './WidgetMenu.vue'
import DeviceSceneIcon from '../pages/device/components/deviceSceneIcon.vue'
import { deviceCategoryLabel, deviceDisplayName, devicePresenceLabel, deviceHomeAddress } from '../pages/device/devicePresentation'
import type { DeviceListItem } from '../pages/device/deviceInventory'
const props=defineProps<{data:Record<string,any>;errors:Record<string,string>;devices:DeviceListItem[];contextualActions?:boolean;summaryLimit?:number;diskLimit?:number;summary?:boolean}>()
const emit=defineEmits<{(e:'navigate',path:string):void;(e:'device',id:string):void;(e:'retry'):void}>()
const english=window.vue_lang==='en',t=(zh:string,en:string)=>english?en:zh
const translate=(text:string)=>english?({'手机':'Phone','电脑':'Computer','平板':'Tablet','电视与影音':'TV & media','网络设备':'Network','智能家居':'Smart home','摄像头':'Camera','游戏设备':'Gaming','存储与服务器':'Storage & server','打印机':'Printer','穿戴设备':'Wearable','未知设备':'Unknown device','未命名设备':'Unnamed device','在线':'Online','离线':'Offline','尚未上线':'Not yet seen','设备':'Device','当前IP':'Current IP','历史':'Prev.','历史地址':'Previous address','MAC地址':'MAC address','地址待确认':'Address unconfirmed'}[text]||text):text

import appUtils from '../utils/app'
const shownDevices=computed(()=>props.devices.slice(0,props.summaryLimit??6))
const deviceRows=computed(()=>shownDevices.value.map(device=>{
 const address=deviceHomeAddress(device,translate)
 const classification=[device.brand,device.classification.source==='fallback'?t('类型待确认','Type unconfirmed'):deviceCategoryLabel(device.scene,translate)].filter(Boolean).join(' · ')
 return {device,address,description:[deviceDisplayName(device,translate),address.description,classification,errorsLabel(device)].join(' · ')}
}))
function errorsLabel(device:DeviceListItem){return props.errors.inventory?t('上次记录','Last recorded'):devicePresenceLabel(device,translate)}
const loading=computed(()=>!props.data.inventory&&!props.errors.inventory)
const inventoryError=computed(()=>props.errors.inventory==='permission'?t('登录或读取权限失效','Session or access expired'):t('设备清单暂不可用','Device inventory unavailable'))
const capabilityMissing=computed(()=>props.data.inventory?.health?.state&&props.data.inventory.health.state!=='ready'),capabilityText=computed(()=>t('部分设备观测信息暂不完整。','Some device observations are incomplete.'))
const authorityText=computed(()=>props.data.routerContext?.multiLAN?t('地址与路线设置按各局域网分别管理。','Address and path settings are managed per LAN.'):props.data.routerContext?.routeEditability?.editable?t('本机负责此局域网地址分配，可在设备详情设置固定地址与上网路线。','This router allocates LAN addresses. Configure reservations and Internet paths in device details.'):props.data.routerContext?.dhcpAuthority==='external_observed'?t('本机不负责此局域网地址分配；固定地址与上网路线请到分配地址的主路由设置。','This router does not allocate LAN addresses. Use the allocating router for reservations and Internet paths.'):t('地址分配方尚不明确；固定地址与上网路线暂为只读。','Address allocator unconfirmed. Reservations and Internet paths are read only.'))
const selectedNetwork=computed(()=>props.data.lanNetworks?.length===1?props.data.lanNetworks[0].networkId:'all')
const navigate=(path:string)=>emit('navigate',path==='/devicemanagement'&&props.data.lanNetworks&&!props.data.lanLegacy?`${path}?network=${encodeURIComponent(selectedNetwork.value)}`:path)
function openDevice(device:DeviceListItem){emit('navigate',`/devicemanagement?device=${encodeURIComponent(device.deviceId)}&network=${encodeURIComponent(device.networkId||selectedNetwork.value)}`)}
async function parental(){if(await appUtils.checkAndInstallApp('luci-app-oaf',t('应用过滤','App filtering')))window.location.assign('/cgi-bin/luci/admin/services/appfilter')}
</script>
<style src="./canvasModules.css" scoped></style>
