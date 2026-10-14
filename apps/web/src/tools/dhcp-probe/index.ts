import { ShieldCheck } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dhcp-probe.title'),
  path: '/dhcp-probe',
  description: translate('tools.dhcp-probe.description'),
  keywords: ['dhcp', 'rogue dhcp', 'dhcp server', 'dhcpdiscover', 'dhcp offer', 'ip conflict'],
  component: () => import('./dhcp-probe.vue'),
  icon: ShieldCheck,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'lan.dhcp',
  requiresLan: true,
})
