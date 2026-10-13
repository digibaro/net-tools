import { ListNumbers } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.http-status-codes.title'),
  path: '/http-status-codes',
  description: translate('tools.http-status-codes.description'),
  keywords: ['http', 'status', 'code', 'response', '404', '500', 'redirect', 'rfc 9110', 'reference'],
  component: () => import('./http-status-codes.vue'),
  icon: ListNumbers,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
