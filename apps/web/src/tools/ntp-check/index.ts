import { Clock } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ntp-check.title'),
  path: '/ntp-check',
  description: translate('tools.ntp-check.description'),
  keywords: ['ntp', 'time', 'clock', 'offset', 'stratum', 'refid', 'sntp', 'time server'],
  component: () => import('./ntp-check.vue'),
  icon: Clock,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'net.ntp',
})
