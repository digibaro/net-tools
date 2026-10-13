import { Plug } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.tcp-ping.title'),
  path: '/tcp-ping',
  description: translate('tools.tcp-ping.description'),
  keywords: ['tcp ping', 'tcping', 'connect time', 'port', 'latency', 'firewall', 'icmp blocked'],
  component: () => import('./tcp-ping.vue'),
  icon: Plug,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'net.tcping',
})
