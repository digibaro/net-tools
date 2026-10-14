import { DevicesPc } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.arp-scan.title'),
  path: '/arp-scan',
  description: translate('tools.arp-scan.description'),
  keywords: ['arp scan', 'host discovery', 'lan scan', 'devices on network', 'mac address', 'vendor', 'arp-scan'],
  component: () => import('./arp-scan.vue'),
  icon: DevicesPc,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'lan.arpscan',
  requiresLan: true,
})
