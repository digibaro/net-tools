import { expect, test } from '@playwright/test'

test('lab-ca-generator: generates a CA and a server certificate', async ({ page }) => {
  await page.goto('/lab-ca-generator?cn=nas.home.arpa&san=192.168.1.10')
  await page.getByTestId('run').click()
  await expect(page.getByTestId('result')).toContainText('BEGIN CERTIFICATE')
  await expect(page.getByTestId('result')).toContainText('BEGIN PRIVATE KEY')
})
