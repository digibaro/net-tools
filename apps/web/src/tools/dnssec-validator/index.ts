import { ShieldLock } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.dnssec-validator.title'),
  path: '/dnssec-validator',
  description: translate('tools.dnssec-validator.description'),
  keywords: ['dnssec', 'chain of trust', 'ds', 'dnskey', 'rrsig', 'bogus', 'servfail', 'key rollover', 'trust anchor'],
  component: () => import('./dnssec-validator.vue'),
  icon: ShieldLock,
  createdAt: new Date('2026-10-14'),
  runMode: 'server',
  capability: 'dns.dnssec',
})
