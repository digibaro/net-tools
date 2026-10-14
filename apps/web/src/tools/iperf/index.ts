import { ArrowsDownUp } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.iperf.title'),
  path: '/iperf',
  description: translate('tools.iperf.description'),
  keywords: ['iperf', 'iperf3', 'bandwidth', 'throughput', 'jitter', 'packet loss', 'speed'],
  component: () => import('./iperf.vue'),
  icon: ArrowsDownUp,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'net.iperf',
})
