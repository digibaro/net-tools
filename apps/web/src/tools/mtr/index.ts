import { Activity } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.mtr.title'),
  path: '/mtr',
  description: translate('tools.mtr.description'),
  keywords: ['mtr', 'my traceroute', 'packet loss', 'latency', 'hops', 'path', 'jitter'],
  component: () => import('./mtr.vue'),
  icon: Activity,
  createdAt: new Date('2026-10-13'),
  runMode: 'hybrid',
  capability: 'net.mtr',
})
