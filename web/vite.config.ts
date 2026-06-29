import { defineConfig, splitVendorChunkPlugin } from 'vite'
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
		vue(),
		splitVendorChunkPlugin(),
	],
	css: {
		preprocessorOptions: {
			scss: {
				charset: false,
				additionalData: `
					@import "./src/style/theme.scss";
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
			output: {
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
