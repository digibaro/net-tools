import { expect, test } from '@playwright/test'
import { LEAF_PEM } from './testdata/certs'

test('certificate-decoder: decodes a pasted PEM', async ({ page }) => {
  await page.goto('/certificate-decoder')
  await page.getByTestId('input').fill(LEAF_PEM)
  const result = page.getByTestId('result')
  await expect(result.getByTestId('kv-Subject')).toHaveText('CN=www.example.test')
  await expect(result.getByTestId('kv-SHA-256 fingerprint')).toHaveText(/^BE:29:C5:4F/)
})
