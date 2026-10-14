import { Cloud } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.cloud-ip-lookup.title'),
  path: '/cloud-ip-lookup',
  description: translate('tools.cloud-ip-lookup.description'),
  keywords: ['cloud ip', 'aws ip ranges', 'gcp', 'cloudflare', 'fastly', 'github', 'oracle', 'digitalocean', 'cdn', 'region', 'which cloud'],
  component: () => import('./cloud-ip-lookup.vue'),
  icon: Cloud,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'bgp.cloudranges',
})
