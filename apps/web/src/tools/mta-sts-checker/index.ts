import { MailOpened } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.mta-sts-checker.title'),
  path: '/mta-sts-checker',
  description: translate('tools.mta-sts-checker.description'),
  keywords: ['mta-sts', 'tls-rpt', 'bimi', 'smtp tls', 'email security', 'vmc', 'brand logo', 'rfc 8461'],
  component: () => import('./mta-sts-checker.vue'),
  icon: MailOpened,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'email.mtasts',
})
