<template>
    <div id="page">
        <h2 name="content">S.M.A.R.T.</h2>
        <div class="cbi-map-descr">
            <p>
                {{ $gettext("S.M.A.R.T.，全称为“Self-Monitoring Analysis and Reporting Technology”，即“自我监测、分析及报告技术”，") }}
            </p>
            <p>
                {{ $gettext("是一种自动的硬盘状态检测与预警系统和规范。通过在硬盘硬件内的检测指令对硬盘的硬件如磁头、盘片、马达、") }}
            </p>
            <p>
                {{ $gettext("电路的运行情况进行监控、记录并与厂商所设定的预设安全值进行比较，若监控情况将要或已超出预设安全值的安全范围，") }}
            </p>
            <p>
                {{ $gettext("就可以通过主机的监控硬件或软件自动向用户作出警告并进行轻微的自动修复，以提前保障硬盘数据的安全。") }}
            </p>
        </div>
        <ul class="tabs">
            <router-link :to="item.to" custom v-slot="{ route, href, navigate, isActive, isExactActive }"
                v-for="item in menus" :key="item.to">
                <li :class="{
                    'active cbi-tab': isActive && isExactActive,
                }">
                    <a :href="href" @click="navigate">{{ item.name }}</a>
                </li>
            </router-link>
        </ul>
        <router-view v-slot="{ Component, route }" name="default" v-if="ok">
            <Suspense>
                <template #default>
                    <component :is="Component" :key="route.path" :config="config" :saveData="saveData" />
                </template>
            </Suspense>
        </router-view>
    </div>
</template>
<script setup lang="ts">
import { reactive, ref } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

import Toast from '/@/components/toast';
import request from '/@/request';
const menus = [
    {
        to: "/smart",
        name: $gettext("常规设置"),
    },
    {
        to: "/smart/task",
        name: $gettext("计划任务"),
    },
    {
        to: "/smart/log",
        name: $gettext("查看日志"),
    },
]
const ok = ref(false);
const config = reactive<PropsSmartConfig>({
    global: {
        enable: false,
        powermode: "never",
        tmpDiff: 0,
        tmpMax: 0
    },
    devices: [],
    tasks: []
})
const setConfig = (data: ResponseSmartConfig) => {
    const { global, devices, tasks } = data
    if (global) {
        config.global.enable = global.enable || false
        config.global.powermode = global.powermode || "never"
    }
    config.devices = devices || []
    config.tasks = tasks || []
}
const getData = async () => {
    try {
        const res = await request.Smart.Config.GET()
        if (res.data) {
            const { result } = res.data
            if (result) {
                setConfig(result)
            }
        }
    } catch (error) {
    } finally {
        ok.value = true
    }
}
getData();
const saveData = async (data: RequestSmartConfig) => {
    const load = Toast.Loading($gettext("保存中..."))
    try {
        const res = await request.Smart.Config.POST(data)
        if (res.data) {
            console.log(res.data);
            const { success, error, result } = res.data
            if (error) {
                throw error
            }
            if ((success || 0) == 0) {
                Toast.Success($gettext("保存成功"))
                if (result) {
                    setConfig(result)
                }
            }

        }
    } catch (error) {
        Toast.Error(`${error}`)
    } finally {
        load.Close()
    }
}
</script>
<style lang="scss" scoped>
#page {
    .cbi-map-descr {
        margin-bottom: 1rem;
    }

    :deep(.cbi-section) {
        padding: 1rem;
    }

    :deep(span.cbi-page-actions.control-group) {
        width: 100%;
        display: block;
    }
}
</style>