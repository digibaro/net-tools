import { expect, test } from '@playwright/test'

test('as-path-visualizer: draws shared upstreams and flags prepends', async ({ page }) => {
  await page.goto('/as-path-visualizer?paths=3356%201299%2013335%0A174%2013335%2013335')
  const graph = page.getByTestId('graph')
  await expect(graph.getByText('AS13335')).toHaveCount(1)
  await expect(graph.getByText('×2')).toBeVisible()
  await expect(page.getByText(/prepends itself 2/)).toBeVisible()
})
