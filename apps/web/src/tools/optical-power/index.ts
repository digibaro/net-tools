import { Bulb } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.optical-power.title'),
  path: '/optical-power',
  description: translate('tools.optical-power.description'),
  keywords: ['dbm', 'mw', 'milliwatt', 'optical power', 'fiber', 'fibre', 'link budget', 'attenuation', 'sfp', 'transceiver'],
  component: () => import('./optical-power.vue'),
  icon: Bulb,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
