import { ArrowBackUp } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.reverse-dns.title'),
  path: '/reverse-dns',
  description: translate('tools.reverse-dns.description'),
  keywords: ['reverse dns', 'ptr', 'rdns', 'fcrdns', 'forward confirmed', 'hostname', 'ip to hostname'],
  component: () => import('./reverse-dns.vue'),
  icon: ArrowBackUp,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'dns.reverse',
})
