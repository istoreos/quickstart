<template>
  <div class="page-container">
    <div style="height: 48px;text-align: right;display: flex;justify-content: flex-end;align-items: start;">
      <div style="display: flex;align-items: center;">
        <a onclick="void(0)" href="https://www.istoreos.com/" target="_blank"
          style="text-decoration: none;color: white;line-height: 1.5em;">{{ $gettext("iStoreOS官网") }}</a>
        <span :title="$gettext('模块设置')" class="model_btn" @click="moduleSettingsVisible = true">
          <moduleIcon style="width: 16px;height: 16px;" />
        </span>
      </div>
    </div>

    <!-- 更新提示模块 -->
    <!-- <SystemUpdateBanner v-if="useSystem.checkUpdate?.needUpdate" /> -->

    <!-- 快捷操作模块 -->
    <div class="card-container" v-if="moduleStates.quickActions">
      <template v-for="(item, index) in cards" :key="index">
        <QuickActions :card="item" v-if="item.icon !== 'baby' || isShow" @click="handleCardClick" />
      </template>
      <!-- <QuickActions v-for="(item, index) in cards" :key="index" :card="item" @click="handleCardClick" /> -->
    </div>

    <!-- 网络流量和网络连接IP地址模块 -->
    <div v-if="showNetworkModules" style="margin-top: 10px;">
      <!-- 当隐藏“网络连接和IP地址”，但显示“网络接口状态”时：上下排列，占满宽度 -->
      <div class="network-stack" v-if="stackFlowAndInterface">
        <div class="stack-item">
          <NetworkFlow style="min-height: 600px;" />
        </div>
        <div class="stack-item" style="margin-top: 10px;">
          <NetworkInterface />
        </div>
      </div>
      <!-- 否则使用原左右布局；当仅“网络流量”独占一行时，设置最小高度600px -->
      <div class="network-container" v-else>
        <div class="left-box" :class="{ 'full-row': onlyNetworkFlow }" v-if="moduleStates.networkFlow">
          <NetworkFlow :style="onlyNetworkFlow ? { minHeight: '600px' } : undefined" />
        </div>
        <div class="right-box" :class="{ 'single-card': singleRightConnection }" style="overflow: visible;"
          v-if="showNetworkRightModules">
          <NetworkConnection v-if="moduleStates.networkConnection" :class="{ 'fill-card': singleRightConnection }" />
          <NetworkInterface v-if="moduleStates.networkInterface"
            :style="{ marginTop: moduleStates.networkConnection ? '10px' : '0px' }" />
        </div>
      </div>
    </div>

    <!-- 卡片和网络接口状态模块 -->
    <div class="network-container align-c" v-if="moduleStates.configModule">
      <div class="left-box">
        <Item />
      </div>
      <!-- <div class="right-box">
        <NetworkInterface />
      </div> -->
    </div>

    <!-- 其他模块 -->
    <div class="other-container" v-if="visibleOtherModules.length">
      <div class="grid-container">
        <div class="grid-item" v-for="(module, index) in visibleOtherModules" :key="module.key"
          :style="{ flexBasis: calcWidth(index, visibleOtherModules.length) }">
          <component :is="module.component" />
        </div>
      </div>
    </div>

    <!-- 系统信息 -->
    <div class="system" v-if="moduleStates.systemInfo">
      <SystemInfo />
    </div>

    <ModuleSettings v-model:visible="moduleSettingsVisible" :modules="moduleSettingOptions" :states="moduleStates"
      @save="handleModuleSettingsSave" />

  </div>
</template>

<script lang="ts" setup>
import SystemUpdateBanner from "./components/SystemUpdateBanner.vue"
import DownloadService from "./service/DownloadService.vue"
import RemoteDomain from "./service/RemoteDomain.vue"

import NetworkFlow from "./network/NetworkFlow.vue"
import NetworkConnection from "./network/NetworkConnection.vue"
import NetworkInterface from "./network/NetworkInterface.vue"
import moduleIcon from "/@/components/svg/module.vue"

import QuickActions from "./nav/QuickActions.vue"
import Item from "./nav/Item.vue"

import Storage from "./Storage/index.vue"

import DiskInfo from "./info/DiskInfo.vue"
import Docker from "./info/Docker.vue"
import SystemInfo from "./info/SystemInfo.vue"

