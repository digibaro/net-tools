import { ChartLine } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.bgp-route-history.title'),
  path: '/bgp-route-history',
  description: translate('tools.bgp-route-history.description'),
  keywords: ['bgp history', 'route visibility', 'origin as', 'hijack', 'moas', 'ris', 'routing history', 'prefix visibility'],
  component: () => import('./bgp-route-history.vue'),
  icon: ChartLine,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'bgp.history',
})
