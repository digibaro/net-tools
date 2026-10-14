import { Scan } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.port-scanner.title'),
  path: '/port-scanner',
  description: translate('tools.port-scanner.description'),
  keywords: ['port scan', 'nmap', 'open ports', 'tcp scan', 'firewall', 'services'],
  component: () => import('./port-scanner.vue'),
  icon: Scan,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'net.portscan',
})
