import { Key } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.cert-key-matcher.title'),
  path: '/cert-key-matcher',
  description: translate('tools.cert-key-matcher.description'),
  keywords: ['certificate key match', 'private key', 'csr', 'modulus', 'public key', 'spki', 'mismatch', 'openssl'],
  component: () => import('./cert-key-matcher.vue'),
  icon: Key,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
