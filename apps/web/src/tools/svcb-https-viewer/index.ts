import { Certificate } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.svcb-https-viewer.title'),
  path: '/svcb-https-viewer',
  description: translate('tools.svcb-https-viewer.description'),
  keywords: ['svcb', 'https record', 'type 65', 'ech', 'encrypted client hello', 'alpn', 'http/3', 'ddr', '_dns.resolver.arpa', 'rfc 9460'],
  component: () => import('./svcb-https-viewer.vue'),
  icon: Certificate,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'dns.svcb',
})
