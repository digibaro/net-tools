import { CalendarTime } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.tls-expiry.title'),
  path: '/tls-expiry',
  description: translate('tools.tls-expiry.description'),
  keywords: ['certificate expiry', 'expiration', 'ssl expiry', 'tls', 'days left', 'bulk', 'renewal'],
  component: () => import('./tls-expiry.vue'),
  icon: CalendarTime,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'tls.expiry',
})
