<template>
    <label>
        <div class="select-editable">
            <select v-model.trim="userSelect" autocomplete="off" @change="onSelectChange">
                <option selected value="">{{ $gettext("请选择%{title}", {title}) }}</option>
                <option :value="item.key" v-for="(item, i) in options" :key="i">{{
                        item.value || item.key
                }}
                </option>
                <option value="useInput">{{ $gettext("- -自定义- -") }}</option>
            </select>
            <input type="text" v-model.trim="userInput" required
                v-if="userSelect == 'useInput'" :placeholder="$gettext('请输入%{title}',{title})" @change="onInputChange"/>
        </div>
    </label>
</template>
<script setup lang="ts">
import { computed,ref,onMounted,PropType,watch } from 'vue';
import { useGettext,formatNumber } from '/@/plugins/i18n'
const { $gettext,$ngettext } = useGettext()

const props = defineProps({
    modelValue: {
        type: String,
        required: true
    },
    title: {
        type: String,
        default: ""
    },
    options: {
        type: Array as PropType<{
            key: string,
            value?: string
        }[]>,
        default: []
    }
})

const userSelect = ref("")
const userInput = ref("")


const emits = defineEmits(['update:modelValue'])
const value = computed<string>({
    get: ()=>props.modelValue.valueOf(),
    set: (v: string)=>emits('update:modelValue', v)
})
const updateValue = (v: string)=>{
    if (v === userSelect.value || ('useInput' === userSelect.value && v === userInput.value)) {
        // avoid dead loop
        return
    }
    if ('' === v || props.options.some((o)=>o.key===v)) {
        userSelect.value = v
    } else {
        userInput.value = v
        userSelect.value = 'useInput'
    }
}
watch(()=>props.modelValue, (v)=>{
    updateValue(v)
})
onMounted(() => {
    const v = value.value;
    updateValue(v)
})

const onSelectChange = (e: Event) => {
    if (userSelect.value === "useInput") {
        value.value = userInput.value
    } else {
        value.value = userSelect.value
    }
}
const onInputChange = (e: Event) => {
    value.value = userInput.value
}
</script>
<style lang="scss" scoped>

.select-editable {
    position: relative;
    // background-color: white;
    line-height: 1.5rem;
    padding: 0.5rem 0.75rem;
    border: 1px solid #dee2e6;
    border-radius: 0.25rem;
    margin: 0.25rem 0.1rem;
    select, input {
        height: 100%;
        padding: 0;
        border: none;
        margin: 0;
    }
}

.select-editable select {
    position: relative;
    width: 100%;
}

.select-editable input {
    position: absolute;
    top: 0px;
    left: 0.75rem;
    width: 88%;
}

.select-editable select:focus,
.select-editable input:focus {
    outline: none;
    box-shadow: none;
}

</style>