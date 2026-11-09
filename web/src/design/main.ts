// Install the isolated transport before importing components that request data.
import './fastnetModel'
import { installFixtures } from './fixtures'
installFixtures()
window.vue_base = '/'
window.vue_lang = new URLSearchParams(location.search).get('lang') || 'zh-cn'
window.quickstart_features = ['dockerd', 'sandbox', 'ota', 'ttyd']
window.quickstart_configs = { update: { disable: true }, deviceInventoryV2: true, ttyd: {port:7681,ssl:false} }
void import('./mount').then(({ mount }) => mount())
