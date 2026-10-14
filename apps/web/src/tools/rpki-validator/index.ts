import { ShieldCheck } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.rpki-validator.title'),
  path: '/rpki-validator',
  description: translate('tools.rpki-validator.description'),
  keywords: ['rpki', 'roa', 'rov', 'route origin validation', 'bgp', 'invalid', 'hijack', 'ripestat'],
  component: () => import('./rpki-validator.vue'),
  icon: ShieldCheck,
  createdAt: new Date('2026-10-14'),
  runMode: 'public-api',
  capability: 'bgp.rpki',
})
