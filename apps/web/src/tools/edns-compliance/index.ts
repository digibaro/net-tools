import { ListCheck } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.edns-compliance.title'),
  path: '/edns-compliance',
  description: translate('tools.edns-compliance.description'),
  keywords: ['edns', 'edns0', 'ednscomp', 'dns flag day', 'badvers', 'nsid', 'cookie', 'authoritative', 'nameserver compliance'],
  component: () => import('./edns-compliance.vue'),
  icon: ListCheck,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'dns.ednscomp',
})
