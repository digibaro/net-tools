import { Microscope } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ipv6-address-analyzer.title'),
  path: '/ipv6-address-analyzer',
  description: translate('tools.ipv6-address-analyzer.description'),
  keywords: ['ipv6', 'address type', 'scope', 'eui-64', 'interface identifier', 'privacy address', 'solicited-node', 'multicast', 'isatap', 'link-local'],
  component: () => import('./ipv6-address-analyzer.vue'),
  icon: Microscope,
  createdAt: new Date('2026-10-16'),
})
