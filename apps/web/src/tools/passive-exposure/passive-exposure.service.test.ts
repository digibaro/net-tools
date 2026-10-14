import { describe, expect, it } from 'vitest'
import { cveUrl } from './passive-exposure.service'

describe('cveUrl', () => {
  it('links NVD', () => {
    expect(cveUrl('CVE-2021-41617')).toBe('https://nvd.nist.gov/vuln/detail/CVE-2021-41617')
  })
})
