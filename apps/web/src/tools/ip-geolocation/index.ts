import { MapPin } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.ip-geolocation.title'),
  path: '/ip-geolocation',
  description: translate('tools.ip-geolocation.description'),
  keywords: ['geolocation', 'geoip', 'location', 'country', 'city', 'isp', 'asn', 'maxmind', 'geolite'],
  component: () => import('./ip-geolocation.vue'),
  icon: MapPin,
  createdAt: new Date('2026-10-13'),
  runMode: 'hybrid',
  capability: 'rep.geo',
})
