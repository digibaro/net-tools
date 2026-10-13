import { Plug } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.port-reference.title'),
  path: '/port-reference',
  description: translate('tools.port-reference.description'),
  keywords: ['port', 'service', 'iana', 'tcp', 'udp', 'well-known', 'registered', 'services', 'lookup'],
  component: () => import('./port-reference.vue'),
  icon: Plug,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
