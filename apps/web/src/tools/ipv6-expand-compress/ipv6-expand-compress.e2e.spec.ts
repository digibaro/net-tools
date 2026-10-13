import { expect, test } from '@playwright/test'

test('ipv6-expand-compress: canonicalizes from the URL', async ({ page }) => {
  await page.goto('/ipv6-expand-compress?ip=FE80:0:0:0:0:0:0:1')
  await expect(page.getByTestId('result').getByTestId('kv-Compressed (RFC 5952)')).toHaveText('fe80::1')
})
