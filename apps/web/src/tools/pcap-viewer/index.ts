import { FileAnalytics } from '@vicons/tabler'
import { translate } from '@/plugins/i18n.plugin'
import { defineTool } from '../tool'

export const tool = defineTool({
  name: translate('tools.pcap-viewer.title'),
  path: '/pcap-viewer',
  description: translate('tools.pcap-viewer.description'),
  keywords: ['pcap', 'pcapng', 'packet capture', 'tcpdump', 'wireshark', 'conversations', 'dns queries'],
  component: () => import('./pcap-viewer.vue'),
  icon: FileAnalytics,
  createdAt: new Date('2026-10-16'),
  runMode: 'browser',
})
