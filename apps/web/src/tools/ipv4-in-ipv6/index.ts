import { ArrowsLeftRight } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ipv4-in-ipv6.title'),
  path: '/ipv4-in-ipv6',
  description: translate('tools.ipv4-in-ipv6.description'),
  keywords: ['ipv4-mapped', '::ffff', '6to4', '2002', 'nat64', '64:ff9b', 'dns64', 'teredo', 'rfc 6052', 'transition'],
  component: () => import('./ipv4-in-ipv6.vue'),
  icon: ArrowsLeftRight,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
