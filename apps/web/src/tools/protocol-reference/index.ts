import { Table } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.protocol-reference.title'),
  path: '/protocol-reference',
  description: translate('tools.protocol-reference.description'),
  keywords: ['protocol number', 'ip protocol', 'icmp', 'icmpv6', 'type', 'code', 'iana', 'next header'],
  component: () => import('./protocol-reference.vue'),
  icon: Table,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
