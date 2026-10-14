import { expect, test } from '@playwright/test'

test('http-request: cannot reach the API itself through the tool (SSRF)', async ({ page }) => {
  await page.goto(`/http-request?url=${encodeURIComponent('http://127.0.0.1:8081/api/healthz')}`)
  await page.getByTestId('run').click()
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})
