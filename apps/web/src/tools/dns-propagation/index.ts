import { World } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dns-propagation.title'),
  path: '/dns-propagation',
  description: translate('tools.dns-propagation.description'),
  keywords: ['dns', 'propagation', 'resolvers', 'compare', 'cache', 'ttl', 'whatsmydns', 'global'],
  component: () => import('./dns-propagation.vue'),
  icon: World,
  createdAt: new Date('2026-10-13'),
  runMode: 'hybrid',
  capability: 'dns.propagation',
})
