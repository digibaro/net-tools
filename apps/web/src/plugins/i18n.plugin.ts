import type { App } from 'vue'
import messages from '@intlify/unplugin-vue-i18n/messages'
import { createI18n } from 'vue-i18n'

const LOCALE_KEY = 'nettools:locale'
export const availableLocales = Object.keys(messages ?? {}).sort()

function initialLocale(): string {
  try {
    const stored = localStorage.getItem(LOCALE_KEY)
    if (stored && availableLocales.includes(stored))
      return stored
  }
  catch {}
  return 'en'
}

export const i18n = createI18n({
  legacy: false,
  locale: typeof localStorage === 'undefined' ? 'en' : initialLocale(),
  fallbackLocale: 'en',
  messages,
})

if (typeof document !== 'undefined')
  document.documentElement.lang = i18n.global.locale.value

export function setLocale(locale: string) {
  if (!availableLocales.includes(locale))
    return
  i18n.global.locale.value = locale
  document.documentElement.lang = locale
  try {
    localStorage.setItem(LOCALE_KEY, locale)
  }
  catch {}
}

export const i18nPlugin = {
  install(app: App) {
    app.use(i18n)
  },
}

/** Translate outside of components (e.g. tool definitions). */
export function translate(key: string, named?: Record<string, unknown>): string {
  return named ? i18n.global.t(key, named) : i18n.global.t(key)
}
