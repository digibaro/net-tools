import { ArrowBackUp } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.reverse-dns-name.title'),
  path: '/reverse-dns-name',
  description: translate('tools.reverse-dns-name.description'),
  keywords: ['reverse', 'dns', 'ptr', 'in-addr.arpa', 'ip6.arpa', 'zone', 'rfc 2317', 'classless', 'delegation'],
  component: () => import('./reverse-dns-name.vue'),
  icon: ArrowBackUp,
  createdAt: new Date('2026-10-13'),
  runMode: 'browser',
})
