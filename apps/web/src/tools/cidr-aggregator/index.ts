import { ArrowsJoin } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.cidr-aggregator.title'),
  path: '/cidr-aggregator',
  description: translate('tools.cidr-aggregator.description'),
  keywords: ['cidr', 'aggregate', 'summarize', 'merge', 'supernet', 'prefix list', 'dedupe', 'route summarization'],
  component: () => import('./cidr-aggregator.vue'),
  icon: ArrowsJoin,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
