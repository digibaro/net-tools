import { ArrowBackUp } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ptr-coverage.title'),
  path: '/ptr-coverage',
  description: translate('tools.ptr-coverage.description'),
  keywords: ['ptr', 'reverse dns', 'rdns', 'fcrdns', 'forward confirmed', 'in-addr.arpa', 'ip6.arpa', 'reverse zone', 'mail server'],
  component: () => import('./ptr-coverage.vue'),
  icon: ArrowBackUp,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'dns.ptrscan',
})
