import { FileText } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.well-known-files.title'),
  path: '/well-known-files',
  description: translate('tools.well-known-files.description'),
  keywords: ['robots.txt', 'sitemap', 'security.txt', 'rfc 9116', 'well-known', 'crawler', 'seo', 'vulnerability disclosure'],
  component: () => import('./well-known-files.vue'),
  icon: FileText,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'web.wellknown',
})
