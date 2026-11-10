import {transform} from 'esbuild'
import {readFile} from 'node:fs/promises'
import {createRequire} from 'node:module'
import type {Plugin} from 'vite'
const require=createRequire(import.meta.url)
export const cardRuntimeUrl='/luci-static/quickstart/card-runtime-v1.js'
// ABI v1 pins Vue 3.5.44. This standalone asset must not import hashed host chunks:
// a cached SDK from the previous host must resolve to the same Vue as the host.
export function cardRuntimePlugin():Plugin{
 return {name:'quickstart-shared-card-runtime',enforce:'pre',apply:'build',
  resolveId(source){if(source==='vue'||source===cardRuntimeUrl)return {id:cardRuntimeUrl,external:true}},
  async buildStart(){
   const vue=require('vue/package.json');if(vue.version!=='3.5.44')throw Error('card ABI v1 requires Vue 3.5.44')
   // Vue's pinned browser production distribution is self-contained. Reusing
   // its production flags also keeps SDK bytes independent of the host graph.
   const source=await readFile(require.resolve('vue/dist/vue.runtime.esm-browser.prod.js'),'utf8')
   const result=await transform(source+'\nexport const cardRuntimeABI="quickstart-card-v1";',{format:'esm',minify:true,target:'es2015',legalComments:'inline'})
   this.emitFile({type:'asset',fileName:cardRuntimeUrl.slice(1),source:result.code})
  },
 }
}
