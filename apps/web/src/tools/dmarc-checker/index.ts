import { ShieldCheck } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dmarc-checker.title'),
  path: '/dmarc-checker',
  description: translate('tools.dmarc-checker.description'),
  keywords: ['dmarc', 'policy', 'rua', 'ruf', 'alignment', 'quarantine', 'reject', 'email', 'spoofing'],
  component: () => import('./dmarc-checker.vue'),
  icon: ShieldCheck,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'email.dmarc',
})
