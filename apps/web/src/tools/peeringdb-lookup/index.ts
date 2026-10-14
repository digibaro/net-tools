import { BuildingCommunity } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.peeringdb-lookup.title'),
  path: '/peeringdb-lookup',
  description: translate('tools.peeringdb-lookup.description'),
  keywords: ['peeringdb', 'peering', 'ixp', 'internet exchange', 'facility', 'colocation', 'peering policy', 'as-set', 'noc'],
  component: () => import('./peeringdb-lookup.vue'),
  icon: BuildingCommunity,
  createdAt: new Date('2026-10-14'),
  runMode: 'public-api',
  capability: 'bgp.peeringdb',
})
