import { MailForward } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.open-relay-test.title'),
  path: '/open-relay-test',
  description: translate('tools.open-relay-test.description'),
  keywords: ['open relay', 'smtp relay', 'relay test', 'spam', 'mail server', 'rcpt to'],
  component: () => import('./open-relay-test.vue'),
  icon: MailForward,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'email.relay',
})
