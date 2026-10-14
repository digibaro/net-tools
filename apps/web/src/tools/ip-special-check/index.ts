import { ShieldCheck } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ip-special-check.title'),
  path: '/ip-special-check',
  description: translate('tools.ip-special-check.description'),
  keywords: ['bogon', 'special', 'reserved', 'private', 'rfc 1918', 'cgnat', 'loopback', 'documentation', 'multicast', 'martian', 'iana'],
  component: () => import('./ip-special-check.vue'),
  icon: ShieldCheck,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
