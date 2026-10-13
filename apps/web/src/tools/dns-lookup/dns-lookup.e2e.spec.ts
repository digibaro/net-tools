import { expect, test } from '@playwright/test'

test('dns-lookup: rejects private resolvers (SSRF) with a readable error', async ({ page }) => {
  await page.goto('/dns-lookup?name=example.com&resolver=custom&custom=127.0.0.1')
  await expect(page.getByTestId('api-error')).toContainText('TARGET_PRIVATE_DENIED')
})

test('dns-lookup: browser DoH mode (mocked provider)', async ({ page }) => {
  await page.route('https://cloudflare-dns.com/**', route => route.fulfill({ json: { Status: 0, RA: true, Answer: [{ name: 'example.com', type: 1, TTL: 60, data: '192.0.2.44' }] } }))
  await page.goto('/dns-lookup?name=example.com&from=browser')
  await expect(page.getByTestId('result').getByText('192.0.2.44')).toBeVisible()
  await expect(page.getByTestId('rcode')).toHaveText('NOERROR')
})
