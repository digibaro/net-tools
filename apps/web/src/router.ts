import type { Tool } from './tools/tool'
import { createRouter, createWebHistory } from 'vue-router'
import { toolsByCategory } from './tools'

declare module 'vue-router' {
  interface RouteMeta {
    tool?: Tool
    titleKey?: string
  }
}

const toolRoutes = toolsByCategory.flatMap(c => c.components).map(tool => ({
  path: tool.path,
  name: tool.path.slice(1),
  component: tool.component,
  meta: { tool },
}))

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', name: 'home', component: () => import('./pages/Home.page.vue') },
    { path: '/about', name: 'about', component: () => import('./pages/About.page.vue'), meta: { titleKey: 'nav.about' } },
    { path: '/settings', name: 'settings', component: () => import('./pages/Settings.page.vue'), meta: { titleKey: 'nav.settings' } },
    ...toolRoutes,
    { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('./pages/NotFound.page.vue'), meta: { titleKey: 'notFound.title' } },
  ],
  scrollBehavior: () => ({ top: 0 }),
})
