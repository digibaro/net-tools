import { Hexagon } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.packet-decoder.title'),
  path: '/packet-decoder',
  description: translate('tools.packet-decoder.description'),
  keywords: ['packet decoder', 'hex dump', 'ethernet', 'ipv4 header', 'ipv6 header', 'tcp header', 'udp', 'icmp', 'dns', 'wireshark'],
  component: () => import('./packet-decoder.vue'),
  icon: Hexagon,
  createdAt: new Date('2026-10-16'),
  runMode: 'browser',
})
