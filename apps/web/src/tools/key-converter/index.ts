import { Key } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.key-converter.title'),
  path: '/key-converter',
  description: translate('tools.key-converter.description'),
  keywords: ['jwk', 'pem', 'ssh key', 'pkcs8', 'pkcs1', 'spki', 'openssh', 'ed25519', 'ecdsa', 'rsa', 'fingerprint', 'thumbprint', 'convert key'],
  component: () => import('./key-converter.vue'),
  icon: Key,
  createdAt: new Date('2026-10-14'),
  runMode: 'browser',
})
