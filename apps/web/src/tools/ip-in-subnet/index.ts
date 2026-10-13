import { ListSearch } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ip-in-subnet.title'),
  path: '/ip-in-subnet',
  description: translate('tools.ip-in-subnet.description'),
  keywords: ['ip', 'in', 'subnet', 'contains', 'match', 'cidr', 'prefix list', 'acl', 'check'],
  component: () => import('./ip-in-subnet.vue'),
  icon: ListSearch,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
