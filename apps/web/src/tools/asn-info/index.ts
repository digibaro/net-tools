import { Sitemap } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.asn-info.title'),
  path: '/asn-info',
  description: translate('tools.asn-info.description'),
  keywords: ['asn', 'autonomous system', 'as number', 'prefixes', 'upstream', 'peers', 'bgp', 'ripestat', 'holder'],
  component: () => import('./asn-info.vue'),
  icon: Sitemap,
  createdAt: new Date('2026-10-14'),
  runMode: 'public-api',
  capability: 'bgp.asn',
})
