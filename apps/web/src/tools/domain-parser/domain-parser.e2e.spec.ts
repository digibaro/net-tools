import { expect, test } from '@playwright/test'

test('domain-parser: registrable domain from a URL', async ({ page }) => {
  await page.goto('/domain-parser?q=www.example.co.uk')
  await expect(page.getByTestId('result').getByTestId('kv-Registrable domain')).toHaveText('example.co.uk')
  await expect(page.getByTestId('result').getByTestId('kv-Public suffix')).toHaveText('co.uk')
})
