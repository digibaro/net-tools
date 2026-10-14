import { World } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.gateway-info.title'),
  path: '/gateway-info',
  description: translate('tools.gateway-info.description'),
  keywords: ['default gateway', 'nat', 'nat-pmp', 'pcp', 'upnp', 'igd', 'external ip', 'double nat', 'cgnat', 'router'],
  component: () => import('./gateway-info.vue'),
  icon: World,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'lan.gateway',
  requiresLan: true,
})
