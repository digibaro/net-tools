import { Lock } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.wireguard-config.title'),
  path: '/wireguard-config',
  description: translate('tools.wireguard-config.description'),
  keywords: ['wireguard', 'vpn', 'wg0.conf', 'x25519', 'keypair', 'peer', 'qr code', 'tunnel', 'allowedips'],
  component: () => import('./wireguard-config.vue'),
  icon: Lock,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
