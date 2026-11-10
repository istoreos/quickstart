import {cardRuntimePlugin} from './cardRuntimePlugin'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from "path";

function rename() {
	return {
		name: 'rename', generateBundle: function (options, bundle, isWrite) {
			for (var key in bundle) {
				if (key.indexOf('?') != -1) {
					var newkey = key.substring(0, key.indexOf('?'))
					bundle[key].fileName = newkey;
					console.log('rename: ', key, newkey)
				}
			}
		}
	};
};
// https://vitejs.dev/config/
export default defineConfig({
	base: "/",
	server: {
		host: "0.0.0.0",
		port: 8301,
		proxy: {
			"/cgi-bin/luci/istore/": {
				target: "http://127.0.0.1:8300"
			}
		},
		hmr: {
			overlay: false,
			host: "127.0.0.1",
			timeout: 0
		},
	},
	resolve: {
		alias: [
			{
				find: '/@',
				replacement: path.resolve(__dirname, 'src')
			},
		],
	},
	plugins: [
		cardRuntimePlugin(),vue(),
	],
	css: {
		preprocessorOptions: {
			scss: {
				charset: false,
				additionalData: `
					@import ${JSON.stringify(path.resolve(__dirname, 'src/style/theme.scss'))};
				`,
			}
		}
	},
	build: {
		target: "es2015",
		outDir: "dist",
		assetsDir: "",
		cssCodeSplit: false,
		assetsInlineLimit: 0,
		chunkSizeWarningLimit: 4096,
		rollupOptions: {
            preserveEntrySignatures:"strict",
			input:path.resolve(__dirname,"index.html"),
			output: {
				// Async modules must never import the HTML bootstrap entry: LuCI adds
				// its own ?v to index.js; a reverse import with Rollup’s different ?v would
				// execute the app a second time. Keep shared app code in its own chunk.
				manualChunks: (id) => id.includes('/node_modules/') ? 'vendor' : id.includes('/src/')&&!id.endsWith('/src/main.ts')&&!id.endsWith('/cards/installClient.ts')&&!id.includes('/cards/FastnetInstallDialog.vue') ? 'app' : undefined,
				entryFileNames: `luci-static/quickstart/[name].js`,
				chunkFileNames: `luci-static/quickstart/[name].js?v=[hash]`,
				assetFileNames: `luci-static/quickstart/[name].[ext]`
			},
			plugins: [rename()],
			external: [
			]
		}
	},
})
