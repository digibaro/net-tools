import type { Tool, ToolCategory } from './tool'
import { translate } from '@/plugins/i18n.plugin'
import { tool as cidrRangeConverter } from './cidr-range-converter'
import { tool as ipv4SubnetCalculator } from './ipv4-subnet-calculator'
import { tool as ipv6SubnetCalculator } from './ipv6-subnet-calculator'

// Category order matches docs/TOOLS_CATALOG.md. Empty categories are hidden.
const categories: { id: string, components: Tool[] }[] = [
  { id: 'ip', components: [ipv4SubnetCalculator, ipv6SubnetCalculator, cidrRangeConverter] },
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
