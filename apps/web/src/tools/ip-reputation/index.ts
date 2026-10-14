import { ShieldX } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ip-reputation.title'),
  path: '/ip-reputation',
  description: translate('tools.ip-reputation.description'),
  keywords: ['ip reputation', 'abuseipdb', 'abuse', 'blacklist', 'spam', 'brute force', 'malicious ip', 'threat'],
  component: () => import('./ip-reputation.vue'),
  icon: ShieldX,
  createdAt: new Date('2026-10-14'),
  runMode: 'public-api',
  capability: 'rep.abuseipdb',
})
