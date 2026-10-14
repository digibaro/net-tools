import { Filter } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.irr-expansion.title'),
  path: '/irr-expansion',
  description: translate('tools.irr-expansion.description'),
  keywords: ['irr', 'as-set', 'route-set', 'bgpq4', 'bgpq3', 'prefix filter', 'peering', 'radb', 'route objects'],
  component: () => import('./irr-expansion.vue'),
  icon: Filter,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'bgp.irr',
})
