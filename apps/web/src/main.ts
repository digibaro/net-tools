import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import { i18nPlugin } from './plugins/i18n.plugin'
import { router } from './router'
import 'virtual:uno.css'

createApp(App)
  .use(createPinia())
  .use(i18nPlugin)
  .use(router)
  .mount('#app')
