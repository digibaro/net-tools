import { Link } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.cert-chain-validator.title'),
  path: '/cert-chain-validator',
  description: translate('tools.cert-chain-validator.description'),
  keywords: ['certificate chain', 'intermediate', 'missing intermediate', 'chain order', 'aia', 'ca issuers', 'fullchain', 'pem', 'trust'],
  component: () => import('./cert-chain-validator.vue'),
  icon: Link,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'tls.chain',
})
