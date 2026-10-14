import type { DnsblCheck } from './blacklist-check.service'
import { describe, expect, it } from 'vitest'
import { sortChecks } from './blacklist-check.service'

const c = (name: string, status: string) => ({ name, status }) as DnsblCheck

describe('sortChecks', () => {
  it('puts listings first', () => {
    expect(sortChecks([c('B', 'clean'), c('Z', 'listed'), c('A', 'error'), c('A', 'clean')]).map(x => x.name + x.status)).toEqual(['Zlisted', 'Aerror', 'Aclean', 'Bclean'])
  })
})
