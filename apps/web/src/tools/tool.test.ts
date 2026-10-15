import { afterEach, describe, expect, it } from 'vitest'
import { i18n } from '@/plugins/i18n.plugin'
import { toolsByCategory } from '.'
import { defineTool, isNew, LATEST_RELEASE } from './tool'

describe('isNew', () => {
  it('marks only tools added after the latest release', () => {
    expect(isNew(new Date('2026-10-01'), new Date('2026-10-14'))).toBe(false)
    expect(isNew(new Date('2026-10-14'), new Date('2026-10-14'))).toBe(false)
    expect(isNew(new Date('2026-10-20'), new Date('2026-10-14'))).toBe(true)
    expect(isNew(undefined)).toBe(false)
  })
  it('badges exactly the tools created after the latest release', () => {
    const tools = toolsByCategory.flatMap(c => c.components)
    for (const t of tools)
      expect(t.isNew, t.path).toBe(!!t.createdAt && t.createdAt.getTime() > LATEST_RELEASE.date.getTime())
    expect(LATEST_RELEASE.version).toMatch(/^\d+\.\d+\.\d+/)
    if (LATEST_RELEASE.version === '0.4.0') // the LAN tools added after v0.4.0 (baseline up to the v0.5.0 build)
      expect(tools.filter(t => t.isNew).map(t => t.path)).toEqual(expect.arrayContaining(['/lldp-neighbors', '/ipv6-neighbors', '/dhcpv6-probe']))
  })
})

describe('translated tool names', () => {
  afterEach(() => {
    i18n.global.locale.value = 'en'
  })
  it('follow the active language', () => {
    const tool = defineTool({ name: 'x', path: '/ping', description: 'x', keywords: [], component: async () => ({}), icon: {}, runMode: 'server' })
    expect(tool.name).toBe('Ping')
    i18n.global.locale.value = 'nl'
    expect(tool.description).toContain('Stuur ICMP echo requests')
    const ip = toolsByCategory.find(c => c.id === 'ip')!
    expect(ip.name).toBe('IP & adressering')
    expect(ip.components[0]!.name).toBe('IPv4-subnetcalculator')
  })
})
