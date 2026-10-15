import { Cloud } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.cdn-waf-detector.title'),
  path: '/cdn-waf-detector',
  description: translate('tools.cdn-waf-detector.description'),
  keywords: ['cdn', 'waf', 'cloudflare', 'akamai', 'fastly', 'cloudfront', 'imperva', 'bot protection', 'hosting', 'which cdn'],
  component: () => import('./cdn-waf-detector.vue'),
  icon: Cloud,
  createdAt: new Date('2026-10-16'),
  runMode: 'server',
  capability: 'web.cdn',
})
