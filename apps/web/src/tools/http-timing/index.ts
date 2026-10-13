import { Tool as ToolIcon } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.http-timing.title'),
  path: '/http-timing',
  description: translate('tools.http-timing.description'),
  keywords: ['http', 'timing'],
  component: () => import('./http-timing.vue'),
  icon: ToolIcon,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
