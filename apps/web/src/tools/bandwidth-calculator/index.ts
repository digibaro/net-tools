import { Clock } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.bandwidth-calculator.title'),
  path: '/bandwidth-calculator',
  description: translate('tools.bandwidth-calculator.description'),
  keywords: ['bandwidth', 'transfer time', 'download time', 'throughput', 'speed', 'mbps', 'overhead', 'calculator'],
  component: () => import('./bandwidth-calculator.vue'),
  icon: Clock,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
