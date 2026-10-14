import { Certificate } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.certificate-decoder.title'),
  path: '/certificate-decoder',
  description: translate('tools.certificate-decoder.description'),
  keywords: ['certificate', 'x509', 'x.509', 'pem', 'der', 'ssl', 'tls', 'decode', 'san', 'fingerprint', 'chain'],
  component: () => import('./certificate-decoder.vue'),
  icon: Certificate,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
