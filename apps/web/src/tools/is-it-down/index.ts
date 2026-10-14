import { World } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.is-it-down.title'),
  path: '/is-it-down',
  description: translate('tools.is-it-down.description'),
  keywords: ['is it down', 'down for everyone', 'global ping', 'worldwide', 'globalping', 'uptime', 'reachability', 'outage'],
  component: () => import('./is-it-down.vue'),
  icon: World,
  createdAt: new Date('2026-10-14'),
  runMode: 'public-api',
  capability: 'remote.globalping',
})
