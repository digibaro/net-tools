import { join } from 'node:path'
import { expect, test } from '@playwright/test'

test('pkcs12-inspector: opens a bundle in the browser', async ({ page }) => {
  await page.goto('/pkcs12-inspector')
  await page.getByTestId('file').setInputFiles(join(import.meta.dirname, 'testdata', 'ec.p12'))
  await page.getByTestId('password').locator('input').fill('secret')
  await page.getByTestId('run').click()
  await expect(page.getByTestId('result')).toContainText('SHA-256')
  await expect(page.getByText('CN=ec.example.test')).toBeVisible()
})
