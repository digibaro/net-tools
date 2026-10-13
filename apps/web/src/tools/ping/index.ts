import { Stethoscope } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ping.title'),
  path: '/ping',
  description: translate('tools.ping.description'),
  keywords: ['ping', 'icmp', 'echo', 'latency', 'rtt', 'packet loss', 'jitter', 'reachability'],
  component: () => import('./ping.vue'),
  icon: Stethoscope,
  createdAt: new Date('2026-10-13'),
  runMode: 'hybrid',
  capability: 'net.ping',
})
