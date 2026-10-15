import { ShieldCheck } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dhcpv6-probe.title'),
  path: '/dhcpv6-probe',
  description: translate('tools.dhcpv6-probe.description'),
  keywords: ['dhcpv6', 'rogue dhcpv6', 'solicit', 'advertise', 'prefix delegation', 'ia_pd', 'ipv6 dhcp'],
  component: () => import('./dhcpv6-probe.vue'),
  icon: ShieldCheck,
  createdAt: new Date('2026-10-15'),
  runMode: 'server',
  capability: 'lan.dhcpv6',
  requiresLan: true,
})
