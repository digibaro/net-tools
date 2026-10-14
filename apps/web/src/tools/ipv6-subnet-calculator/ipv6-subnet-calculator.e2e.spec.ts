import { expect, test } from '@playwright/test'

test('ipv6-subnet-calculator: computes from the URL', async ({ page }) => {
  await page.goto('/ipv6-subnet-calculator?ip=2001:db8:1::/56')
  const result = page.getByTestId('result')
  await expect(result.getByTestId('kv-Network')).toHaveText('2001:db8:1::/56')
  await expect(result.getByTestId('kv-/64 subnets')).toHaveText('256')
  await page.getByTestId('input').fill('10.0.0.0/8')
  await expect(result).toBeHidden()
})
