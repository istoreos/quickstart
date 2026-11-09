<template>
  <details ref="menu" class="theme-selector" data-theme-selector>
    <summary :aria-label="t('主题：','Theme: ')+selectedLabel" :title="t('主题','Theme')">
      <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3a9 9 0 1 0 9 9c0-.5-.04-1-.12-1.48A7 7 0 0 1 12 3Z"/></svg>
    </summary>
    <div class="theme-menu" role="radiogroup" :aria-label="t('主题模式','Theme mode')">
      <strong>{{t('主题','Theme')}}</strong>
      <button v-for="option in options" :key="option.value" type="button" role="radio" :aria-checked="preference===option.value" :class="{selected:preference===option.value}" @click="choose(option.value)">
        <span>{{t(option.zh,option.en)}}</span>
        <small v-if="option.value==='system'">{{t('当前：','Current: ')}}{{effective==='dark'?t('深色','Dark'):t('浅色','Light')}}</small>
        <i aria-hidden="true">{{preference===option.value?'✓':''}}</i>
      </button>
      <p>{{t('此选择保存在当前浏览器，并与 LuCI 主题同步。','Saved in this browser and synchronized with the LuCI theme.')}}</p>
    </div>
  </details>
</template>
<script setup lang="ts">
import {computed,onMounted,onUnmounted,ref} from 'vue'
import {getTheme,getThemePreference,setThemePreference,subscribeTheme,type ThemePreference} from '/@/utils/theme'
const english=window.vue_lang==='en',t=(zh:string,en:string)=>english?en:zh
const options:{value:ThemePreference;zh:string;en:string}[]=[
  {value:'system',zh:'跟随系统',en:'Use system setting'},
  {value:'light',zh:'浅色',en:'Light'},
  {value:'dark',zh:'深色',en:'Dark'},
]
const menu=ref<HTMLDetailsElement|null>(null),preference=ref(getThemePreference()),effective=ref(getTheme())
const selectedLabel=computed(()=>{const option=options.find(item=>item.value===preference.value)!;return t(option.zh,option.en)})
let unsubscribe=()=>{}
function choose(value:ThemePreference){setThemePreference(value);menu.value?.removeAttribute('open')}
onMounted(()=>{unsubscribe=subscribeTheme(snapshot=>{preference.value=snapshot.preference;effective.value=snapshot.effective})})
onUnmounted(()=>unsubscribe())
</script>
<style scoped>
.theme-selector{position:relative;color:#fff}.theme-selector>summary{display:grid;place-items:center;box-sizing:border-box;min-width:24px;min-height:24px;padding:0;border:0;border-radius:5px;color:inherit;cursor:pointer;list-style:none}.theme-selector>summary::-webkit-details-marker{display:none}.theme-selector>summary:hover{background:#ffffff20}.theme-selector svg{width:17px;height:17px;fill:none;stroke:currentColor;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round}.theme-menu{position:absolute;right:0;top:calc(100% + 7px);z-index:120;width:min(270px,calc(100vw - 32px));box-sizing:border-box;padding:8px;border:1px solid var(--c-line,#dce3eb);border-radius:9px;background:var(--c-panel,#fff);box-shadow:0 12px 32px #07121e38;color:var(--c-text,#253247);font:14px/1.45 Arial,"Microsoft YaHei",sans-serif}.theme-menu strong{display:block;padding:6px 9px 8px}.theme-menu button{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:0 10px;width:100%;min-height:44px;box-sizing:border-box;padding:7px 9px;border:0;border-radius:6px;background:transparent;color:inherit;font:inherit;text-align:left;cursor:pointer}.theme-menu button:hover,.theme-menu button:focus-visible{background:var(--c-hover,#edf2f8)}.theme-menu button.selected{background:var(--home-option-selected,#f0f5ff);color:var(--c-accent,#155dfc)}.theme-menu small{grid-column:1;color:var(--c-muted,#5b687a);font-size:11px}.theme-menu i{grid-column:2;grid-row:1/3;align-self:center;font-style:normal}.theme-menu p{margin:7px 9px 3px;color:var(--c-muted,#5b687a);font-size:11px}@media(pointer:coarse){.theme-selector>summary{min-width:44px;min-height:44px}}
</style>
