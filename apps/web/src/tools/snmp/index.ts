import { Sitemap } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.snmp.title'),
  path: '/snmp',
  description: translate('tools.snmp.description'),
  keywords: ['snmp', 'snmpwalk', 'snmpget', 'mib', 'oid', 'v2c', 'v3', 'community', 'ifTable', 'sysDescr'],
  component: () => import('./snmp.vue'),
  icon: Sitemap,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'net.snmp',
})
