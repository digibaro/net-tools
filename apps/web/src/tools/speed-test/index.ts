import { Gauge } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.speed-test.title'),
  path: '/speed-test',
  description: translate('tools.speed-test.description'),
  keywords: ['speed test', 'bandwidth', 'download speed', 'upload speed', 'mbps', 'throughput', 'librespeed'],
  component: () => import('./speed-test.vue'),
  icon: Gauge,
  createdAt: new Date('2026-10-14'),
  runMode: 'hybrid',
  capability: 'net.speedtest',
})