import { computed, ref, watch } from 'vue'
import { getData, useNetworkStore, useSystemStore } from "/@/plugins/store"
import request from '/@/request';
import { useRouter } from 'vue-router'
import appUtils from "/@/utils/app";
import Toast from "/@/components/toast";
import { feature } from "/@/utils/features"
import { useGettext } from '/@/plugins/i18n'
import ModuleSettings from "./components/ModuleSettings.vue"
import type { ModuleSettingItem } from "./components/ModuleSettings.vue"
import type { Component } from 'vue'
const { $gettext } = useGettext()
const router = useRouter()
getData()

const useSystem = useSystemStore()
const enableAutoCheckUpdateOri = ref(true)
if (window.quickstart_configs?.update?.disable) {
  enableAutoCheckUpdateOri.value = false
}
if (enableAutoCheckUpdateOri.value) {
  setTimeout(() => {
    useSystem.requestCheckUpdate()
  }, 1100)
}

const networkStore = useNetworkStore()
const deviceList = computed(() => networkStore.deviceList)

type IconType = 'navigation' | 'wifi' | 'topology' | 'speed' | 'baby' | 'appStore';
type ColorTheme = 'purple' | 'blue' | 'green' | 'pink' | 'skyblue' | 'orange';
interface Card {
  title: string;
  subtitle: string;
  icon: IconType;
  color: ColorTheme;
  badge?: string;
  status?: string;
  isActive?: boolean;
  extra?: string;
  num?: number;
  tag?: string;
  link?: string;
  alink?: string;
}
const cards = computed<Card[]>(() => [
  {
    icon: 'navigation' as IconType,
    title: $gettext("网络向导"),
    subtitle: $gettext("网络配置引导"),
    tag: $gettext('智能配置'),
    status: '',
    extra: '',
    color: 'purple' as ColorTheme,
    link: '/network'
  },
  {
    icon: 'topology' as IconType,
    title: $gettext('局域网设备管理'),
    subtitle: $gettext('管理网络中的所有设备'),
    badge: $gettext('推荐'),
    status: '',
    num: deviceList.value?.devices?.length || 0,
    color: 'blue' as ColorTheme,
    link: '/devicemanagement'
  },
  // {
  //   icon: 'wifi' as IconType,
  //   title: $gettext('无线网络'),
  //   subtitle: $gettext('WiFi设置与管理'),
  //   status: $gettext('已启用'),
  //   extra: $gettext('2.4G/5G双频'),
  //   color: 'green' as ColorTheme,
  //   isActive: true,
  //   link: '/quickwifi'
  // },

  {
    icon: 'speed' as IconType,
    title: $gettext('网络测速'),
    subtitle: $gettext('检测网络速度'),
    status: '',
    // tag: $gettext('点击测试'),
    color: 'skyblue' as ColorTheme,
    link: '/networkSpeedTest'
  },
  {
    icon: 'baby' as IconType,
    title: $gettext('家长控制'),
    subtitle: $gettext('儿童上网保护'),
    badge: $gettext('保护'),
    status: '',
    extra: '',
    color: 'pink' as ColorTheme,
    isActive: true,
    alink: '/cgi-bin/luci/admin/services/appfilter'
  },
  {
    icon: 'appStore' as IconType,
    title: $gettext('应用商店'),
    subtitle: $gettext('百款应用，自由选择'),
    status: '',
    color: 'orange' as ColorTheme,
    alink: '/cgi-bin/luci/admin/store/pages/store'
  },
])


const handleCardClick = (card: any) => {
  if (!card.title) return
  if (card.icon == 'speed') {
    return onClickHomebox()
  }
  if (card.icon == 'baby') {
    return onClickBaby()
  }
  if (card.link) {
    router.push(card.link)
  } else if (card.alink) {
    window.location.href = card.alink
  }
}

const onClickBaby = async () => {
  if (await appUtils.checkAndInstallApp("luci-app-oaf", "应用过滤")) {
    window.location.href = '/cgi-bin/luci/admin/services/appfilter'
  }
}

const onClickHomebox = async () => {
  const pkg = "app-meta-fastnet"
  if (await appUtils.checkAndInstallApp(pkg, $gettext("FastNet"))) {
    window.location.href = "/cgi-bin/luci/admin/services/fastnet"
  }
}

