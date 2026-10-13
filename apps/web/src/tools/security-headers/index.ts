import { Tool as ToolIcon } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.security-headers.title'),
  path: '/security-headers',
  description: translate('tools.security-headers.description'),
  keywords: ['security', 'headers'],
  component: () => import('./security-headers.vue'),
  icon: ToolIcon,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
