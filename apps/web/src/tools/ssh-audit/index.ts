import { Terminal2 } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ssh-audit.title'),
  path: '/ssh-audit',
  description: translate('tools.ssh-audit.description'),
  keywords: ['ssh', 'ssh-audit', 'host key', 'fingerprint', 'kex', 'ciphers', 'terrapin', 'openssh', 'hardening'],
  component: () => import('./ssh-audit.vue'),
  icon: Terminal2,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'ssh.audit',
})
