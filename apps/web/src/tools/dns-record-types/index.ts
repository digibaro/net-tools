import { Tool as ToolIcon } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dns-record-types.title'),
  path: '/dns-record-types',
  description: translate('tools.dns-record-types.description'),
  keywords: ['dns', 'record', 'types'],
  component: () => import('./dns-record-types.vue'),
  icon: ToolIcon,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
