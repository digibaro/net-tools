import { Router } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ssdp-discovery.title'),
  path: '/ssdp-discovery',
  description: translate('tools.ssdp-discovery.description'),
  keywords: ['ssdp', 'upnp', 'dlna', 'igd', 'smart tv', 'media server', 'device discovery'],
  component: () => import('./ssdp-discovery.vue'),
  icon: Router,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'lan.ssdp',
  requiresLan: true,
})
