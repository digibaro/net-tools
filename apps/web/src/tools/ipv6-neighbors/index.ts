import { Hierarchy2 } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ipv6-neighbors.title'),
  path: '/ipv6-neighbors',
  description: translate('tools.ipv6-neighbors.description'),
  keywords: ['ndp', 'ipv6 neighbors', 'neighbour discovery', 'router advertisement', 'ra', 'slaac', 'rdnss', 'ipv6 lan scan'],
  component: () => import('./ipv6-neighbors.vue'),
  icon: Hierarchy2,
  createdAt: new Date('2026-10-15'),
  runMode: 'server',
  capability: 'lan.ndp',
  requiresLan: true,
})
