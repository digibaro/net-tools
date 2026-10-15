import { EyeCheck } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dns-proxy-check.title'),
  path: '/dns-proxy-check',
  description: translate('tools.dns-proxy-check.description'),
  keywords: ['dns hijack', 'nxdomain hijacking', 'dns filtering', 'pi-hole', 'transparent proxy', 'http proxy', 'via header', 'dns manipulation', 'captive'],
  component: () => import('./dns-proxy-check.vue'),
  icon: EyeCheck,
  createdAt: new Date('2026-10-16'),
  runMode: 'hybrid',
  capability: 'dns.lookup',
})
