import { Filter } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.wildcard-mask-calculator.title'),
  path: '/wildcard-mask-calculator',
  description: translate('tools.wildcard-mask-calculator.description'),
  keywords: ['wildcard', 'inverse mask', 'acl', 'cisco', 'netmask', 'subnet mask', 'ospf', 'prefix'],
  component: () => import('./wildcard-mask-calculator.vue'),
  icon: Filter,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
