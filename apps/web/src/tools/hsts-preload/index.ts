import { ListCheck } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.hsts-preload.title'),
  path: '/hsts-preload',
  description: translate('tools.hsts-preload.description'),
  keywords: ['hsts', 'preload', 'hstspreload', 'strict-transport-security', 'https only', 'chromium preload list'],
  component: () => import('./hsts-preload.vue'),
  icon: ListCheck,
  createdAt: new Date('2026-10-14'),
  runMode: 'public-api',
  capability: 'web.hsts',
})
