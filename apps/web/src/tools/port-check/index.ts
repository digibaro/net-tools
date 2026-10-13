import { Plug } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.port-check.title'),
  path: '/port-check',
  description: translate('tools.port-check.description'),
  keywords: ['port', 'open port', 'check', 'tcp', 'udp', 'firewall', 'reachable', 'is port open'],
  component: () => import('./port-check.vue'),
  icon: Plug,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'net.portcheck',
})
