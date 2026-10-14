import { ListDetails } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.tls-scanner.title'),
  path: '/tls-scanner',
  description: translate('tools.tls-scanner.description'),
  keywords: ['tls scan', 'ssl scan', 'cipher suites', 'tls 1.0', 'tls 1.3', 'weak ciphers', 'forward secrecy', 'post-quantum', 'ssllabs'],
  component: () => import('./tls-scanner.vue'),
  icon: ListDetails,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'tls.scan',
})
