import { ArrowsJoin } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.common-ixps.title'),
  path: '/common-ixps',
  description: translate('tools.common-ixps.description'),
  keywords: ['ixp', 'internet exchange', 'peering', 'common ix', 'facility', 'colocation', 'peeringdb', 'where to peer'],
  component: () => import('./common-ixps.vue'),
  icon: ArrowsJoin,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'bgp.common',
})
