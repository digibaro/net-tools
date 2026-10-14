import { EyeCheck } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.webrtc-leak-test.title'),
  path: '/webrtc-leak-test',
  description: translate('tools.webrtc-leak-test.description'),
  keywords: ['webrtc', 'ip leak', 'vpn leak', 'stun', 'ice candidates', 'privacy', 'local ip'],
  component: () => import('./webrtc-leak-test.vue'),
  icon: EyeCheck,
  createdAt: new Date('2026-10-14'),
  runMode: 'browser',
})
