import { Ruler } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.path-mtu.title'),
  path: '/path-mtu',
  description: translate('tools.path-mtu.description'),
  keywords: ['pmtu', 'path mtu', 'mtu discovery', 'fragmentation', 'don\'t fragment', 'mss', 'pppoe', 'blackhole'],
  component: () => import('./path-mtu.vue'),
  icon: Ruler,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'net.pmtu',
})
