import { LockOpen } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.open-resolver-test.title'),
  path: '/open-resolver-test',
  description: translate('tools.open-resolver-test.description'),
  keywords: ['open resolver', 'open recursive', 'dns amplification', 'ddos', 'recursion', 'resolver security'],
  component: () => import('./open-resolver-test.vue'),
  icon: LockOpen,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'dns.openresolver',
})
