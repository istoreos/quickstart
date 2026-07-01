<template>
    <ul class="app-container_samba">
        <template v-if="sambas">
            <li class="sambas-item">
                <div class="sambas-item_name">
                    <span>{{ $gettext("当前状态:") }}</span>
                </div>
                <div class="sambas-item_value">
                    <span>{{ sambas?.length ? $gettext('已启用') : $gettext('未启用') }}</span>
                </div>
            </li>
        </template>

        <li class="sambas-item">
            <div class="sambas-item_name tit">
                <span>{{ $gettext("地址") }}</span>
            </div>
            <div class="sambas-item_value tit">
                <span>{{ $gettext("目录") }}</span>
            </div>
        </li>

        <li class="samba-item" v-for="item in sambas">
            <div class="samba-item_name">
                <span>smb://{{ hostname }}/{{ item.shareName }}</span>
            </div>
            <div class="samba-item_value" :title="item.path">
                <a target="_blank" :href="'/cgi-bin/luci/admin/services/linkease/file/?path=/root' + item.path">{{ item.path
                }}</a>
            </div>
        </li>
    </ul>
</template>
<script setup lang="ts">
import { PropType } from 'vue';
import { useGettext } from '/@/plugins/i18n'
const { $gettext } = useGettext()

const props = defineProps({
    sambas: {
        type: Array as PropType<NasServiceSambaInfo[]>,
    }
})
const hostname = window.location.hostname

</script>
<style lang="scss" scoped>
li.sambas-item {
    width: 100%;
    display: flex;
    flex-wrap: wrap;
    margin: 1rem 0;

    .sambas-item_name {
        flex: 0 0 100%;
        max-width: 50%;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        padding-right: 10px;
        color: var(--app-container_title-color);
    }

    .sambas-item_value {
        flex: 0 0 100%;
        max-width: 50%;
        padding-left: 10px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: var(--app-container_title-color);
    }
}

.app-container_samba {
    li.samba-item {
        width: 100%;
        display: flex;
        flex-wrap: wrap;
        margin: 1rem 0;

        .samba-item_name {
            flex: 0 0 100%;
            max-width: 50%;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            padding-right: 10px;
            color: var(--app-container_title-color);
        }

        .samba-item_value {
            flex: 0 0 100%;
            max-width: 50%;
            padding-left: 10px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;

            button {
                background: none;
                border: none;
                width: 100%;
                text-align: right;
                color: #297ff3;
                cursor: pointer;

                &:hover {
                    opacity: 0.7;
                }
            }
        }
    }
}

.tit {
    color: var(--tit-color);
    font-weight: bold;
    font-size: 16px;
}
</style>
