import { History } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.cache-header-analyzer.title'),
  path: '/cache-header-analyzer',
  description: translate('tools.cache-header-analyzer.description'),
  keywords: ['cache-control', 'caching', 'max-age', 's-maxage', 'etag', 'last-modified', 'vary', 'age', 'cdn cache', '304', 'rfc 9111'],
  component: () => import('./cache-header-analyzer.vue'),
  icon: History,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'web.cache',
})
