import { Eye } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.looking-glass.title'),
  path: '/looking-glass',
  description: translate('tools.looking-glass.description'),
  keywords: ['looking glass', 'bgp', 'as path', 'ris', 'route collector', 'communities', 'ripestat', 'routes'],
  component: () => import('./looking-glass.vue'),
  icon: Eye,
  createdAt: new Date('2026-10-14'),
  runMode: 'public-api',
  capability: 'bgp.lg',
})
