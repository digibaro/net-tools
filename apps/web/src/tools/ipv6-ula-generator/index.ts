import { Dice } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ipv6-ula-generator.title'),
  path: '/ipv6-ula-generator',
  description: translate('tools.ipv6-ula-generator.description'),
  keywords: ['ipv6', 'ula', 'unique local', 'rfc 4193', 'fd00', 'private', 'generator', 'random', 'prefix'],
  component: () => import('./ipv6-ula-generator.vue'),
  icon: Dice,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
