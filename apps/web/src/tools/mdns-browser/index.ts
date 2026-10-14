import { AccessPoint } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.mdns-browser.title'),
  path: '/mdns-browser',
  description: translate('tools.mdns-browser.description'),
  keywords: ['mdns', 'bonjour', 'zeroconf', 'dns-sd', 'avahi', 'airplay', 'chromecast', 'printer discovery', '.local'],
  component: () => import('./mdns-browser.vue'),
  icon: AccessPoint,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'lan.mdns',
  requiresLan: true,
})
