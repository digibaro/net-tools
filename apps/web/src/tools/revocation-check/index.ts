import { Ban } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.revocation-check.title'),
  path: '/revocation-check',
  description: translate('tools.revocation-check.description'),
  keywords: ['ocsp', 'crl', 'revocation', 'revoked certificate', 'ocsp stapling', 'certificate status'],
  component: () => import('./revocation-check.vue'),
  icon: Ban,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'tls.revocation',
})
