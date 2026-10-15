import { MailForward } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.smtp-banner-check.title'),
  path: '/smtp-banner-check',
  description: translate('tools.smtp-banner-check.description'),
  keywords: ['smtp banner', 'helo', 'ehlo', 'fcrdns', 'ptr', 'reverse dns', 'mail server', 'deliverability', 'spam'],
  component: () => import('./smtp-banner-check.vue'),
  icon: MailForward,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'email.banner',
})
