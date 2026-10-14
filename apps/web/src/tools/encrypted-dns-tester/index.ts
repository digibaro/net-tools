import { ShieldCheck } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.encrypted-dns-tester.title'),
  path: '/encrypted-dns-tester',
  description: translate('tools.encrypted-dns-tester.description'),
  keywords: ['doh', 'dot', 'dns over https', 'dns over tls', 'encrypted dns', 'tls', 'latency', 'resolver'],
  component: () => import('./encrypted-dns-tester.vue'),
  icon: ShieldCheck,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'dns.encrypted',
})
