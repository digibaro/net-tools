import { LayoutRows } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.latency-matrix.title'),
  path: '/latency-matrix',
  description: translate('tools.latency-matrix.description'),
  keywords: ['latency', 'compare', 'multi ping', 'parallel ping', 'fastest server', 'rtt', 'matrix'],
  component: () => import('./latency-matrix.vue'),
  icon: LayoutRows,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'net.latmatrix',
})
