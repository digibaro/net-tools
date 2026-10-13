import { expect, test } from '@playwright/test'

test('what-is-my-ip: shows the server view and the (mocked) public view', async ({ page }) => {
  await page.route('https://api.ipify.org/**', route => route.fulfill({ json: { ip: '203.0.113.50' } }))
  await page.route('https://api6.ipify.org/**', route => route.abort())
  await page.goto('/what-is-my-ip')
  await expect(page.getByTestId('public').getByTestId('kv-IPv4')).toHaveText('203.0.113.50')
  await expect(page.getByTestId('public').getByTestId('kv-IPv6')).toHaveText('Not available')
  await expect(page.getByTestId('server').getByTestId('kv-Address')).toHaveText(/^[\d.:a-f]+$/)
})
