import { Buffer } from 'node:buffer'
import { expect, test } from '@playwright/test'

const XML = `<?xml version="1.0"?><feedback><report_metadata><org_name>google.com</org_name><report_id>1</report_id><date_range><begin>1791936000</begin><end>1792022399</end></date_range></report_metadata>
<policy_published><domain>example.com</domain><p>reject</p></policy_published>
<record><row><source_ip>192.0.2.10</source_ip><count>9</count><policy_evaluated><disposition>none</disposition><dkim>pass</dkim><spf>pass</spf></policy_evaluated></row><identifiers><header_from>example.com</header_from></identifiers><auth_results><dkim><domain>example.com</domain><result>pass</result></dkim></auth_results></record>
<record><row><source_ip>198.51.100.7</source_ip><count>1</count><policy_evaluated><disposition>reject</disposition><dkim>fail</dkim><spf>fail</spf></policy_evaluated></row><identifiers><header_from>example.com</header_from></identifiers><auth_results><spf><domain>bad.example</domain><result>fail</result></spf></auth_results></record></feedback>`

test('dmarc-report-viewer: uploads an XML report', async ({ page }) => {
  await page.goto('/dmarc-report-viewer')
  await page.getByTestId('file').setInputFiles({ name: 'report.xml', mimeType: 'application/xml', buffer: Buffer.from(XML) })
  const result = page.getByTestId('result')
  await expect(result).toContainText('example.com')
  await expect(result).toContainText('9 (90.0 %)')
  await expect(page.getByText('198.51.100.7')).toBeVisible()
})
