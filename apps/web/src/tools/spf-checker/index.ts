import { Mail } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.spf-checker.title'),
  path: '/spf-checker',
  description: translate('tools.spf-checker.description'),
  keywords: ['spf', 'sender policy framework', 'include', 'dns lookups', 'permerror', 'email', 'txt'],
  component: () => import('./spf-checker.vue'),
  icon: Mail,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'email.spf',
})
