import { Sitemap } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ipv6-subnet-calculator.title'),
  path: '/ipv6-subnet-calculator',
  description: translate('tools.ipv6-subnet-calculator.description'),
  keywords: ['ipv6', 'subnet', 'prefix', 'cidr', '/64', '/48', 'range', 'calculator'],
  component: () => import('./ipv6-subnet-calculator.vue'),
  icon: Sitemap,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
