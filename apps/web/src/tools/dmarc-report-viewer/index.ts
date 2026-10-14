import { FileAnalytics } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dmarc-report-viewer.title'),
  path: '/dmarc-report-viewer',
  description: translate('tools.dmarc-report-viewer.description'),
  keywords: ['dmarc', 'rua', 'aggregate report', 'dmarc report', 'xml', 'spf', 'dkim', 'email authentication'],
  component: () => import('./dmarc-report-viewer.vue'),
  icon: FileAnalytics,
  createdAt: new Date('2026-10-14'),
  runMode: 'browser',
})
