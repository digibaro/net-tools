import { FileCertificate } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.pkcs12-inspector.title'),
  path: '/pkcs12-inspector',
  description: translate('tools.pkcs12-inspector.description'),
  keywords: ['pkcs12', 'pkcs#12', 'pfx', 'p12', 'keystore', 'certificate bundle', 'private key', 'friendly name', 'openssl legacy'],
  component: () => import('./pkcs12-inspector.vue'),
  icon: FileCertificate,
  createdAt: new Date('2026-10-16'),
  runMode: 'browser',
})
