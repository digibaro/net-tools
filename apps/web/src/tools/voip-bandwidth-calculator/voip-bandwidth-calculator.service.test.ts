import { describe, expect, it } from 'vitest'
import { channelsFor, erlangB, erlangs, perCall } from './voip-bandwidth-calculator.service'

describe('voip-bandwidth-calculator', () => {
  it('computes the classic per-call figures', () => {
    // G.711, 20 ms, IPv4 over Ethernet without preamble/IFG: 87.2 kbit/s; G.729: 31.2 kbit/s
    expect(perCall({ codec: 'G.711', ptimeMs: 20, ipv6: false, srtp: false, vlan: false, link: 'ethernet-no-phy' }).kbps).toBeCloseTo(87.2, 5)
    expect(perCall({ codec: 'G.729', ptimeMs: 20, ipv6: false, srtp: false, vlan: false, link: 'ethernet-no-phy' }).kbps).toBeCloseTo(31.2, 5)
    expect(perCall({ codec: 'G.711', ptimeMs: 20, ipv6: false, srtp: false, vlan: false, link: 'none' }).kbps).toBeCloseTo(80, 5)
    expect(perCall({ codec: 'G.711', ptimeMs: 30, ipv6: true, srtp: true, vlan: false, link: 'none' }).pps).toBeCloseTo(33.33, 2)
  })
  it('sizes trunks with Erlang B', () => {
    expect(erlangB(10, 10)).toBeCloseTo(0.2146, 4)
    expect(channelsFor(10, 0.01)).toBe(18)
    expect(erlangs(120, 3)).toBe(6)
  })
})
