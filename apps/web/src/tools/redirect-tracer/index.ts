import { ArrowsShuffle } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.redirect-tracer.title'),
  path: '/redirect-tracer',
  description: translate('tools.redirect-tracer.description'),
  keywords: ['redirect', '301', '302', 'chain', 'location', 'http', 'https', 'url', 'trace'],
  component: () => import('./redirect-tracer.vue'),
  icon: ArrowsShuffle,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'web.redirects',
})
