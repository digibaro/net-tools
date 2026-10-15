import { ArrowsLeftRight } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.nat64-dns64-check.title'),
  path: '/nat64-dns64-check',
  description: translate('tools.nat64-dns64-check.description'),
  keywords: ['nat64', 'dns64', 'ipv4only.arpa', 'rfc 7050', 'rfc 6052', '64:ff9b', 'ipv6-only', '464xlat', 'clat'],
  component: () => import('./nat64-dns64-check.vue'),
  icon: ArrowsLeftRight,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'dns.dns64',
})
