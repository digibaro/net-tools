import { FileCode } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.zone-builder.title'),
  path: '/zone-builder',
  description: translate('tools.zone-builder.description'),
  keywords: ['zone file', 'bind', 'dns record', 'srv', 'txt', 'spf', 'caa', 'mx', 'soa', 'record generator'],
  component: () => import('./zone-builder.vue'),
  icon: FileCode,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
