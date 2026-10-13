import { expect, test } from '@playwright/test'

test('csr-tool: generates a CSR and decodes it', async ({ page }) => {
  await page.goto('/csr-tool?tab=generate')
  await page.getByTestId('subject-cn').locator('input').fill('www.example.com')
  await page.getByTestId('generate').click()
  const pem = await page.getByTestId('generated-csr').locator('pre').textContent()
  expect(pem).toContain('BEGIN CERTIFICATE REQUEST')
  await expect(page.getByTestId('generated-key')).toContainText('BEGIN PRIVATE KEY')
  await page.goto('/csr-tool?tab=decode')
  await page.getByTestId('csr').locator('textarea').fill(pem!)
  await expect(page.getByTestId('result')).toContainText('CN=www.example.com')
})
