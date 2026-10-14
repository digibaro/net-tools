import { Mask } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.tor-exit-check.title'),
  path: '/tor-exit-check',
  description: translate('tools.tor-exit-check.description'),
  keywords: ['tor', 'exit node', 'onion', 'relay', 'anonymizer', 'tor exit list', 'onionoo'],
  component: () => import('./tor-exit-check.vue'),
  icon: Mask,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'rep.tor',
})
