import { Browser } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.user-agent-parser.title'),
  path: '/user-agent-parser',
  description: translate('tools.user-agent-parser.description'),
  keywords: ['user agent', 'ua', 'browser', 'os', 'device', 'parse', 'client hints'],
  component: () => import('./user-agent-parser.vue'),
  icon: Browser,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
