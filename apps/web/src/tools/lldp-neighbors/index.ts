import { Hierarchy } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.lldp-neighbors.title'),
  path: '/lldp-neighbors',
  description: translate('tools.lldp-neighbors.description'),
  keywords: ['lldp', 'cdp', 'switch port', 'neighbor', 'neighbour', 'which port', 'vlan', 'network topology'],
  component: () => import('./lldp-neighbors.vue'),
  icon: Hierarchy,
  createdAt: new Date('2026-10-15'),
  runMode: 'server',
  capability: 'lan.lldp',
  requiresLan: true,
})
