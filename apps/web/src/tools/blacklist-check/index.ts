import { Forbid } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.blacklist-check.title'),
  path: '/blacklist-check',
  description: translate('tools.blacklist-check.description'),
  keywords: ['blacklist', 'blocklist', 'dnsbl', 'rbl', 'spamhaus', 'spamcop', 'barracuda', 'email deliverability', 'listed'],
  component: () => import('./blacklist-check.vue'),
  icon: Forbid,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'email.dnsbl',
})
