import { expect, test } from '@playwright/test'

test('tcp-ping: metadata address is blocked', async ({ page }) => {
  await page.goto('/tcp-ping?target=169.254.169.254&port=80')
  await expect(page.getByTestId('api-error')).toContainText('TARGET_BLOCKED')
})
