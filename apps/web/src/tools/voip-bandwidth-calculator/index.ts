import { PhoneCall } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.voip-bandwidth-calculator.title'),
  path: '/voip-bandwidth-calculator',
  description: translate('tools.voip-bandwidth-calculator.description'),
  keywords: ['voip', 'codec', 'g.711', 'g.729', 'opus', 'bandwidth per call', 'erlang', 'erlang b', 'trunk', 'sip channels'],
  component: () => import('./voip-bandwidth-calculator.vue'),
  icon: PhoneCall,
  createdAt: new Date('2026-10-16'),
  runMode: 'browser',
})
