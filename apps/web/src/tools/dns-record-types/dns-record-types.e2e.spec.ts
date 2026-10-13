import { expect, test } from '@playwright/test'

test('dns-record-types: finds HTTPS with an example', async ({ page }) => {
  await page.goto('/dns-record-types?q=https')
  await expect(page.getByTestId('result').getByText(/alpn="h3,h2"/)).toBeVisible()
})
