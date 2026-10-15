import { Table } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ipam-sheet.title'),
  path: '/ipam-sheet',
  description: translate('tools.ipam-sheet.description'),
  keywords: ['ipam', 'ip address plan', 'address plan', 'subnet plan', 'vlan', 'allocation', 'ipv4', 'ipv6', 'csv'],
  component: () => import('./ipam-sheet.vue'),
  icon: Table,
  createdAt: new Date('2026-10-16'),
  runMode: 'browser',
})
