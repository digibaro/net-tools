import { Scan } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.passive-exposure.title'),
  path: '/passive-exposure',
  description: translate('tools.passive-exposure.description'),
  keywords: ['shodan', 'internetdb', 'open ports', 'exposure', 'cve', 'vulnerabilities', 'attack surface', 'passive scan'],
  component: () => import('./passive-exposure.vue'),
  icon: Scan,
  createdAt: new Date('2026-10-14'),
  runMode: 'public-api',
  capability: 'rep.internetdb',
})
