import { ArrowsSplit2 } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.subnet-splitter.title'),
  path: '/subnet-splitter',
  description: translate('tools.subnet-splitter.description'),
  keywords: ['subnet', 'split', 'divide', 'subnetting', 'cidr', 'prefix', 'ipv4', 'ipv6'],
  component: () => import('./subnet-splitter.vue'),
  icon: ArrowsSplit2,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
