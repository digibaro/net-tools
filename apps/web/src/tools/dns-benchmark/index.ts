import { Gauge } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dns-benchmark.title'),
  path: '/dns-benchmark',
  description: translate('tools.dns-benchmark.description'),
  keywords: ['dns benchmark', 'fastest dns', 'resolver speed', 'namebench', '1.1.1.1', '8.8.8.8', 'quad9', 'latency'],
  component: () => import('./dns-benchmark.vue'),
  icon: Gauge,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'dns.benchmark',
})
