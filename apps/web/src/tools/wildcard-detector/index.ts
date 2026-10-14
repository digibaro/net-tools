import { Asterisk } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.wildcard-detector.title'),
  path: '/wildcard-detector',
  description: translate('tools.wildcard-detector.description'),
  keywords: ['wildcard dns', '*.domain', 'catch-all', 'subdomain', 'random label', 'dns'],
  component: () => import('./wildcard-detector.vue'),
  icon: Asterisk,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'dns.wildcard',
})
