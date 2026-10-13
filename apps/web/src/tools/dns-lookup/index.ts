import { World } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dns-lookup.title'),
  path: '/dns-lookup',
  description: translate('tools.dns-lookup.description'),
  keywords: ['dns', 'dig', 'nslookup', 'lookup', 'record', 'a', 'aaaa', 'mx', 'txt', 'cname', 'resolver', 'doh', 'dot'],
  component: () => import('./dns-lookup.vue'),
  icon: World,
  createdAt: new Date('2026-10-13'),
  runMode: 'hybrid',
})
