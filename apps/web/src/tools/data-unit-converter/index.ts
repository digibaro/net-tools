import { Ruler2 } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.data-unit-converter.title'),
  path: '/data-unit-converter',
  description: translate('tools.data-unit-converter.description'),
  keywords: ['bits', 'bytes', 'units', 'convert', 'kib', 'mib', 'mbps', 'mb/s', 'gigabit', 'binary prefix', 'si'],
  component: () => import('./data-unit-converter.vue'),
  icon: Ruler2,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
