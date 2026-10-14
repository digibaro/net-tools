import { Database } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dns-record-types.title'),
  path: '/dns-record-types',
  description: translate('tools.dns-record-types.description'),
  keywords: ['dns', 'record', 'rr', 'type', 'mx', 'txt', 'caa', 'srv', 'https', 'svcb', 'dnssec', 'zone file', 'reference'],
  component: () => import('./dns-record-types.vue'),
  icon: Database,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
