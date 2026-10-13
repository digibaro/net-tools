import { World } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.what-is-my-ip.title'),
  path: '/what-is-my-ip',
  description: translate('tools.what-is-my-ip.description'),
  keywords: ['my ip', 'what is my ip', 'public ip', 'external ip', 'ipv4', 'ipv6', 'nat', 'ipify'],
  component: () => import('./what-is-my-ip.vue'),
  icon: World,
  createdAt: new Date('2026-10-13'),
  runMode: 'hybrid',
})
