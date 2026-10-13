import { Activity } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.browser-latency.title'),
  path: '/browser-latency',
  description: translate('tools.browser-latency.description'),
  keywords: ['latency', 'rtt', 'jitter', 'browser', 'http ping', 'round trip', 'connection quality', 'packet loss'],
  component: () => import('./browser-latency.vue'),
  icon: Activity,
  createdAt: new Date('2026-10-13'),
  runMode: 'hybrid',
  requiresApi: true,
})
