import { Sitemap } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ip-to-asn.title'),
  path: '/ip-to-asn',
  description: translate('tools.ip-to-asn.description'),
  keywords: ['asn', 'autonomous system', 'bgp', 'origin', 'prefix', 'rir', 'cymru', 'isp', 'network owner'],
  component: () => import('./ip-to-asn.vue'),
  icon: Sitemap,
  createdAt: new Date('2026-10-13'),
  runMode: 'hybrid',
  capability: 'bgp.ip2asn',
})
