import { expect, test } from '@playwright/test'

test('wireguard-config: generates server and peer configs', async ({ page }) => {
  await page.goto('/wireguard-config?peers=3')
  await expect(page.getByTestId('server-config')).toContainText('ListenPort = 51820')
  await expect(page.getByTestId('peer')).toHaveCount(3)
  await expect(page.getByTestId('peer').first().locator('img')).toBeVisible()
})
