import process from 'node:process'
import { fileURLToPath, URL } from 'node:url'
import VueI18n from '@intlify/unplugin-vue-i18n/vite'
import vue from '@vitejs/plugin-vue'
import UnoCSS from 'unocss/vite'
import { VitePWA } from 'vite-plugin-pwa'
import { defineConfig } from 'vitest/config'
import { cspScriptHashes } from './build/csp-hashes.ts'

const apiProxy = process.env.VITE_API_PROXY ?? 'http://localhost:8081'

export default defineConfig({
  plugins: [
    vue(),
    UnoCSS(),
    VueI18n({
      include: fileURLToPath(new URL('./locales/**', import.meta.url)),
      runtimeOnly: true,
      compositionOnly: true,
      fullInstall: false,
    }),
    VitePWA({
      registerType: 'autoUpdate',
      injectRegister: 'script-defer',
      manifest: {
        name: 'NET-TOOLS',
        short_name: 'NET-TOOLS',
        description: 'Self-hosted network tools and troubleshooting utilities',
        theme_color: '#18181c',
        background_color: '#ffffff',
        display: 'standalone',
        icons: [
          { src: '/favicon.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any' },
          { src: '/icon-maskable.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'maskable' },
        ],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,svg,json,woff2}'],
        maximumFileSizeToCacheInBytes: 3 * 1024 * 1024,
        navigateFallback: '/index.html',
        // API responses are live data: never serve them from the cache.
        navigateFallbackDenylist: [/^\/api\//],
      },
    }),
    cspScriptHashes(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  define: {
    __APP_VERSION__: JSON.stringify(process.env.VITE_APP_VERSION ?? 'dev'),
  },
  server: {
    proxy: {
      '/api': { target: apiProxy, changeOrigin: false },
    },
  },
  test: {
    include: ['src/**/*.test.ts', 'build/**/*.test.ts'],
    environment: 'node',
  },
})
