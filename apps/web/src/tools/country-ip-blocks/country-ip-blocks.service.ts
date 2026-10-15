import type { components } from '@/api/schema'

export type CountryBlocks = components['schemas']['CountryBlocks']

export const FORMATS = ['plain', 'cisco', 'juniper', 'nginx', 'ipset', 'mikrotik'] as const
export type Format = typeof FORMATS[number]

export const isCountryCode = (s: string) => /^[a-z]{2}$/i.test(s.trim())

/** Renders prefixes as a list or a ready-to-paste config snippet. `name` is used as list/set name. */
export function render(prefixes: string[], format: Format, name: string): string {
  const v6 = (p: string) => p.includes(':')
  switch (format) {
    case 'plain':
      return prefixes.join('\n')
    case 'cisco':
      return prefixes.map(p => `${v6(p) ? 'ipv6' : 'ip'} prefix-list ${name}${v6(p) ? '-V6' : ''} permit ${p}`).join('\n')
    case 'juniper':
      return [`policy-options {`, `    prefix-list ${name} {`, ...prefixes.map(p => `        ${p};`), `    }`, `}`].join('\n')
    case 'nginx':
      return [...prefixes.map(p => `deny ${p};`), 'allow all;'].join('\n')
    case 'ipset': {
      const sets = [['inet', prefixes.filter(p => !v6(p)), name], ['inet6', prefixes.filter(v6), `${name}-v6`]] as const
      return sets.filter(([, list]) => list.length).flatMap(([family, list, set]) => [`create ${set} hash:net family ${family} maxelem ${Math.max(65536, list.length * 2)} -exist`, ...list.map(p => `add ${set} ${p} -exist`)]).join('\n')
    }
    case 'mikrotik': {
      const v4 = prefixes.filter(p => !v6(p))
      const v6list = prefixes.filter(v6)
      return [
        ...(v4.length ? ['/ip firewall address-list', ...v4.map(p => `add list=${name} address=${p}`)] : []),
        ...(v6list.length ? ['/ipv6 firewall address-list', ...v6list.map(p => `add list=${name} address=${p}`)] : []),
      ].join('\n')
    }
  }
}
