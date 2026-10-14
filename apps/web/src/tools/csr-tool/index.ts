import { FileCertificate } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.csr-tool.title'),
  path: '/csr-tool',
  description: translate('tools.csr-tool.description'),
  keywords: ['csr', 'certificate signing request', 'pkcs10', 'private key', 'generate csr', 'decode csr', 'openssl req', 'san'],
  component: () => import('./csr-tool.vue'),
  icon: FileCertificate,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
