import { Rocket } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.http-protocols.title'),
  path: '/http-protocols',
  description: translate('tools.http-protocols.description'),
  keywords: ['http/2', 'http2', 'http/3', 'http3', 'quic', 'alpn', 'alt-svc', 'h2', 'h3'],
  component: () => import('./http-protocols.vue'),
  icon: Rocket,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'web.protocols',
})
