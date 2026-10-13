import { Sitemap } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.domain-parser.title'),
  path: '/domain-parser',
  description: translate('tools.domain-parser.description'),
  keywords: ['domain', 'parse', 'public suffix', 'psl', 'tld', 'subdomain', 'registrable', 'etld+1', 'apex'],
  component: () => import('./domain-parser.vue'),
  icon: Sitemap,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
