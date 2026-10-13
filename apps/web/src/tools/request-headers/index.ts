import { Api } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.request-headers.title'),
  path: '/request-headers',
  description: translate('tools.request-headers.description'),
  keywords: ['headers', 'request', 'http', 'echo', 'proxy', 'via', 'x-forwarded-for', 'cdn', 'middlebox'],
  component: () => import('./request-headers.vue'),
  icon: Api,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  requiresApi: true,
})
