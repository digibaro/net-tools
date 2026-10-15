import { BuildingFactory } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.lab-ca-generator.title'),
  path: '/lab-ca-generator',
  description: translate('tools.lab-ca-generator.description'),
  keywords: ['certificate authority', 'self-signed', 'lab ca', 'homelab', 'internal ca', 'san', 'tls certificate', 'generate certificate', 'openssl'],
  component: () => import('./lab-ca-generator.vue'),
  icon: BuildingFactory,
  createdAt: new Date('2026-10-16'),
  runMode: 'browser',
})
