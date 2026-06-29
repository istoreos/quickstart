<template>
    <div id="page">
        <div style="text-align: left;display: flex;align-items: center;margin-bottom: 20px;padding-top: 4px;">
            <router-link to="/" style="text-decoration: none;color: var(--breadcrumbs-tit-color);line-height: 1.5em;margin-right: 4px;">{{ $gettext("首页") }}</router-link>
            <svg width="20" height="20" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
                <path d="M20 30 L50 50 L20 70" stroke="#d6dbf8" stroke-width="8" stroke-linecap="round" fill="none" />
            </svg>
            <a style="text-decoration: none;color: var(--breadcrumbs-tit-color1);line-height: 1.5em;">{{
                $gettext("设备管理") }}</a>
        </div>
    </div>
    <div class="device_container" style="color: black;">
        <div class="tab-container">
            <div class="tabs_box_g">
                <button v-for="(tab, index) in tabs" :key="index" @click="activeTab = index"
                    :class="{ active: activeTab === index }">
                    {{ $gettext(tab) }}
                </button>
            </div>
            <div class="tab-content_g">
                <div v-if="activeTab === 0" class="content-item">
                    <DeviceListVue @openGloba="openGloba" />
                </div>
                <div v-if="activeTab === 1" class="content-item">
                    <StaticStateListVue />
                </div>
                <div v-if="activeTab === 2" class="content-item">
                    <SpeedLimitListVue @openGloba="openGloba" />
                </div>
                <div v-if="activeTab === 3" class="content-item">
                    <ConfigureVue ref="configureRef" />
                </div>
            </div>
            <div style="height: 30px;"></div>
        </div>
    </div>
</template>
<script setup lang="ts">
import { ref, nextTick, onMounted } from 'vue'
import DeviceListVue from "./deviceList.vue";
import StaticStateListVue from "./staticStateList.vue";
import SpeedLimitListVue from "./speedLimitList.vue";
import ConfigureVue from "./configure.vue";
import request from '/@/request';
import Toast from "/@/components/toast";
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()
const tabs = ref([
    '设备列表',
    '静态分配列表',
    '限速设备列表',
    '全局设置'
]);
const activeTab = ref(0);
const configureRef = ref<any>(null)
const openGloba = async () => {
    activeTab.value = 3
    await nextTick()
    if (configureRef.value) {
        configureRef.value.activeTab = 'ip'
    }
}
</script>
<style lang="scss" scoped>
.tab-container {
    margin: 0 auto;
}

.tabs_box_g {
    display: flex;
}

.tabs_box_g button {
    padding: 14px 24px;
    border: none;
    background: none;
    cursor: pointer;
    font-size: 14px;
    // color: var(--flow-span-color);
    border-radius: 8px 8px 0 0;
    margin: 0;
    transition: all 0.3s ease;
}

.tabs_box_g button.active {
    background: var(--card-bg-color);
    color: #553afe;
    font-weight: bold;
    position: relative;
}

.tab-content_g {
    background: var(--card-bg-color);
    padding: 16px;
    border-radius: 0px 8px 8px 8px;
}

.content-item {
    min-height: 60vh;
}
</style>

<style lang="scss" scoped>
/* 移动端样式 - 基于827px设计图 */
@media (max-width: 827px) {
    .tabs_box_g button {
        padding: 7px 12px !important;
    }

    .tab-content_g {
        border-radius: 0px 0px 8px 8px;
    }
}
</style>