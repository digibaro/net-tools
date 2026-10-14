import { CalendarTime } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.domain-expiry.title'),
  path: '/domain-expiry',
  description: translate('tools.domain-expiry.description'),
  keywords: ['domain expiry', 'expiration', 'renewal', 'registration date', 'rdap', 'whois', 'registrar'],
  component: () => import('./domain-expiry.vue'),
  icon: CalendarTime,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'reg.expiry',
})
