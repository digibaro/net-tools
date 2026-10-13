import { Certificate } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.tls-inspector.title'),
  path: '/tls-inspector',
  description: translate('tools.tls-inspector.description'),
  keywords: ['tls', 'ssl', 'certificate', 'chain', 'sni', 'starttls', 'cipher', 'handshake', 'openssl s_client'],
  component: () => import('./tls-inspector.vue'),
  icon: Certificate,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'tls.inspect',
})
