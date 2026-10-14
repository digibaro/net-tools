import { FileZip } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.compression-check.title'),
  path: '/compression-check',
  description: translate('tools.compression-check.description'),
  keywords: ['compression', 'gzip', 'brotli', 'br', 'zstd', 'deflate', 'content-encoding', 'page size'],
  component: () => import('./compression-check.vue'),
  icon: FileZip,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'web.compression',
})
