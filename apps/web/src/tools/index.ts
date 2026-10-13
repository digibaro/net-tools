import type { Tool, ToolCategory } from './tool'
import { translate } from '@/plugins/i18n.plugin'
import { tool as bandwidthCalculator } from './bandwidth-calculator'
import { tool as bdpCalculator } from './bdp-calculator'
import { tool as caaChecker } from './caa-checker'
import { tool as certificateDecoder } from './certificate-decoder'
import { tool as cidrAggregator } from './cidr-aggregator'
import { tool as cidrRangeConverter } from './cidr-range-converter'
import { tool as dataUnitConverter } from './data-unit-converter'
import { tool as dnsLookup } from './dns-lookup'
import { tool as dnsRecordTypes } from './dns-record-types'
import { tool as domainParser } from './domain-parser'
import { tool as eui64Calculator } from './eui64-calculator'
import { tool as httpStatusCodes } from './http-status-codes'
import { tool as ipInSubnet } from './ip-in-subnet'
import { tool as ipSpecialCheck } from './ip-special-check'
import { tool as ipv4FormatConverter } from './ipv4-format-converter'
import { tool as ipv4SubnetCalculator } from './ipv4-subnet-calculator'
import { tool as ipv6ExpandCompress } from './ipv6-expand-compress'
import { tool as ipv6SubnetCalculator } from './ipv6-subnet-calculator'
import { tool as ipv6UlaGenerator } from './ipv6-ula-generator'
import { tool as macAddressLookup } from './mac-address-lookup'
import { tool as macFormatter } from './mac-formatter'
import { tool as mtuCalculator } from './mtu-calculator'
import { tool as portReference } from './port-reference'
import { tool as protocolReference } from './protocol-reference'
import { tool as punycodeConverter } from './punycode-converter'
import { tool as requestHeaders } from './request-headers'
import { tool as reverseDns } from './reverse-dns'
import { tool as reverseDnsName } from './reverse-dns-name'
import { tool as subnetCheatSheet } from './subnet-cheat-sheet'
import { tool as subnetSplitter } from './subnet-splitter'
import { tool as urlParser } from './url-parser'
import { tool as userAgentParser } from './user-agent-parser'
import { tool as whatIsMyIp } from './what-is-my-ip'
import { tool as wifiQrGenerator } from './wifi-qr-generator'
import { tool as wildcardMaskCalculator } from './wildcard-mask-calculator'

// Category order matches docs/TOOLS_CATALOG.md. Empty categories are hidden.
const categories: { id: string, components: Tool[] }[] = [
  { id: 'ip', components: [ipv4SubnetCalculator, ipv6SubnetCalculator, cidrRangeConverter, cidrAggregator, subnetSplitter, ipInSubnet, ipv4FormatConverter, ipv6ExpandCompress, ipv6UlaGenerator, ipSpecialCheck, wildcardMaskCalculator, reverseDnsName, macAddressLookup, macFormatter, eui64Calculator, whatIsMyIp] },
  { id: 'dns', components: [punycodeConverter, domainParser, dnsLookup, reverseDns, caaChecker] },
  { id: 'connectivity', components: [] },
  { id: 'web', components: [urlParser, userAgentParser, httpStatusCodes] },
  { id: 'tls', components: [certificateDecoder] },
  { id: 'email', components: [] },
  { id: 'bgp', components: [] },
  { id: 'reputation', components: [] },
  { id: 'lan', components: [] },
  { id: 'calculators', components: [bandwidthCalculator, dataUnitConverter, bdpCalculator, mtuCalculator, portReference, protocolReference, dnsRecordTypes, subnetCheatSheet, wifiQrGenerator] },
  { id: 'client', components: [requestHeaders] },
]

export const toolsByCategory: ToolCategory[] = categories
  .filter(c => c.components.length > 0)
  .map(c => ({ ...c, name: translate(`categories.${c.id}`) }))
