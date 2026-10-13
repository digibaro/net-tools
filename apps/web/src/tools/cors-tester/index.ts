import { Tool as ToolIcon } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.cors-tester.title'),
  path: '/cors-tester',
  description: translate('tools.cors-tester.description'),
  keywords: ['cors', 'tester'],
  component: () => import('./cors-tester.vue'),
  icon: ToolIcon,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
