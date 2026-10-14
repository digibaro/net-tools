import { Qrcode } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.wifi-qr-generator.title'),
  path: '/wifi-qr-generator',
  description: translate('tools.wifi-qr-generator.description'),
  keywords: ['wifi', 'wi-fi', 'qr', 'qr code', 'wpa', 'wpa3', 'guest network', 'share', 'ssid'],
  component: () => import('./wifi-qr-generator.vue'),
  icon: Qrcode,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