const isShow = ref(false); //  true: 已启用, false: 未启用
const getStatus = async () => {
  try {
    const res = await request.App.Check.POST({
      name: 'luci-app-oaf'
    })
    if (res?.data) {
      const { result, error } = res.data
      if (error) {
        isShow.value = false
      } else if (result) {
        if (result.status == "installed") {
          isShow.value = true
        } else {
          isShow.value = false
        }
      } else {
        isShow.value = false
      }
    }
  } catch (error) {
    isShow.value = false
  }
}
getStatus()

type ModuleKey =
  | 'quickActions'
  | 'networkFlow'
  | 'networkConnection'
  | 'networkInterface'
  | 'configModule'
  | 'diskInfo'
  | 'storage'
  | 'docker'
  | 'downloadService'
  | 'remoteDomain'
  | 'systemInfo'

type ModuleStates = Record<ModuleKey, boolean>

const defaultModuleStates: ModuleStates = {
  quickActions: true,
  networkFlow: true,
  networkConnection: true,
  networkInterface: true,
  configModule: true,
  diskInfo: true,
  storage: true,
  docker: true,
  downloadService: true,
  remoteDomain: true,
  systemInfo: true,
}

// 初始状态设为全部隐藏，等待接口返回后再根据接口数据更新
const moduleStates = ref<ModuleStates>({
  quickActions: false,
  networkFlow: false,
  networkConnection: false,
  networkInterface: false,
  configModule: false,
  diskInfo: false,
  storage: false,
  docker: false,
  downloadService: false,
  remoteDomain: false,
  systemInfo: false,
})
const moduleSettingsVisible = ref(false)

const moduleSettingOptions = computed<ModuleSettingItem[]>(() => {
  const items: ModuleSettingItem[] = [
    {
      key: 'quickActions',
      title: $gettext('快捷入口'),
      description: $gettext('顶部快捷捷径按钮组'),
    },
    {
      key: 'networkFlow',
      title: $gettext('网络流量'),
      description: $gettext('实时流量统计图表'),
    },
    {
      key: 'networkConnection',
      title: $gettext('网络连接和IP地址'),
      description: $gettext('联网时间和设备信息'),
    },
    {
      key: 'networkInterface',
      title: $gettext('网络接口状态'),
      description: $gettext('网络接口详细信息'),
    },
    {
      key: 'configModule',
      title: $gettext('配置模块'),
      description: $gettext('内网配置、DNS配置等工具'),
    },
    {
      key: 'diskInfo',
      title: $gettext('磁盘信息'),
      description: $gettext('磁盘使用情况与容量概览'),
    },
    {
      key: 'storage',
      title: $gettext('存储服务'),
      description: $gettext('共享与存储服务概览'),
    },
    {
      key: 'downloadService',
      title: $gettext('下载服务'),
      description: $gettext('下载任务与服务状态'),
    },
    {
      key: 'remoteDomain',
      title: $gettext('远程域名'),
      description: $gettext('远程访问域名管理'),
    },
    {
      key: 'systemInfo',
      title: $gettext('系统信息'),
      description: $gettext('设备系统信息概览'),
    },
  ]

  if (feature('dockerd')) {
    items.splice(7, 0, {
      key: 'docker',
      title: $gettext('Docker模块'),
      description: $gettext('容器运行状态与管理'),
    })
  }
  return items
})

const showNetworkModules = computed(
  () =>
    moduleStates.value.networkFlow ||
    moduleStates.value.networkConnection ||
    moduleStates.value.networkInterface
)

const showNetworkRightModules = computed(
  () => moduleStates.value.networkConnection || moduleStates.value.networkInterface
)

// 当隐藏“网络连接和IP地址”但显示“网络接口状态”，需要上下排列并占满宽度
const stackFlowAndInterface = computed(
  () =>
    moduleStates.value.networkFlow &&
    !moduleStates.value.networkConnection &&
    moduleStates.value.networkInterface
)

// 当“网络流量”模块独占一行（右侧无任何模块）时，用于提升高度
const onlyNetworkFlow = computed(
  () =>
    moduleStates.value.networkFlow &&
    !moduleStates.value.networkConnection &&
    !moduleStates.value.networkInterface
)

// 当只有“网络连接和IP地址”显示在右侧时，保持卡片高度与左侧同步
const singleRightConnection = computed(
  () => moduleStates.value.networkConnection && !moduleStates.value.networkInterface
)

