import { Map2 } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.country-ip-blocks.title'),
  path: '/country-ip-blocks',
  description: translate('tools.country-ip-blocks.description'),
  keywords: ['country', 'geoblocking', 'geo block', 'country ip ranges', 'rir', 'delegated', 'prefix-list', 'ipset', 'address-list', 'ipv4', 'ipv6'],
  component: () => import('./country-ip-blocks.vue'),
  icon: Map2,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'ip.country',
})
