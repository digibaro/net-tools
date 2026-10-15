import { FileAlert } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.mixed-content-checker.title'),
  path: '/mixed-content-checker',
  description: translate('tools.mixed-content-checker.description'),
  keywords: ['mixed content', 'insecure content', 'http resources', 'https', 'upgrade-insecure-requests', 'padlock'],
  component: () => import('./mixed-content-checker.vue'),
  icon: FileAlert,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'web.mixed',
})
