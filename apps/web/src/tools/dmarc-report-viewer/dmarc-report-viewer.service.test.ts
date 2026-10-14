import type { ReportError } from './dmarc-report-viewer.service'
// @vitest-environment jsdom
import { gzipSync, strToU8, zipSync } from 'fflate'
import { describe, expect, it } from 'vitest'
import { MAX_ZIP_ENTRIES, parseReportXml, readReports, summarize } from './dmarc-report-viewer.service'

const XML = `<?xml version="1.0" encoding="UTF-8" ?>
<feedback>
  <report_metadata>
    <org_name>google.com</org_name>
    <email>noreply-dmarc-support@google.com</email>
    <report_id>1234567890123456789</report_id>
    <date_range><begin>1791936000</begin><end>1792022399</end></date_range>
  </report_metadata>
  <policy_published><domain>example.com</domain><adkim>r</adkim><aspf>r</aspf><p>quarantine</p><sp>quarantine</sp><pct>100</pct></policy_published>
  <record>
    <row><source_ip>192.0.2.10</source_ip><count>40</count><policy_evaluated><disposition>none</disposition><dkim>pass</dkim><spf>pass</spf></policy_evaluated></row>
    <identifiers><header_from>example.com</header_from></identifiers>
    <auth_results><dkim><domain>example.com</domain><selector>s1</selector><result>pass</result></dkim><spf><domain>example.com</domain><result>pass</result></spf></auth_results>
  </record>
  <record>
    <row><source_ip>198.51.100.7</source_ip><count>3</count><policy_evaluated><disposition>quarantine</disposition><dkim>fail</dkim><spf>fail</spf><reason><type>forwarded</type><comment>list</comment></reason></policy_evaluated></row>
    <identifiers><header_from>example.com</header_from></identifiers>
    <auth_results><spf><domain>spammer.example</domain><result>softfail</result></spf></auth_results>
  </record>
  <record>
    <row><source_ip>192.0.2.10</source_ip><count>2</count><policy_evaluated><disposition>none</disposition><dkim>fail</dkim><spf>pass</spf></policy_evaluated></row>
    <identifiers><header_from>example.com</header_from></identifiers>
    <auth_results><spf><domain>example.com</domain><result>pass</result></spf></auth_results>
  </record>
</feedback>`

describe('dmarc-report-viewer', () => {
  it('parses a report', () => {
    const r = parseReportXml(XML)
    expect(r.orgName).toBe('google.com')
    expect(r.policy).toMatchObject({ domain: 'example.com', p: 'quarantine', pct: '100' })
    expect(r.begin?.toISOString()).toBe('2026-10-14T00:00:00.000Z')
    expect(r.records).toHaveLength(3)
    expect(r.records[1]).toMatchObject({ disposition: 'quarantine', pass: false, reasons: ['forwarded: list'] })
    expect(r.records[0]!.dkimAuth[0]).toEqual({ domain: 'example.com', result: 'pass', selector: 's1', scope: undefined })
  })
  it('summarizes by source, failures first', () => {
    const s = summarize([parseReportXml(XML)])
    expect(s.messages).toBe(45)
    expect(s.pass).toBe(42)
    expect(s.dispositions).toEqual({ none: 42, quarantine: 3 })
    expect(s.sources[0]!.sourceIp).toBe('198.51.100.7')
    expect(s.sources[1]).toMatchObject({ sourceIp: '192.0.2.10', count: 42, pass: 42 })
  })
  it('reads xml, gzip and zip', () => {
    expect(readReports('a.xml', strToU8(XML))).toHaveLength(1)
    expect(readReports('a.xml.gz', gzipSync(strToU8(XML)))[0]!.records).toHaveLength(3)
    const zip = zipSync({ 'google.com!example.com!1!2.xml': strToU8(XML), 'readme.txt': strToU8('x') })
    const rs = readReports('a.zip', zip)
    expect(rs).toHaveLength(1)
    expect(rs[0]!.file).toBe('a.zip › google.com!example.com!1!2.xml')
  })
  it('refuses DTDs, non-reports and bombs', () => {
    const fail = (fn: () => unknown) => {
      try {
        fn()
      }
      catch (e) {
        return (e as ReportError).key
      }
      return 'ok'
    }
    expect(fail(() => parseReportXml('<?xml version="1.0"?><!DOCTYPE x [<!ENTITY a "aaaa">]><feedback/>'))).toBe('doctype')
    expect(fail(() => parseReportXml('<html/>'))).toBe('notDmarc')
    expect(fail(() => readReports('x.bin', new Uint8Array([1, 2, 3])))).toBe('unsupported')
    const big = gzipSync(new Uint8Array(21 * 1024 * 1024))
    expect(fail(() => readReports('bomb.gz', big))).toBe('tooLarge')
    const many: Record<string, Uint8Array> = {}
    for (let i = 0; i <= MAX_ZIP_ENTRIES; i++) many[`${i}.xml`] = strToU8(XML)
    expect(fail(() => readReports('many.zip', zipSync(many)))).toBe('tooMany')
  })
})
