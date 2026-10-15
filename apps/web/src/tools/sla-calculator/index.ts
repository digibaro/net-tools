import { Percentage } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.sla-calculator.title'),
  path: '/sla-calculator',
  description: translate('tools.sla-calculator.description'),
  keywords: ['sla', 'uptime', 'availability', 'nines', '99.9', 'downtime', 'composite availability', 'redundancy'],
  component: () => import('./sla-calculator.vue'),
  icon: Percentage,
  createdAt: new Date('2026-10-16'),
  runMode: 'browser',
})
