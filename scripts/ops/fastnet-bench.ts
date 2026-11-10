// Test entry only. The shell, runtime, styles and business ESM are production code.
import {createApp,defineComponent,h,ref} from 'vue'
import CardShell from '../../web/src/home/cards/CardShell.vue'
import {cardManifest} from '../../web/src/home/cards/contract'
import {initializeTheme} from '../../web/src/utils/theme'
import '../../web/src/home/homeSemanticColors.css'
initializeTheme()
const raw=await fetch('/cgi-bin/luci/istore/home/v1/cards/').then(r=>r.json())
const manifest=cardManifest(raw.cards.find((c:any)=>c.id==='plugin.fastnet'))
window.vue_lang='zh-cn'
createApp(defineComponent({setup(){const visible=ref(false);return()=>h('main',{class:'home-semantic-colors ordered-home',style:'padding:24px;background:var(--c-bg);color:var(--c-text)'},[h('button',{'data-toggle-card':'',onClick:()=>visible.value=!visible.value},visible.value?'Unmount':'Mount'),h('div',{style:'height:340px;width:520px;max-width:100%'},visible.value?[h(CardShell,{manifest,form:'standard'})]:[])])}})).mount('#app')
