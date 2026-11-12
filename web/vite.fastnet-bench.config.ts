// Explicit performance-fixture build; excluded from normal product builds.
import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'node:path'
import {cardRuntimePlugin} from './cardRuntimePlugin'
export default defineConfig({root:__dirname,plugins:[cardRuntimePlugin(),vue()],resolve:{alias:[{find:'/@',replacement:path.resolve(__dirname,'src')}]},build:{target:'es2022',emptyOutDir:true,outDir:path.join(process.env.BUILD_CACHE_WORK_DIR!,'fastnet-bench'),rollupOptions:{input:path.resolve(__dirname,'../scripts/ops/fastnet-bench.ts'),output:{entryFileNames:'luci-static/quickstart/bench/bench.js',assetFileNames:'luci-static/quickstart/bench/[name].[ext]'}}}})
