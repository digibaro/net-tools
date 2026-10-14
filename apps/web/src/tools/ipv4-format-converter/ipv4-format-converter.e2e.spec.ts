import { expect, test } from '@playwright/test'

test('ipv4-format-converter: decodes short inet_aton forms', async ({ page }) => {
  await page.goto('/ipv4-format-converter?ip=0x7f.1')
  await expect(page.getByTestId('result').getByTestId('kv-Dotted decimal')).toHaveText('127.0.0.1')
  await expect(page.getByTestId('lenient-note')).toBeVisible()
})
