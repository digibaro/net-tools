import { Route } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.prefix-status.title'),
  path: '/prefix-status',
  description: translate('tools.prefix-status.description'),
  keywords: ['prefix', 'bgp', 'routing', 'visibility', 'origin', 'moas', 'hijack', 'ris', 'ripestat', 'announced'],
  component: () => import('./prefix-status.vue'),
  icon: Route,
  createdAt: new Date('2026-10-14'),
  runMode: 'public-api',
  capability: 'bgp.prefix',
})
