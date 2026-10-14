import { Code } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.curl-builder.title'),
  path: '/curl-builder',
  description: translate('tools.curl-builder.description'),
  keywords: ['curl', 'curl builder', 'curl parser', 'curl to form', 'http request', 'command line', 'api'],
  component: () => import('./curl-builder.vue'),
  icon: Code,
  createdAt: new Date('2026-10-14'),
  runMode: 'browser',
})
