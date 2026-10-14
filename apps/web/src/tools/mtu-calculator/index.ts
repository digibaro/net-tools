import { Ruler } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.mtu-calculator.title'),
  path: '/mtu-calculator',
  description: translate('tools.mtu-calculator.description'),
  keywords: ['mtu', 'mss', 'overhead', 'pppoe', 'vlan', 'gre', 'ipsec', 'wireguard', 'vxlan', 'geneve', 'tunnel', 'mpls', 'clamp'],
  component: () => import('./mtu-calculator.vue'),
  icon: Ruler,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