if (typeof window !== 'undefined') {
  watch(
    () => [moduleStates.value.networkConnection, moduleStates.value.networkInterface],
    () => {
      requestAnimationFrame(() => {
        window.dispatchEvent(new Event('resize'))
      })
    }
  )
}

interface VisibleOtherModule {
  key: ModuleKey
  component: Component
}

const visibleOtherModules = computed<VisibleOtherModule[]>(() => {
  const modules: VisibleOtherModule[] = []
  if (moduleStates.value.diskInfo) {
    modules.push({ key: 'diskInfo', component: DiskInfo })
  }
  if (moduleStates.value.storage) {
    modules.push({ key: 'storage', component: Storage })
  }
  if (feature('dockerd') && moduleStates.value.docker) {
    modules.push({ key: 'docker', component: Docker })
  }
  if (moduleStates.value.downloadService) {
    modules.push({ key: 'downloadService', component: DownloadService })
  }
  if (moduleStates.value.remoteDomain) {
    modules.push({ key: 'remoteDomain', component: RemoteDomain })
  }
  return modules
})

const calcWidth = (index: number, total: number) => {
  if (total <= 0) {
    return '100%'
  }
  if (total <= 2) {
    return `calc((100% - ${(total - 1) * 24}px) / ${total})`
  }
  const firstRowCount = Math.min(2, total)
  if (index < firstRowCount) {
    return `calc((100% - ${(firstRowCount - 1) * 24}px) / ${firstRowCount})`
  }
  const remaining = total - firstRowCount
  if (remaining <= 0) {
    return '100%'
  }
  return `calc((100% - ${(remaining - 1) * 24}px) / ${remaining})`
}

// 不再使用 localStorage，完全依赖接口控制模块显示隐藏

// 保存初始状态用于比较是否有改动
const originalModuleStates = ref<ModuleStates | null>(null)

const handleModuleSettingsSave = async (value: Record<string, boolean>) => {
  // 检查是否有改动
  const hasChanges = originalModuleStates.value &&
    Object.keys(value).some(key => value[key] !== originalModuleStates.value?.[key as ModuleKey])

  if (!hasChanges) {
    // 没有改动，直接关闭弹窗
    moduleSettingsVisible.value = false
    return
  }

  // 有改动，先保存到服务器，成功后再更新本地状态
  const newStates = { ...moduleStates.value, ...value } as ModuleStates
  const success = await saveModuleSettings(newStates)

  // 只有保存成功才更新本地状态并关闭弹窗
  if (success) {
    moduleStates.value = newStates
    moduleSettingsVisible.value = false
  }
  // 如果失败，弹窗保持打开状态，用户可以重新尝试或取消
}

// ------ 接口回显/保存 ------
const fetchModuleSettings = async () => {
  try {
    const res = await request.ModuleSettings.GET()
    
    // 尝试多种可能的路径
    const diableDisplay = (res?.data?.result?.diableDisplay) as string[] | undefined

    // 如果 diableDisplay 是空数组或不存在，则显示所有模块
    if (!diableDisplay || diableDisplay.length === 0 || (diableDisplay.length === 1 && diableDisplay[0] === '')) {
      // 所有模块显示
      const newStates: ModuleStates = { ...defaultModuleStates }
      moduleStates.value = newStates
      originalModuleStates.value = { ...newStates }
    } else {
      // 将所有模块设置为显示
      const newStates: ModuleStates = { ...defaultModuleStates }

      // 将 diableDisplay 中的模块设置为隐藏（过滤掉空字符串）
      diableDisplay.forEach((key) => {
        if (key && key.trim() && key in newStates) {
          newStates[key as ModuleKey] = false
        }
      })
      moduleStates.value = newStates
      originalModuleStates.value = { ...newStates }
    }
  } catch (e) {
    console.warn('[ModuleSettings] fetch failed', e)
    // 接口失败时，使用默认值（全部显示）
    moduleStates.value = { ...defaultModuleStates }
    originalModuleStates.value = { ...defaultModuleStates }
  }
}

