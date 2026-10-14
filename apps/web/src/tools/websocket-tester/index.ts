import { PlugConnected } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.websocket-tester.title'),
  path: '/websocket-tester',
  description: translate('tools.websocket-tester.description'),
  keywords: ['websocket', 'ws', 'wss', 'socket tester', 'echo', 'close code', 'subprotocol', 'realtime'],
  component: () => import('./websocket-tester.vue'),
  icon: PlugConnected,
  createdAt: new Date('2026-10-14'),
  runMode: 'browser',
})
