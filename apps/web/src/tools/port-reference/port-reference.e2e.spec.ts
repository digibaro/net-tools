import { expect, test } from '@playwright/test'

test('port-reference: looks up a port', async ({ page }) => {
  await page.goto('/port-reference?q=22&proto=tcp')
  await expect(page.getByTestId('result').getByText('The Secure Shell (SSH) Protocol')).toBeVisible()
})
