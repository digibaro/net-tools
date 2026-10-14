import { expect, test } from '@playwright/test'

test('redirect-tracer: metadata URL is blocked', async ({ page }) => {
  await page.goto(`/redirect-tracer?url=${encodeURIComponent('http://169.254.169.254/latest/meta-data/')}`)
  await expect(page.getByTestId('api-error')).toContainText('TARGET_BLOCKED')
})
