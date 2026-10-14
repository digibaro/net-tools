import { FileDownload } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.zone-transfer-test.title'),
  path: '/zone-transfer-test',
  description: translate('tools.zone-transfer-test.description'),
  keywords: ['axfr', 'zone transfer', 'dns security', 'audit', 'name server', 'zone leak'],
  component: () => import('./zone-transfer-test.vue'),
  icon: FileDownload,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'dns.axfr',
})
