import { Terminal2 } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.router-snippets.title'),
  path: '/router-snippets',
  description: translate('tools.router-snippets.description'),
  keywords: ['prefix-list', 'acl', 'access-list', 'cisco', 'juniper', 'junos', 'arista', 'bird', 'mikrotik', 'route filter', 'config'],
  component: () => import('./router-snippets.vue'),
  icon: Terminal2,
  createdAt: new Date('2026-10-14'),
  runMode: 'browser',
})
