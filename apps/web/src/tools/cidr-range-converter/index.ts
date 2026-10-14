import { ArrowsLeftRight } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.cidr-range-converter.title'),
  path: '/cidr-range-converter',
  description: translate('tools.cidr-range-converter.description'),
  keywords: ['cidr', 'range', 'ip range', 'prefix', 'first', 'last', 'convert', 'ipv4', 'ipv6'],
  component: () => import('./cidr-range-converter.vue'),
  icon: ArrowsLeftRight,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
