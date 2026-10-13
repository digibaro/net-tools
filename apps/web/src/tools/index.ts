import type { Tool, ToolCategory } from './tool'
import { translate } from '@/plugins/i18n.plugin'
import { tool as cidrAggregator } from './cidr-aggregator'
import { tool as cidrRangeConverter } from './cidr-range-converter'
import { tool as ipInSubnet } from './ip-in-subnet'
import { tool as ipv4FormatConverter } from './ipv4-format-converter'
import { tool as ipv4SubnetCalculator } from './ipv4-subnet-calculator'
import { tool as ipv6ExpandCompress } from './ipv6-expand-compress'
import { tool as ipv6SubnetCalculator } from './ipv6-subnet-calculator'
import { tool as subnetSplitter } from './subnet-splitter'

// Category order matches docs/TOOLS_CATALOG.md. Empty categories are hidden.
const categories: { id: string, components: Tool[] }[] = [
  { id: 'ip', components: [ipv4SubnetCalculator, ipv6SubnetCalculator, cidrRangeConverter, cidrAggregator, subnetSplitter, ipInSubnet, ipv4FormatConverter, ipv6ExpandCompress] },
  { id: 'dns', components: [] },
  { id: 'connectivity', components: [] },
  { id: 'web', components: [] },
  { id: 'tls', components: [] },
  { id: 'email', components: [] },
  { id: 'bgp', components: [] },
  { id: 'reputation', components: [] },
  { id: 'lan', components: [] },
  { id: 'calculators', components: [] },
  { id: 'client', components: [] },
]

export const toolsByCategory: ToolCategory[] = categories
  .filter(c => c.components.length > 0)
  .map(c => ({ ...c, name: translate(`categories.${c.id}`) }))