const saveModuleSettings = async (states: ModuleStates): Promise<boolean> => {
  try {
    // 构建 diableDisplay 数组：收集所有为 false 的模块 key
    const diableDisplay: string[] = []
    Object.keys(states).forEach((key) => {
      if (!states[key as ModuleKey]) {
        diableDisplay.push(key)
      }
    })

    // 调用 POST 接口（注意：接口字段名是 diableDisplay）
    await request.ModuleSettings.POST({ diableDisplay })

    // 更新原始状态
    originalModuleStates.value = { ...states }

    // 提示保存成功
    Toast.Success($gettext("保存成功"))
    return true
  } catch (e) {
    console.error('[ModuleSettings] save failed', e)
    Toast.Warning($gettext("保存失败！"))
    return false
  }
}

// 监听弹窗打开，回显数据
watch(moduleSettingsVisible, (visible) => {
  if (visible) {
    // 弹窗打开时，保存当前状态作为原始状态
    originalModuleStates.value = { ...moduleStates.value }
  }
})

// 首次进入立即从接口获取模块状态
fetchModuleSettings()
</script>

<style lang="scss" scoped>
.page-container {

  .model_btn {
    cursor: pointer;
    margin-left: 16px;
  }

  .card-container {
    display: flex;
    flex-wrap: nowrap;
    overflow-x: auto;
    gap: 16px;
    width: 100%;
    padding-bottom: 10px;
    overflow-y: hidden;
    -webkit-overflow-scrolling: touch;
    scrollbar-gutter: stable;
    scrollbar-width: thin;
    scrollbar-color: rgba(0, 0, 0, .35) transparent;

    &::-webkit-scrollbar {
      height: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background: #ccc;
      border-radius: 3px;
    }
  }

  .card-container>* {
    flex: 0 0 auto;
  }

  :deep(.card-container::-webkit-scrollbar) {
    height: 8px;
  }

  :deep(.card-container::-webkit-scrollbar-thumb) {
    border-radius: 4px;
    background: rgba(0, 0, 0, .35);
  }

  :deep(.card-container::-webkit-scrollbar-track) {
    background: transparent;
  }

  .network-container {
    display: flex;
    gap: 24px;
    width: 100%;
    margin-top: 20px;
    align-items: stretch;

    .left-box {
      flex: 2;
      // overflow: hidden;
      min-width: 0;
    }

    .right-box {
      flex: 1;
      overflow: hidden;
      min-width: 0;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
    }
  }

  /* 上下栈布局：网络流量 + 网络接口状态 各占一行 */
  .network-stack {
    display: flex;
    flex-direction: column;
    width: 100%;
    align-items: stretch;

    .stack-item {
      width: 100%;
    }
  }

  /* 当网络流量独占一行时，提高最小高度到600px */
  .full-row :deep(.network_container) {
    min-height: 600px;
  }

  /* 直接作用于 NetworkConnection 根元素，保证内部卡片拉伸填满（上上次版本） */
  .fill-card {
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  .fill-card :deep(.reusable-card) {
    flex: 1 1 auto;
    display: flex;
    flex-direction: column;
  }

  .fill-card :deep(.card-body) {
    flex: 1 1 auto;
  }

  .align-c {
    align-items: center;
  }

  .other-container {
    width: 100%;
    margin-top: 20px;

    .grid-container {
      display: flex;
      flex-wrap: wrap;
      gap: 24px;

      .grid-item {
        display: flex;
        // align-items: center;
        justify-content: center;
        border-radius: 8px;
      }
    }
  }

  .btns {
    margin-top: 20px;
  }

  .system {
    margin-top: 24px;
  }
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 1300px) {
  .page-container {

    .other-container {
      width: 100%;
      margin-top: 16px;

      .grid-container {
        flex-direction: column;
        gap: 12px;

        .grid-item {
          border-radius: 6px;
        }
      }
    }

  }
}
</style>

<style lang="scss" scoped>
@media screen and (max-width: 768px) {
  .page-container {
    .card-container {
      flex-wrap: nowrap;
      overflow-x: auto;
      -webkit-overflow-scrolling: touch;
      gap: 16px;

      &>* {
        flex: 0 0 auto;
        min-width: 280px;
      }

      scrollbar-width: none;
      -ms-overflow-style: none;

      &::-webkit-scrollbar {
        display: none;
      }
    }

    .network-container {
      flex-direction: column;
      margin-top: 10px;
      gap: 10px;

      .right-box {
        flex: none;
        width: 100%;
      }
    }

    .other-container {
      width: 100%;
      margin-top: 16px;

      .grid-container {
        flex-direction: column;
        gap: 12px;

        .grid-item {
          border-radius: 6px;
        }
      }
    }

  }
}
</style>
