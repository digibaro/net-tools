import { ShieldCheckered } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.roa-coverage.title'),
  path: '/roa-coverage',
  description: translate('tools.roa-coverage.description'),
  keywords: ['roa', 'rpki', 'route origin validation', 'rov', 'maxlength', 'invalid', 'coverage', 'rfc 6811', 'rfc 9319'],
  component: () => import('./roa-coverage.vue'),
  icon: ShieldCheckered,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'bgp.roacoverage',
})
