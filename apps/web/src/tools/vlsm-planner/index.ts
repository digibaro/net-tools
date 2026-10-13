import { LayoutGrid } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.vlsm-planner.title'),
  path: '/vlsm-planner',
  description: translate('tools.vlsm-planner.description'),
  keywords: ['vlsm', 'variable length subnet mask', 'subnet planner', 'address plan', 'hosts', 'ipv4', 'ipv6', 'cidr'],
  component: () => import('./vlsm-planner.vue'),
  icon: LayoutGrid,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
