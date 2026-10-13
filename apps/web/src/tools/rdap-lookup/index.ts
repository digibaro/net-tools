import { Database } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.rdap-lookup.title'),
  path: '/rdap-lookup',
  description: translate('tools.rdap-lookup.description'),
  keywords: ['rdap', 'whois', 'registration', 'registrar', 'ip owner', 'asn', 'abuse contact', 'rir'],
  component: () => import('./rdap-lookup.vue'),
  icon: Database,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'reg.rdap',
})
