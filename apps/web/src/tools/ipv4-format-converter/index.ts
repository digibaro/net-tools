import { Binary } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ipv4-format-converter.title'),
  path: '/ipv4-format-converter',
  description: translate('tools.ipv4-format-converter.description'),
  keywords: ['ipv4', 'convert', 'decimal', 'integer', 'hex', 'binary', 'octal', 'inet_aton', 'dotted'],
  component: () => import('./ipv4-format-converter.vue'),
  icon: Binary,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
