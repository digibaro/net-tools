import { expect, test } from '@playwright/test'

test('ipv6-ula-generator: shows a shared prefix and generates new ones', async ({ page }) => {
  await page.goto('/ipv6-ula-generator?gid=123456789a')
  const prefix = page.getByTestId('result').getByTestId('kv-ULA prefix')
  await expect(prefix).toHaveText('fd12:3456:789a::/48')
  await page.getByTestId('generate').click()
  await expect(prefix).toHaveText(/^fd[0-9a-f]{2}:[0-9a-f]{1,4}:[0-9a-f]{1,4}::\/48$/)
  await expect(page).toHaveURL(/gid=[0-9a-f]{10}/)
})
