import { expect, test } from '@playwright/test'

test('home renders the shell', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByTestId('sidebar')).toBeAttached()
  await expect(page.getByTestId('search-open')).toBeVisible()
})

test('about page shows the API version via the proxy', async ({ page }) => {
  const versionResponse = page.waitForResponse(r => r.url().endsWith('/api/v1/version'))
  await page.goto('/about')
  expect((await versionResponse).status()).toBe(200)
  await expect(page.getByRole('heading', { name: 'About', level: 1 })).toBeVisible()
  await expect(page.getByTestId('kv-API version')).toHaveText(/^\S+ \(\S+\)$/)
})

test('unknown routes show the 404 page', async ({ page }) => {
  const res = await page.goto('/some/deep/link')
  expect(res?.status()).toBe(200)
  await expect(page.getByTestId('not-found')).toBeVisible()
})

test('API health is reachable through the proxy', async ({ request }) => {
  const res = await request.get('/api/healthz')
  expect(res.ok()).toBeTruthy()
  expect(await res.json()).toEqual({ status: 'ok' })
  expect(res.headers()['x-request-id']).toBeTruthy()
})

for (const path of ['/', '/about', '/settings']) {
  test(`${path} loads without console errors (e.g. CSP violations)`, async ({ page }) => {
    const errors: string[] = []
    page.on('console', (msg) => {
      if (msg.type() === 'error')
        errors.push(msg.text())
    })
    page.on('pageerror', err => errors.push(err.message))
    await page.goto(path)
    await page.waitForLoadState('networkidle')
    expect(errors).toEqual([])
  })
}

test('CSP allows exactly the inline theme script', async ({ request }) => {
  const res = await request.get('/')
  expect(res.headers()['content-security-policy']).toMatch(/script-src 'self' 'sha256-[\w+/=]+';/)
})
