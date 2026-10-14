import { LetterCase } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.punycode-converter.title'),
  path: '/punycode-converter',
  description: translate('tools.punycode-converter.description'),
  keywords: ['punycode', 'idn', 'idna', 'xn--', 'unicode', 'international', 'domain', 'uts46'],
  component: () => import('./punycode-converter.vue'),
  icon: LetterCase,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
