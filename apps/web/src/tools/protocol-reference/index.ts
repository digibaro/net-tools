import { Tool as ToolIcon } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.protocol-reference.title'),
  path: '/protocol-reference',
  description: translate('tools.protocol-reference.description'),
  keywords: ['protocol', 'reference'],
  component: () => import('./protocol-reference.vue'),
  icon: ToolIcon,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
