import { expect, test } from '@playwright/test'

test('wildcard-mask-calculator: netmask to wildcard with ACL example', async ({ page }) => {
  await page.goto('/wildcard-mask-calculator?mask=255.255.252.0&net=10.1.7.9')
  const result = page.getByTestId('result')
  await expect(result.getByTestId('kv-Wildcard mask')).toHaveText('0.0.3.255')
  await expect(result.getByTestId('kv-ACL example')).toHaveText('access-list 10 permit 10.1.4.0 0.0.3.255')
})
