import { Server } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.server-network-info.title'),
  path: '/server-network-info',
  description: translate('tools.server-network-info.description'),
  keywords: ['server', 'interfaces', 'routes', 'routing table', 'gateway', 'resolv.conf', 'ip addr', 'netstat'],
  component: () => import('./server-network-info.vue'),
  icon: Server,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'net.serverinfo',
})
