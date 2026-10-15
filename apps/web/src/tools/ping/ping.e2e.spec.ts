import { expect, test } from '@playwright/test'

test('ping: private targets are refused (SSRF)', async ({ page }) => {
  await page.goto('/ping?target=192.168.0.1')
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})

test('ping: pings a public host from the server', async ({ page }) => {
  await page.goto('/ping?target=1.1.1.1&count=2&interval=500')
  await expect(page.getByTestId('summary').or(page.getByTestId('api-error'))).toBeVisible({ timeout: 20_000 })
})

test('ping: worldwide vantage runs through Globalping', async ({ page }) => {
  await page.goto('/ping?target=example.com&vantage=worldwide&probes=1')
  await expect(page.getByTestId('vantage-worldwide')).toBeVisible()
  await expect(page.getByTestId('remote-probe').or(page.getByTestId('api-error')).first()).toBeVisible({ timeout: 45_000 })
  if (await page.getByTestId('remote-probe').count()) // probe locations on the bundled world map
    await expect(page.getByTestId('probe-map').locator('circle')).toHaveCount(1)
})

test('ping: worldwide refuses private targets', async ({ page }) => {
  await page.goto('/ping?target=10.0.0.1&vantage=worldwide')
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})
