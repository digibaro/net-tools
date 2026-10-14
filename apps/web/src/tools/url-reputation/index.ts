import { ShieldLock } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.url-reputation.title'),
  path: '/url-reputation',
  description: translate('tools.url-reputation.description'),
  keywords: ['url reputation', 'phishing', 'malware', 'safe browsing', 'virustotal', 'urlscan', 'domain reputation', 'blocklist'],
  component: () => import('./url-reputation.vue'),
  icon: ShieldLock,
  createdAt: new Date('2026-10-14'),
  runMode: 'public-api',
  capability: 'rep.url',
})
