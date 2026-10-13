import { Clock } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.http-timing.title'),
  path: '/http-timing',
  description: translate('tools.http-timing.description'),
  keywords: ['http', 'timing', 'ttfb', 'latency', 'performance', 'dns', 'tls', 'connect', 'curl -w', 'waterfall'],
  component: () => import('./http-timing.vue'),
  icon: Clock,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'web.timing',
})
