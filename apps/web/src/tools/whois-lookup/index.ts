import { FileSearch } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.whois-lookup.title'),
  path: '/whois-lookup',
  description: translate('tools.whois-lookup.description'),
  keywords: ['whois', 'registrar', 'domain owner', 'port 43', 'ip owner', 'asn', 'referral', 'iana'],
  component: () => import('./whois-lookup.vue'),
  icon: FileSearch,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'reg.whois',
})
