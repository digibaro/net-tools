import { Affiliate } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.asn-neighbour-graph.title'),
  path: '/asn-neighbour-graph',
  description: translate('tools.asn-neighbour-graph.description'),
  keywords: ['as graph', 'upstream', 'downstream', 'transit', 'peers', 'neighbours', 'neighbors', 'customer cone', 'topology'],
  component: () => import('./asn-neighbour-graph.vue'),
  icon: Affiliate,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'bgp.graph',
})
