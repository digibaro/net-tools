import { ArrowsMaximize } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ipv6-expand-compress.title'),
  path: '/ipv6-expand-compress',
  description: translate('tools.ipv6-expand-compress.description'),
  keywords: ['ipv6', 'expand', 'compress', 'rfc 5952', 'canonical', 'shorten', 'ip6.arpa', 'reverse', 'nibble'],
  component: () => import('./ipv6-expand-compress.vue'),
  icon: ArrowsMaximize,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
