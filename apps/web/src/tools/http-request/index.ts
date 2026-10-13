import { Api } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.http-request.title'),
  path: '/http-request',
  description: translate('tools.http-request.description'),
  keywords: ['http', 'request', 'curl', 'headers', 'response', 'status', 'api', 'inspector', 'rest'],
  component: () => import('./http-request.vue'),
  icon: Api,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'web.request',
})
