import { Table } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.subnet-cheat-sheet.title'),
  path: '/subnet-cheat-sheet',
  description: translate('tools.subnet-cheat-sheet.description'),
  keywords: ['subnet', 'cheat sheet', 'cidr', 'netmask', 'table', 'prefix', 'hosts', 'ipv6', 'reference'],
  component: () => import('./subnet-cheat-sheet.vue'),
  icon: Table,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
