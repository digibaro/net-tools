import { ListSearch } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ct-search.title'),
  path: '/ct-search',
  description: translate('tools.ct-search.description'),
  keywords: ['certificate transparency', 'ct logs', 'crt.sh', 'subdomain discovery', 'subdomains', 'issued certificates', 'audit'],
  component: () => import('./ct-search.vue'),
  icon: ListSearch,
  createdAt: new Date('2026-10-14'),
  runMode: 'public-api',
  capability: 'tls.ct',
})
