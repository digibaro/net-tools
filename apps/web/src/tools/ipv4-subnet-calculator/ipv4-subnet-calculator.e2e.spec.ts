import { expect, test } from '@playwright/test'

test('ipv4-subnet-calculator: computes from the URL and validates input', async ({ page }) => {
  await page.goto('/ipv4-subnet-calculator?ip=10.20.30.40/20')
  const result = page.getByTestId('result')
  await expect(result.getByTestId('kv-Network')).toHaveText('10.20.16.0/20')
  await expect(result.getByTestId('kv-Broadcast')).toHaveText('10.20.31.255')
  await expect(result.getByTestId('kv-Usable hosts')).toHaveText('4,094')

  await page.getByTestId('input').fill('10.0.0.1/33')
  await expect(page.getByText('Enter an IPv4 address')).toBeVisible()
  await expect(result).toBeHidden()
})
