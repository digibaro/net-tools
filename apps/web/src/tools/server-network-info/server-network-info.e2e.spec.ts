import { expect, test } from '@playwright/test'

test('server-network-info: disabled in the public profile', async ({ page }) => {
  await page.goto('/server-network-info')
  await expect(page.getByTestId('tool-disabled')).toContainText('Disabled by the administrator')
})
