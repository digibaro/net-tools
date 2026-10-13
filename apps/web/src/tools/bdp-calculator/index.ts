import { Gauge } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.bdp-calculator.title'),
  path: '/bdp-calculator',
  description: translate('tools.bdp-calculator.description'),
  keywords: ['bdp', 'bandwidth delay product', 'tcp window', 'rwnd', 'window scaling', 'rtt', 'throughput', 'mathis', 'long fat network'],
  component: () => import('./bdp-calculator.vue'),
  icon: Gauge,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
