import { Route } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.traceroute.title'),
  path: '/traceroute',
  description: translate('tools.traceroute.description'),
  keywords: ['traceroute', 'tracert', 'path', 'hops', 'route', 'icmp', 'udp', 'tcp'],
  component: () => import('./traceroute.vue'),
  icon: Route,
  createdAt: new Date('2026-10-13'),
  runMode: 'hybrid',
  capability: 'net.traceroute',
})
