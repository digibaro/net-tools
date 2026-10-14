import { Power } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.wake-on-lan.title'),
  path: '/wake-on-lan',
  description: translate('tools.wake-on-lan.description'),
  keywords: ['wake on lan', 'wol', 'magic packet', 'power on', 'remote wake'],
  component: () => import('./wake-on-lan.vue'),
  icon: Power,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'lan.wol',
  requiresLan: true,
})
