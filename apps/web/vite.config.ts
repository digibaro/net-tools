import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import process from 'node:process'
import { fileURLToPath, URL } from 'node:url'
import VueI18n from '@intlify/unplugin-vue-i18n/vite'
import vue from '@vitejs/plugin-vue'
import UnoCSS from 'unocss/vite'
import { VitePWA } from 'vite-plugin-pwa'
import { defineConfig } from 'vitest/config'
import { cspScriptHashes } from './build/csp-hashes.ts'

const apiProxy = process.env.VITE_API_PROXY ?? 'http://localhost:8081'

/**
 * Latest release ("0.4.0@2026-10-14"): VITE_LATEST_RELEASE (Docker/CI builds) or the newest v* tag in
 * git; without either nothing is badged "new".
 */
function latestRelease(): { version: string, date: string } {
  const env = process.env.VITE_LATEST_RELEASE
  const m = env ? /^v?(\d+\.\d+\.\d[\w.-]*)@(\d{4}-\d{2}-\d{2})$/.exec(env) : null
  if (m)
    return { version: m[1]!, date: m[2]! }
  try {
    const tag = execFileSync('git', ['describe', '--tags', '--abbrev=0', '--match', 'v[0-9]*'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] }).trim()
    const date = execFileSync('git', ['log', '-1', '--format=%cs', tag], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] }).trim()
    return { version: tag.replace(/^v/, ''), date }
  }
  catch {
    return { version: '0.0.0', date: '9999-12-31' }
  }
}

function appVersion(): string {
  const v = process.env.VITE_APP_VERSION
  if (v && v !== 'dev')
    return v.replace(/^v/, '')
  const pkg = JSON.parse(readFileSync(new URL('./package.json', import.meta.url), 'utf8')) as { version: string }
  return `${pkg.version}-dev`
}

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
    // Release builds pass the tag; local and dev builds show the package version with -dev.
    __APP_VERSION__: JSON.stringify(appVersion()),
    __LATEST_RELEASE__: JSON.stringify(latestRelease()),
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
