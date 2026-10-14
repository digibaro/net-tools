import { Link } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.url-parser.title'),
  path: '/url-parser',
  description: translate('tools.url-parser.description'),
  keywords: ['url', 'uri', 'parse', 'query', 'params', 'encode', 'decode', 'percent', 'urlencode'],
  component: () => import('./url-parser.vue'),
  icon: Link,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
