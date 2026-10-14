import { Route } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.as-path-visualizer.title'),
  path: '/as-path-visualizer',
  description: translate('tools.as-path-visualizer.description'),
  keywords: ['as path', 'as-path', 'prepend', 'bgp path', 'as_set', 'route', 'transit', 'upstream'],
  component: () => import('./as-path-visualizer.vue'),
  icon: Route,
  createdAt: new Date('2026-10-14'),
  runMode: 'browser',
})
