import { Fingerprint } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dkim-checker.title'),
  path: '/dkim-checker',
  description: translate('tools.dkim-checker.description'),
  keywords: ['dkim', 'domainkeys', 'selector', 'signature', 'public key', 'email', 'txt'],
  component: () => import('./dkim-checker.vue'),
  icon: Fingerprint,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'email.dkim',
})
