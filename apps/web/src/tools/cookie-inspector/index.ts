import { Cookie } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.cookie-inspector.title'),
  path: '/cookie-inspector',
  description: translate('tools.cookie-inspector.description'),
  keywords: ['cookies', 'set-cookie', 'samesite', 'httponly', 'secure', '__host-', 'session cookie', 'gdpr'],
  component: () => import('./cookie-inspector.vue'),
  icon: Cookie,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'web.cookies',
})
