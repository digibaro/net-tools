import { Plug } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.eui64-calculator.title'),
  path: '/eui64-calculator',
  description: translate('tools.eui64-calculator.description'),
  keywords: ['eui-64', 'eui64', 'link-local', 'fe80', 'slaac', 'interface id', 'mac', 'ipv6'],
  component: () => import('./eui64-calculator.vue'),
  icon: Plug,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
