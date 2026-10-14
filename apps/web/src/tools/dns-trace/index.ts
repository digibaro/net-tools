import { Sitemap } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dns-trace.title'),
  path: '/dns-trace',
  description: translate('tools.dns-trace.description'),
  keywords: ['dns', 'trace', 'dig +trace', 'delegation', 'iterative', 'root', 'referral', 'glue'],
  component: () => import('./dns-trace.vue'),
  icon: Sitemap,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'dns.trace',
})
