import { Dice } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.random-ip-generator.title'),
  path: '/random-ip-generator',
  description: translate('tools.random-ip-generator.description'),
  keywords: ['random ip', 'generator', 'test data', 'fake ip', 'sample addresses', 'ipv4', 'ipv6', 'prefix'],
  component: () => import('./random-ip-generator.vue'),
  icon: Dice,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
