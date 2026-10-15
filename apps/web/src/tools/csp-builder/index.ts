import { Code } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.csp-builder.title'),
  path: '/csp-builder',
  description: translate('tools.csp-builder.description'),
  keywords: ['csp', 'content security policy', 'csp evaluator', 'nonce', 'strict-dynamic', 'xss', 'security header', 'unsafe-inline'],
  component: () => import('./csp-builder.vue'),
  icon: Code,
  createdAt: new Date('2026-10-16'),
  runMode: 'browser',
})
