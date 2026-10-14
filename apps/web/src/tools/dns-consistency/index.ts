import { Sitemap } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dns-consistency.title'),
  path: '/dns-consistency',
  description: translate('tools.dns-consistency.description'),
  keywords: ['soa', 'serial', 'ns', 'nameserver', 'authoritative', 'consistency', 'lame delegation', 'zone', 'dns'],
  component: () => import('./dns-consistency.vue'),
  icon: Sitemap,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'dns.consistency',
})
