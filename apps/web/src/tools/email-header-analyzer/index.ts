import { MailForward } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.email-header-analyzer.title'),
  path: '/email-header-analyzer',
  description: translate('tools.email-header-analyzer.description'),
  keywords: ['email headers', 'received', 'message headers', 'delay', 'hops', 'authentication-results', 'spf', 'dkim', 'dmarc', 'trace email'],
  component: () => import('./email-header-analyzer.vue'),
  icon: MailForward,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
