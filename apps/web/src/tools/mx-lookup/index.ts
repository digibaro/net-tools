import { Mail } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.mx-lookup.title'),
  path: '/mx-lookup',
  description: translate('tools.mx-lookup.description'),
  keywords: ['mx', 'mail server', 'smtp', 'starttls', 'port 25', 'banner', 'email', 'deliverability'],
  component: () => import('./mx-lookup.vue'),
  icon: Mail,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'email.mx',
})
