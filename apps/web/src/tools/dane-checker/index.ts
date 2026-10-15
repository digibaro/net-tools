import { ShieldLock } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dane-checker.title'),
  path: '/dane-checker',
  description: translate('tools.dane-checker.description'),
  keywords: ['dane', 'tlsa', 'rfc 6698', 'rfc 7672', 'smtp dane', 'dnssec', 'mail tls', 'certificate pinning', '_25._tcp'],
  component: () => import('./dane-checker.vue'),
  icon: ShieldLock,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'tls.dane',
})
