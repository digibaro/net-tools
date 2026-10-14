import { expect, test } from '@playwright/test'

test('url-parser: parses and encodes', async ({ page }) => {
  await page.goto(`/url-parser?url=${encodeURIComponent('https://example.com:8080/p?a=1&b=x%20y')}`)
  await expect(page.getByTestId('result').getByTestId('kv-Hostname')).toHaveText('example.com')
  await expect(page.getByText('x y')).toBeVisible()
  await page.getByTestId('plain').fill('a b')
  await expect(page.getByTestId('encoded')).toHaveValue('a%20b')
})
