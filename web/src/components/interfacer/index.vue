<template>
    <div class="app-interfaces" ref="el">
        <template v-if="checkELBtn">
            <a class="btn-f" @click="ClickF">
                <span>
                    {{ "<" }} </span>
            </a>
            <a class="btn-r" @click="ClickR">
                <span>
                    {{ ">" }}
                </span>
            </a>
        </template>
        <item-vue :item="item" v-for="(item, i) in portList" :transform="transform" />
    </div>
</template>
<script setup lang="ts">
import { computed, onMounted, PropType, reactive, ref, nextTick } from 'vue';
import itemVue from './item.vue';
const props = defineProps({
    portList: {
        type: Array as PropType<NetworkPort[]>,
        required: true
    }
})
const el = ref<HTMLElement | null>()
const elMaxWidth = ref(0) //元素的最大宽度
const clientWidth = ref(0) //元素的可见宽度
const transform = ref(0)
const checkELBtn = ref(false)
const ClickF = () => {
    if (transform.value >= 0) {
        transform.value = 0
        return
    }
    transform.value += 100
}
const ClickR = () => {
    if (transform.value <= (0 - elMaxWidth.value + clientWidth.value)) {
        transform.value = 0 - elMaxWidth.value + clientWidth.value
        return
    }
    transform.value -= 100
}

onMounted(() => {
    nextTick(() => {
        if (el.value) {
            elMaxWidth.value = el.value.scrollWidth
            clientWidth.value = el.value.clientWidth
            checkELBtn.value = elMaxWidth.value > clientWidth.value
        }
    })
})
</script>
<style lang="scss" scoped>
.app-interfaces {
    width: 100%;
    height: 80px;
    display: flex;
    flex-wrap: nowrap;
    overflow: hidden;

    a {
        list-style: none;
        text-decoration: none;
    }

    a.btn-f {
        position: absolute;
        left: 0;
        width: 50px;
        height: 80px;
        left: 0;
        line-height: 80px;
        text-align: center;
        color: #fff;
        font-size: 26px;
        cursor: pointer;
        background-color: #00000059;
        opacity: 0;
        transition: 0.3s;
        z-index: 1;

        &:hover {
            opacity: 1;
            transition: 0.3s;
        }
    }

    a.btn-r {
        position: absolute;
        right: 0;
        width: 50px;
        line-height: 80px;
        text-align: center;
        color: #fff;
        font-size: 26px;
        cursor: pointer;
        background-color: #00000059;
        opacity: 0;
        transition: 0.3s;
        z-index: 1;

        &:hover {
            opacity: 1;
            transition: 0.3s;
        }
    }
}
</style>