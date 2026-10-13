import process from 'node:process'
import { fileURLToPath, URL } from 'node:url'
import VueI18n from '@intlify/unplugin-vue-i18n/vite'
import vue from '@vitejs/plugin-vue'
import UnoCSS from 'unocss/vite'
import { defineConfig } from 'vitest/config'

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
    include: ['src/**/*.test.ts'],
    environment: 'node',
  },
})
