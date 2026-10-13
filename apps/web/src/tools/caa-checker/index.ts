import { ShieldCheck } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.caa-checker.title'),
  path: '/caa-checker',
  description: translate('tools.caa-checker.description'),
  keywords: ['caa', 'certificate authority authorization', 'issue', 'issuewild', 'iodef', 'rfc 8659', 'dns', 'certificate'],
  component: () => import('./caa-checker.vue'),
  icon: ShieldCheck,
  createdAt: new Date('2026-10-13'),
  runMode: 'server',
  capability: 'dns.caa',
})
