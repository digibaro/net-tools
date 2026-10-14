import { Fingerprint } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.mac-address-lookup.title'),
  path: '/mac-address-lookup',
  description: translate('tools.mac-address-lookup.description'),
  keywords: ['mac', 'oui', 'vendor', 'manufacturer', 'ieee', 'lookup', 'ethernet', 'hardware address'],
  component: () => import('./mac-address-lookup.vue'),
  icon: Fingerprint,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
