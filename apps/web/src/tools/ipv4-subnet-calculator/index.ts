import { Sitemap } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ipv4-subnet-calculator.title'),
  path: '/ipv4-subnet-calculator',
  description: translate('tools.ipv4-subnet-calculator.description'),
  keywords: ['ipv4', 'subnet', 'cidr', 'netmask', 'mask', 'broadcast', 'network', 'host', 'range', 'calculator', 'ipcalc'],
  component: () => import('./ipv4-subnet-calculator.vue'),
  icon: Sitemap,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
