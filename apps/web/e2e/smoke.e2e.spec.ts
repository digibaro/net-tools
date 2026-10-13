import { expect, test } from '@playwright/test'

test('SPA loads and shows the API version', async ({ page }) => {
  const versionResponse = page.waitForResponse(r => r.url().endsWith('/api/v1/version'))
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'NetTools' })).toBeVisible()
  expect((await versionResponse).status()).toBe(200)
  await expect(page.getByTestId('api-status')).toHaveText(/^API \S+ \(\S+\)$/)
})

test('unknown routes fall back to the SPA', async ({ page }) => {
  const res = await page.goto('/some/deep/link')
  expect(res?.status()).toBe(200)
  await expect(page.locator('#app')).toBeAttached()
})

test('API health is reachable through the proxy', async ({ request }) => {
  const res = await request.get('/api/healthz')
  expect(res.ok()).toBeTruthy()
  expect(await res.json()).toEqual({ status: 'ok' })
  expect(res.headers()['x-request-id']).toBeTruthy()
})

test('home page loads without console errors (e.g. CSP violations)', async ({ page }) => {
  const errors: string[] = []
  page.on('console', (msg) => {
    if (msg.type() === 'error')
      errors.push(msg.text())
  })
  page.on('pageerror', err => errors.push(err.message))
  await page.goto('/')
  await expect(page.getByTestId('api-status')).toHaveText(/^API /)
  expect(errors).toEqual([])
})
