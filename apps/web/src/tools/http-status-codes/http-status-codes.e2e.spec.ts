import { expect, test } from '@playwright/test'

test('http-status-codes: searches by code', async ({ page }) => {
  await page.goto('/http-status-codes?q=429')
  const result = page.getByTestId('result')
  await expect(result.getByText('Too Many Requests')).toBeVisible()
  await expect(result.getByRole('link', { name: 'RFC 6585 §4' })).toHaveAttribute('href', 'https://www.rfc-editor.org/rfc/rfc6585#section-4')
})
