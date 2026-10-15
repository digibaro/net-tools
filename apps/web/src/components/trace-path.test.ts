import type { TraceHop } from '@/utils/trace'
import { describe, expect, it } from 'vitest'
import { asPath, hopIps } from './trace-path'

const h = (hop: number, ip: string, asn?: number) => ({ hop, ip, asn: asn ?? null }) as unknown as TraceHop

describe('trace-path', () => {
  it('collapses the AS path', () => {
    expect(asPath([h(3, 'c', 1299), h(1, 'a'), h(2, 'b', 1103), h(4, 'd', 1299), h(5, 'e', 13335)])).toEqual([1103, 1299, 13335])
    expect(hopIps([h(2, '2.2.2.2'), h(1, '1.1.1.1'), h(3, '')])).toEqual(['1.1.1.1', '2.2.2.2'])
  })
})
