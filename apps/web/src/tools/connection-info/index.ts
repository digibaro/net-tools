import { Wifi } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.connection-info.title'),
  path: '/connection-info',
  description: translate('tools.connection-info.description'),
  keywords: ['ipv6 test', 'dual stack', 'connection type', 'network information', 'effective type', 'http/3', 'page load timing', 'browser network'],
  component: () => import('./connection-info.vue'),
  icon: Wifi,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
