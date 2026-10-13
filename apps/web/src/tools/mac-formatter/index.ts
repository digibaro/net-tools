import { Hash } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.mac-formatter.title'),
  path: '/mac-formatter',
  description: translate('tools.mac-formatter.description'),
  keywords: ['mac', 'format', 'normalize', 'cisco', 'colon', 'hyphen', 'random', 'generator', 'locally administered'],
  component: () => import('./mac-formatter.vue'),
  icon: Hash,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
