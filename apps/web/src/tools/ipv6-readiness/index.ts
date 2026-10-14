import { Number6 } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ipv6-readiness.title'),
  path: '/ipv6-readiness',
  description: translate('tools.ipv6-readiness.description'),
  keywords: ['ipv6', 'aaaa', 'ipv6 ready', 'dual stack', 'ipv6 test', 'name server ipv6', 'mail ipv6'],
  component: () => import('./ipv6-readiness.vue'),
  icon: Number6,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'net.ipv6ready',
})
