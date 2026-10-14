import { LockAccess } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.starttls-check.title'),
  path: '/starttls-check',
  description: translate('tools.starttls-check.description'),
  keywords: ['starttls', 'smtp tls', 'imap', 'pop3', 'ftps', 'xmpp', 'mail server certificate', 'opportunistic tls'],
  component: () => import('./starttls-check.vue'),
  icon: LockAccess,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'tls.inspect',
})
