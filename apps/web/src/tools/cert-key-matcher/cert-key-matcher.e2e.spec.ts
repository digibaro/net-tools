import { expect, test } from '@playwright/test'

test('cert-key-matcher: matches a generated key and CSR', async ({ page }) => {
  await page.goto('/csr-tool?tab=generate')
  await page.getByTestId('subject-cn').locator('input').fill('match.example.com')
  await page.getByTestId('generate').click()
  const csr = await page.getByTestId('generated-csr').locator('pre').textContent()
  const key = await page.getByTestId('generated-key').locator('pre').textContent()
  await page.goto('/cert-key-matcher')
  await page.getByTestId('key').locator('textarea').fill(key!)
  await page.getByTestId('csr').locator('textarea').fill(csr!)
  await expect(page.getByTestId('verdict')).toHaveText(/match/i)
})
